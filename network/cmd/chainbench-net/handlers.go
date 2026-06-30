package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/drivers/sshremote"
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
//	handlers_node_read.go       — node.block_number / node.chain_id / node.balance /
//	                              node.gas_price / node.contract_call (eth_call wrapper,
//	                              calldata or ABI mode) / node.events_get
//	                              (eth_getLogs wrapper, optional ABI log decode) /
//	                              node.account_state (composite balance/nonce/code/
//	                              storage subset reader, fields-selectable) /
//	                              node.rpc (generic JSON-RPC passthrough)
//	handlers_node_tx.go         — node.tx_send (incl. EIP-7702 SetCode path) /
//	                              node.contract_deploy (legacy + 1559; SetCode N/A) /
//	                              node.tx_fee_delegation_send (go-stablenet 0x16) /
//	                              node.tx_wait + nonce / gas / gas-price resolvers
//	                              + receipt helpers
//
// This file keeps the dispatch table plus shared node-resolution helpers.
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
	return map[string]Handler{
		"network.load":                newHandleNetworkLoad(stateDir),
		"network.probe":               newHandleNetworkProbe(),
		"network.init":                newHandleNetworkInit(stateDir, chainbenchDir),
		"network.start_all":           newHandleNetworkStartAll(stateDir, chainbenchDir),
		"network.restart":             newHandleNetworkRestart(stateDir, chainbenchDir),
		"network.clean":               newHandleNetworkClean(stateDir, chainbenchDir),
		"network.stop_all":            newHandleNetworkStopAll(stateDir, chainbenchDir),
		"network.status":              newHandleNetworkStatus(stateDir, chainbenchDir),
		"network.attach":              newHandleNetworkAttach(stateDir),
		"network.detach":              newHandleNetworkDetach(stateDir),
		"network.list":                newHandleNetworkList(stateDir),
		"network.capabilities":        newHandleNetworkCapabilities(stateDir),
		"node.stop":                   newHandleNodeStop(stateDir, chainbenchDir),
		"node.start":                  newHandleNodeStart(stateDir, chainbenchDir),
		"node.restart":                newHandleNodeRestart(stateDir, chainbenchDir),
		"node.rpc":                    newHandleNodeRpc(stateDir),
		"node.tail_log":               newHandleNodeTailLog(stateDir),
		"node.account_state":          newHandleNodeAccountState(stateDir),
		"node.block_number":           newHandleNodeBlockNumber(stateDir),
		"node.chain_id":               newHandleNodeChainID(stateDir),
		"node.balance":                newHandleNodeBalance(stateDir),
		"node.contract_call":          newHandleNodeContractCall(stateDir),
		"node.contract_deploy":        newHandleNodeContractDeploy(stateDir),
		"node.events_get":             newHandleNodeEventsGet(stateDir),
		"node.gas_price":              newHandleNodeGasPrice(stateDir),
		"node.tx_send":                newHandleNodeTxSend(stateDir),
		"node.tx_fee_delegation_send": newHandleNodeTxFeeDelegationSend(stateDir),
		"node.tx_wait":                newHandleNodeTxWait(stateDir),
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

// dialNode opens a remote.Client for the given node, dispatching on the node's
// provider. Caller owns Close (which also tears down an SSH tunnel, if any).
// Errors are returned as APIError sentinels so handlers propagate them unchanged:
//
//	INVALID_ARGS   — ssh-remote node with malformed/incomplete ssh-password auth.
//	UPSTREAM_ERROR — auth setup (missing env var), SSH dial, or RPC dial failure.
//
// Shared by every read-only command so the dial+auth boilerplate lives in one
// place. The "remote" (HTTP/WS + api-key/jwt) and "ssh-remote" (SSH-tunneled,
// Sprint 5b.1) providers route here; "local" nodes never reach dialNode.
func dialNode(ctx context.Context, node *types.Node) (*remote.Client, error) {
	switch node.Provider {
	case types.NodeProviderSshRemote:
		return dialSSHNode(ctx, node)
	case types.NodeProviderRemote, "":
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
	default:
		return nil, NewNotSupported(fmt.Sprintf("provider %q is not dialable for RPC", node.Provider))
	}
}

// sshCredsFromNode parses an ssh-remote node's ssh-password auth into SSH
// credentials. The auth supplies user/host/port and the *name* of the env var
// holding the password; the password value is read here (os.Getenv) and never
// stored in state or logged. Shared by dialSSHNode (5b.1 tunnel) and the
// process/fs handlers (5b.2 Exec).
//
//	INVALID_ARGS   — auth type not ssh-password, or user/host/env missing.
//	UPSTREAM_ERROR — the named env var is empty.
func sshCredsFromNode(node *types.Node) (sshremote.Credentials, error) {
	var zero sshremote.Credentials
	if rawType, _ := node.Auth["type"].(string); rawType != "ssh-password" {
		return zero, NewInvalidArgs(fmt.Sprintf(
			"ssh-remote node requires ssh-password auth, got %q", rawType))
	}
	user, _ := node.Auth["user"].(string)
	host, _ := node.Auth["host"].(string)
	envName, _ := node.Auth["env"].(string)
	if user == "" || host == "" || envName == "" {
		return zero, NewInvalidArgs("ssh-password auth requires user, host, and env")
	}
	password := os.Getenv(envName)
	if password == "" {
		// Name the env var, never a value.
		return zero, NewUpstream("ssh auth", fmt.Errorf("env var %q is empty", envName))
	}
	return sshremote.Credentials{
		User:     user,
		Host:     host,
		Port:     authPort(node.Auth),
		Password: password,
	}, nil
}

// dialSSHNode builds an SSH-tunneled remote.Client from an ssh-remote node.
// Host key verification follows the security policy in
// sshremote.ResolveHostKeyCallback.
func dialSSHNode(ctx context.Context, node *types.Node) (*remote.Client, error) {
	creds, err := sshCredsFromNode(node)
	if err != nil {
		return nil, err
	}
	hostKey, err := sshremote.ResolveHostKeyCallback(os.Getenv)
	if err != nil {
		return nil, NewUpstream("ssh host key", err)
	}
	client, err := sshremote.Dial(ctx, creds, node.Http, hostKey)
	if err != nil {
		return nil, NewUpstream(fmt.Sprintf("ssh dial %s", node.Http), err)
	}
	return client, nil
}

// execSSHNode runs a single command on an ssh-remote node's host and returns
// the result. Centralizes creds + host-key resolution for the process/fs
// handlers (Sprint 5b.2). Errors are classified APIErrors; a non-zero command
// exit is returned in the ExecResult (ExitCode), not as an error.
func execSSHNode(ctx context.Context, node *types.Node, command string) (sshremote.ExecResult, error) {
	creds, err := sshCredsFromNode(node)
	if err != nil {
		return sshremote.ExecResult{}, err
	}
	hostKey, err := sshremote.ResolveHostKeyCallback(os.Getenv)
	if err != nil {
		return sshremote.ExecResult{}, NewUpstream("ssh host key", err)
	}
	res, err := sshremote.Exec(ctx, creds, hostKey, command)
	if err != nil {
		return sshremote.ExecResult{}, NewUpstream(fmt.Sprintf("ssh exec on %s", creds.Host), err)
	}
	return res, nil
}

// authPort extracts the optional ssh-password "port", defaulting to 0 (the
// sshremote driver applies the SSH default of 22). JSON numbers decode to
// float64 through the loose Auth map, so handle both float64 and int.
func authPort(auth types.Auth) int {
	switch v := auth["port"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
