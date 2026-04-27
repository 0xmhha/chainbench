package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)

// Handler is the common signature for all command handlers.
// args is the raw JSON of cmd.args from the wire envelope; bus is available
// for mid-flight progress/events. Returns (data, nil) on success — data is
// wrapped into EmitResult(true, data). Returns (_, *APIError) for typed
// failures; any other error is treated as INTERNAL by the dispatcher.
type Handler func(args json.RawMessage, bus *events.Bus) (map[string]any, error)

// allHandlers builds the command → handler dispatch table. stateDir and
// chainbenchDir are bound via closure into handlers that need them.
//
// Handler implementations live in sibling files in this package:
//
//	handlers_network.go         — network.load / network.probe / network.attach
//	handlers_node_lifecycle.go  — node.stop / node.start / node.restart / node.tail_log
//	handlers_node_read.go       — node.block_number / node.chain_id / node.balance / node.gas_price
//	handlers_node_tx.go         — node.tx_send / node.tx_wait (newHandleNodeTxWait)
//	                              + nonce / gas / gas-price resolvers + receipt helpers
//
// This file keeps the dispatch table plus shared node-resolution helpers.
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
	return map[string]Handler{
		"network.load":      newHandleNetworkLoad(stateDir),
		"network.probe":     newHandleNetworkProbe(),
		"network.attach":    newHandleNetworkAttach(stateDir),
		"node.stop":         newHandleNodeStop(stateDir, chainbenchDir),
		"node.start":        newHandleNodeStart(stateDir, chainbenchDir),
		"node.restart":      newHandleNodeRestart(stateDir, chainbenchDir),
		"node.tail_log":     newHandleNodeTailLog(stateDir),
		"node.block_number": newHandleNodeBlockNumber(stateDir),
		"node.chain_id":     newHandleNodeChainID(stateDir),
		"node.balance":      newHandleNodeBalance(stateDir),
		"node.gas_price":    newHandleNodeGasPrice(stateDir),
		"node.tx_send":      newHandleNodeTxSend(stateDir),
		"node.tx_wait":      newHandleNodeTxWait(stateDir),
	}
}

// resolveNodeIDFromString validates nodeID's "node" prefix and confirms it
// exists in the active network's pids.json. Core resolution used by handlers
// that already parsed their args independently. Returns APIError sentinels:
//
//	INVALID_ARGS  — empty or malformed node_id, node not found
//	UPSTREAM_ERROR — state.LoadActive failure (pids.json missing, parse error)
func resolveNodeIDFromString(stateDir, nodeID string) (string, string, error) {
	// Local-only convention: node<N> where N is the numeric pids.json key.
	// chainbench.sh takes the raw numeric suffix as its arg.
	if nodeID == "" {
		return "", "", NewInvalidArgs("args.node_id is required")
	}
	if !strings.HasPrefix(nodeID, "node") {
		return "", "", NewInvalidArgs(fmt.Sprintf(`node_id must start with "node" prefix (got %q)`, nodeID))
	}
	num := strings.TrimPrefix(nodeID, "node")
	if num == "" {
		return "", "", NewInvalidArgs("node_id missing numeric suffix")
	}
	// Delegate existence check through resolveNode (M4 absorption — single
	// state.LoadActive call site). resolveNode does the node-lookup + error
	// mapping; we only need the numeric suffix the caller passes back to the
	// local chainbench.sh CLI.
	if _, _, err := resolveNode(stateDir, "local", nodeID); err != nil {
		return "", "", err
	}
	return nodeID, num, nil
}

// resolveNodeID parses the command envelope args into {node_id} and delegates
// to resolveNodeIDFromString. Used by handlers that receive the full wire
// envelope args payload.
func resolveNodeID(stateDir string, args json.RawMessage) (string, string, error) {
	var req struct {
		NodeID string `json:"node_id"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &req); err != nil {
			return "", "", NewInvalidArgs(fmt.Sprintf("args: %v", err))
		}
	}
	return resolveNodeIDFromString(stateDir, req.NodeID)
}

// resolveNode resolves (networkName, nodeID) to the resolved (canonical) name
// and a copy of the node record. networkName=="" defaults to "local"; non-local
// names are pattern-checked at the handler boundary. Returns APIError sentinels:
//
//	INVALID_ARGS   — empty/malformed node_id, bad network name pattern, or
//	                 node not in resolved network
//	UPSTREAM_ERROR — state.LoadActive failure (missing pids.json or
//	                 networks/<name>.json)
//
// The node is returned by value (Node is ~80 bytes) so callers do not alias
// the state-layer's slice backing array. Distinct from resolveNodeID, which
// is local-only and enforces the "node<N>" numeric-suffix convention for
// chainbench.sh arg shape. This helper is network-aware and node-id agnostic.
func resolveNode(stateDir, networkName, nodeID string) (string, types.Node, error) {
	var zero types.Node
	if nodeID == "" {
		return "", zero, NewInvalidArgs("args.node_id is required")
	}
	if networkName == "" {
		networkName = "local"
	}
	if networkName != "local" && !state.IsValidRemoteName(networkName) {
		return "", zero, NewInvalidArgs(fmt.Sprintf(
			"args.network must be 'local' or match [a-z0-9][a-z0-9_-]*: %q", networkName,
		))
	}
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: networkName})
	if err != nil {
		return "", zero, NewUpstream("failed to load network state", err)
	}
	for _, n := range net.Nodes {
		if n.Id == nodeID {
			return net.Name, n, nil
		}
	}
	return "", zero, NewInvalidArgs(fmt.Sprintf(
		"node_id %q not found in network %q", nodeID, networkName,
	))
}

// dialNode opens a remote.Client for the given node, wiring auth if
// node.Auth is populated. Caller owns Close. Errors are returned as
// APIError sentinels so handlers can propagate them unchanged:
//
//	UPSTREAM_ERROR — auth setup failure (missing env var, malformed type)
//	                 or dial failure (DNS, TCP, TLS, bad URL).
//
// Shared by every read-only remote command (node.block_number,
// node.chain_id, node.balance, node.gas_price) so the dial+auth
// boilerplate lives in one place.
func dialNode(ctx context.Context, node *types.Node) (*remote.Client, error) {
	var rt http.RoundTripper
	if len(node.Auth) > 0 {
		got, aerr := remote.AuthFromNode(node, os.Getenv)
		if aerr != nil {
			return nil, NewUpstream("auth setup", aerr)
		}
		rt = got
	}
	client, err := remote.DialWithOptions(ctx, node.Http, remote.DialOptions{Transport: rt})
	if err != nil {
		return nil, NewUpstream(fmt.Sprintf("dial %s", node.Http), err)
	}
	return client, nil
}
