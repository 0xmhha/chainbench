package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/network/internal/drivers/local"
	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/probe"
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
	}
}

// newHandleNetworkLoad returns the "network.load" handler closing over stateDir.
func newHandleNetworkLoad(stateDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		var req struct {
			Name string `json:"name"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Name == "" {
			return nil, NewInvalidArgs("args.name is required")
		}
		// Pre-check name shape at the handler boundary. "local" is always legal
		// here; remote names must match the schema pattern. Rejecting here keeps
		// structural input errors as INVALID_ARGS; state-layer errors (missing
		// file) below stay as UPSTREAM_ERROR.
		if req.Name != "local" && !state.IsValidRemoteName(req.Name) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.name must be 'local' or match [a-z0-9][a-z0-9_-]*: %q", req.Name))
		}

		// Non-local names are resolved by state.LoadActive → loadRemote, which
		// returns a wrapped ErrStateNotFound when no matching attached network
		// file exists. That surfaces to callers as UPSTREAM_ERROR below.
		net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: req.Name})
		if err != nil {
			return nil, NewUpstream("failed to load active state", err)
		}

		// Marshal through JSON so the result is a plain map[string]any matching
		// the generated schema layout.
		raw, err := json.Marshal(net)
		if err != nil {
			return nil, NewInternal("marshal network", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal network", err)
		}
		return data, nil
	}
}

// resolveNodeIDFromString validates nodeID's "node" prefix and confirms it
// exists in the active network's pids.json. Core resolution used by handlers
// that already parsed their args independently. Returns APIError sentinels:
//
//	INVALID_ARGS  — empty or malformed node_id, node not found
//	UPSTREAM_ERROR — state.LoadActive failure (pids.json missing, parse error)
func resolveNodeIDFromString(stateDir, nodeID string) (string, string, error) {
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
	net, lerr := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "local"})
	if lerr != nil {
		return "", "", NewUpstream("failed to load active state", lerr)
	}
	for _, n := range net.Nodes {
		if n.Id == nodeID {
			return nodeID, num, nil
		}
	}
	return "", "", NewInvalidArgs(fmt.Sprintf("node_id %q not found in active network", nodeID))
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

// newHandleNodeStop returns a Handler that stops a node by id via LocalDriver.
// Args: { "node_id": "nodeN" } where N is the numeric pids.json key.
// On success: emits a "node.stopped" bus event, returns {node_id, stopped:true}.
func newHandleNodeStop(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		// Local-only guard: reject non-local network attachments before we
		// touch resolveNodeID, which hardcodes name:"local" when looking up
		// pids.json. Leaving this off would surface as a misleading
		// "node_id not found in active network" UPSTREAM-ish error.
		var pre struct {
			Network string `json:"network"`
		}
		if len(args) > 0 {
			_ = json.Unmarshal(args, &pre) // best-effort; main parse in resolveNodeID
		}
		if pre.Network != "" && pre.Network != "local" {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.stop is only supported on the local network (got %q)", pre.Network,
			))
		}
		nodeID, nodeNum, err := resolveNodeID(stateDir, args)
		if err != nil {
			return nil, err
		}

		driver := local.NewDriver(chainbenchDir)
		result, err := driver.StopNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("subprocess exec failed", err)
		}
		if result.ExitCode != 0 {
			tail := strings.TrimSpace(result.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("chainbench.sh node stop %s exited %d: %s", nodeNum, result.ExitCode, tail),
				nil,
			)
		}

		_ = bus.Publish(events.Event{
			Name: types.EventName("node.stopped"),
			Data: map[string]any{"node_id": nodeID, "reason": "manual"},
		})
		return map[string]any{"node_id": nodeID, "stopped": true}, nil
	}
}

// newHandleNodeStart returns a Handler that starts a previously-stopped node
// via LocalDriver. Args: { "node_id": "nodeN" }.
// On success: emits "node.started" event, returns {node_id, started:true}.
func newHandleNodeStart(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		var pre struct {
			Network string `json:"network"`
		}
		if len(args) > 0 {
			_ = json.Unmarshal(args, &pre)
		}
		if pre.Network != "" && pre.Network != "local" {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.start is only supported on the local network (got %q)", pre.Network,
			))
		}
		nodeID, nodeNum, err := resolveNodeID(stateDir, args)
		if err != nil {
			return nil, err
		}

		driver := local.NewDriver(chainbenchDir)
		result, err := driver.StartNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("subprocess exec failed", err)
		}
		if result.ExitCode != 0 {
			tail := strings.TrimSpace(result.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("chainbench.sh node start %s exited %d: %s", nodeNum, result.ExitCode, tail),
				nil,
			)
		}

		_ = bus.Publish(events.Event{
			Name: types.EventName("node.started"),
			Data: map[string]any{"node_id": nodeID},
		})
		return map[string]any{"node_id": nodeID, "started": true}, nil
	}
}

// newHandleNodeRestart returns a Handler that composes node.stop then
// node.start via LocalDriver. Args: { "node_id": "nodeN" }.
//
// Event ordering invariant:
//  1. If stop fails: no events emitted, return UPSTREAM_ERROR.
//  2. If stop succeeds + start fails: emit "node.stopped", return UPSTREAM_ERROR.
//  3. If both succeed: emit "node.stopped" then "node.started".
//
// Returns {node_id, restarted:true} on full success.
func newHandleNodeRestart(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		var pre struct {
			Network string `json:"network"`
		}
		if len(args) > 0 {
			_ = json.Unmarshal(args, &pre)
		}
		if pre.Network != "" && pre.Network != "local" {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.restart is only supported on the local network (got %q)", pre.Network,
			))
		}
		nodeID, nodeNum, err := resolveNodeID(stateDir, args)
		if err != nil {
			return nil, err
		}

		driver := local.NewDriver(chainbenchDir)

		// --- stop phase ---
		stopRes, err := driver.StopNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("restart aborted: stop exec failed", err)
		}
		if stopRes.ExitCode != 0 {
			tail := strings.TrimSpace(stopRes.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("restart aborted: stop exited %d: %s", stopRes.ExitCode, tail),
				nil,
			)
		}
		_ = bus.Publish(events.Event{
			Name: types.EventName("node.stopped"),
			Data: map[string]any{"node_id": nodeID, "reason": "restart"},
		})

		// --- start phase ---
		startRes, err := driver.StartNode(context.Background(), nodeNum)
		if err != nil {
			return nil, NewUpstream("restart incomplete: stop ok, start exec failed", err)
		}
		if startRes.ExitCode != 0 {
			tail := strings.TrimSpace(startRes.Stderr)
			if len(tail) > 512 {
				tail = tail[:512]
			}
			return nil, NewUpstream(
				fmt.Sprintf("restart incomplete: stop ok, start exited %d: %s", startRes.ExitCode, tail),
				nil,
			)
		}
		_ = bus.Publish(events.Event{
			Name: types.EventName("node.started"),
			Data: map[string]any{"node_id": nodeID},
		})

		return map[string]any{"node_id": nodeID, "restarted": true}, nil
	}
}

const (
	defaultTailLines = 50
	maxTailLines     = 1000
)

// newHandleNodeTailLog returns a Handler that reads the tail of a node's log
// file. Args: { "node_id": "nodeN", "lines": 50 (optional) }.
// No subprocess, no events — pure file read. Returns {node_id, log_file, lines}.
func newHandleNodeTailLog(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string `json:"network"`
			NodeID  string `json:"node_id"`
			Lines   *int   `json:"lines"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Network != "" && req.Network != "local" {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.tail_log is only supported on the local network (got %q)", req.Network,
			))
		}
		nodeID, _, rerr := resolveNodeIDFromString(stateDir, req.NodeID)
		if rerr != nil {
			return nil, rerr
		}
		lines := defaultTailLines
		if req.Lines != nil {
			lines = *req.Lines
		}
		if lines < 1 {
			return nil, NewInvalidArgs(fmt.Sprintf("args.lines must be >= 1, got %d", lines))
		}
		if lines > maxTailLines {
			return nil, NewInvalidArgs(fmt.Sprintf("args.lines must be <= %d, got %d", maxTailLines, lines))
		}

		net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "local"})
		if err != nil {
			return nil, NewUpstream("failed to load active state", err)
		}
		var logPath string
		for _, n := range net.Nodes {
			if n.Id == nodeID {
				if v, ok := n.ProviderMeta["log_file"].(string); ok {
					logPath = v
				}
				break
			}
		}
		if logPath == "" {
			return nil, NewUpstream(fmt.Sprintf("log_file unknown for node %q", nodeID), nil)
		}

		tailed, err := state.TailFile(logPath, lines)
		if err != nil {
			return nil, NewUpstream(fmt.Sprintf("tail log %s", logPath), err)
		}
		return map[string]any{
			"node_id":  nodeID,
			"log_file": logPath,
			"lines":    tailed,
		}, nil
	}
}

const (
	minProbeTimeoutMs = 100
	maxProbeTimeoutMs = 60000
)

// newHandleNetworkProbe returns the "network.probe" handler. It accepts
//
//	{ "rpc_url": "...", "timeout_ms"?: 100..60000, "override"?: "stablenet|wbft|wemix|ethereum" }
//
// and returns the probe.Result marshalled as a plain map. Error mapping:
//
//	INVALID_ARGS    — malformed args, missing rpc_url, out-of-range timeout_ms,
//	                  or probe.IsInputError(err) == true (sentinel
//	                  ErrMissingURL / ErrInvalidURL / ErrUnknownOverride).
//	UPSTREAM_ERROR  — any other probe.Detect failure (RPC HTTP/transport/parse).
//	INTERNAL        — JSON round-trip failure (should not happen in practice).
//
// Stateless: no stateDir / chainbenchDir dependencies, so constructed with no
// args and registered in allHandlers with a zero-arg closure.
func newHandleNetworkProbe() Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			RPCURL    string `json:"rpc_url"`
			TimeoutMs *int   `json:"timeout_ms"`
			Override  string `json:"override"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.RPCURL == "" {
			return nil, NewInvalidArgs("args.rpc_url is required")
		}
		opts := probe.Options{
			RPCURL:   req.RPCURL,
			Override: req.Override,
		}
		if req.TimeoutMs != nil {
			if *req.TimeoutMs < minProbeTimeoutMs || *req.TimeoutMs > maxProbeTimeoutMs {
				return nil, NewInvalidArgs(fmt.Sprintf(
					"args.timeout_ms must be %d..%d, got %d",
					minProbeTimeoutMs, maxProbeTimeoutMs, *req.TimeoutMs,
				))
			}
			opts.Timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
		}

		result, err := probe.Detect(context.Background(), opts)
		if err != nil {
			// Sentinel-based classification (errors.Is via probe.IsInputError)
			// avoids fragile substring matching on error messages.
			if probe.IsInputError(err) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("probe failed", err)
		}

		// Round-trip through JSON so the returned map matches the wire schema
		// layout exactly (same approach used by newHandleNetworkLoad).
		raw, err := json.Marshal(result)
		if err != nil {
			return nil, NewInternal("marshal probe result", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal probe result", err)
		}
		return data, nil
	}
}

// newHandleNetworkAttach returns the "network.attach" handler. It probes a
// remote RPC endpoint via probe.Detect, builds a types.Network from the
// result, and persists it as <stateDir>/networks/<name>.json. Subsequent
// network.load calls can resolve the network by name.
//
// Args: { "rpc_url": "...", "name": "...", "override"?: "stablenet|wbft|wemix|ethereum" }
//
// Returns: {name, chain_type, chain_id, rpc_url, nodes, created}
//
//	created=true  — no prior state file existed
//	created=false — an existing file was overwritten
//
// Error mapping:
//
//	INVALID_ARGS   — malformed args, missing/invalid name, reserved "local" name,
//	                 missing rpc_url, or probe.IsInputError(err) sentinel.
//	UPSTREAM_ERROR — probe.Detect failure on the RPC endpoint, or SaveRemote
//	                 I/O failure.
//	INTERNAL       — JSON round-trip failure.
//
// Probe runs BEFORE any filesystem write so a failed classification never
// leaves a partial state file behind.
func newHandleNetworkAttach(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			RPCURL   string `json:"rpc_url"`
			Name     string `json:"name"`
			Override string `json:"override"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.RPCURL == "" {
			return nil, NewInvalidArgs("args.rpc_url is required")
		}
		if req.Name == "" {
			return nil, NewInvalidArgs("args.name is required")
		}
		if req.Name == "local" {
			return nil, NewInvalidArgs("args.name 'local' is reserved")
		}
		// Reject structurally-invalid names at the boundary so we don't waste a
		// probe round-trip on a name we can't persist anyway.
		if !state.IsValidRemoteName(req.Name) {
			return nil, NewInvalidArgs(fmt.Sprintf(
				"args.name must match [a-z0-9][a-z0-9_-]*: %q", req.Name,
			))
		}

		result, err := probe.Detect(context.Background(), probe.Options{
			RPCURL:   req.RPCURL,
			Override: req.Override,
		})
		if err != nil {
			if probe.IsInputError(err) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("probe failed", err)
		}

		// Detect prior existence to set created=true on first save, false on
		// overwrite. Stat after probe succeeds so we never report a pseudo-
		// create on a failed attach.
		path := filepath.Join(stateDir, "networks", req.Name+".json")
		_, statErr := os.Stat(path)
		created := os.IsNotExist(statErr)

		net := &types.Network{
			Name:      req.Name,
			ChainType: types.NetworkChainType(result.ChainType),
			ChainId:   int(result.ChainID),
			Nodes: []types.Node{{
				Id:       "node1",
				Provider: types.NodeProvider("remote"),
				Http:     req.RPCURL,
			}},
		}
		if err := state.SaveRemote(stateDir, net); err != nil {
			// The handler pre-checks reserved and invalid names, so the sentinel
			// branches should be unreachable. Classify defensively anyway — a
			// future refactor that skips the pre-check must not surface a
			// structural-input error as an endpoint/UPSTREAM failure.
			if errors.Is(err, state.ErrReservedName) || errors.Is(err, state.ErrInvalidName) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("save remote state", err)
		}

		raw, err := json.Marshal(net)
		if err != nil {
			return nil, NewInternal("marshal network", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal network", err)
		}
		data["rpc_url"] = req.RPCURL
		data["created"] = created
		return data, nil
	}
}

// newHandleNodeBlockNumber returns a handler that opens an ethclient
// connection to the resolved node's HTTP endpoint and returns the current
// head block number. Works uniformly across local and remote networks
// because both populate types.Node.Http.
//
// Args: {network?: "local"|"<remote-name>", node_id: "nodeN"}
// Returns: {network, node_id, block_number}
//
// Error mapping:
//
//	INVALID_ARGS   — missing node_id, bad network name, unknown node
//	UPSTREAM_ERROR — state load failure, Dial failure, or eth_blockNumber failure
func newHandleNodeBlockNumber(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string `json:"network"`
			NodeID  string `json:"node_id"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := remote.Dial(ctx, node.Http)
		if err != nil {
			return nil, NewUpstream(fmt.Sprintf("dial %s", node.Http), err)
		}
		defer client.Close()
		bn, err := client.BlockNumber(ctx)
		if err != nil {
			return nil, NewUpstream("eth_blockNumber", err)
		}
		return map[string]any{
			"network":      networkName,
			"node_id":      node.Id,
			"block_number": bn,
		}, nil
	}
}
