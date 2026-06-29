package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/network/internal/drivers/local"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)

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
			var nid struct {
				NodeID string `json:"node_id"`
			}
			_ = json.Unmarshal(args, &nid)
			return handleSSHNodeLifecycle(stateDir, pre.Network, nid.NodeID, "stop", bus)
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
// via LocalDriver. Args: { "node_id": "nodeN", "binary_path": "/abs/path"
// (optional) }.
//
// When binary_path is supplied, it is forwarded to chainbench.sh as
// `--binary-path <path>` so the LLM caller can override the profile's
// chain.binary_path for this single start without rewriting the profile.
// The path must be absolute (start with `/`) — relative paths and the empty
// string are rejected as INVALID_ARGS at the handler boundary so the
// downstream bash script never sees a malformed override.
//
// On success: emits "node.started" event, returns {node_id, started:true}.
func newHandleNodeStart(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		// *string discriminates "key absent" (nil) from "key present with empty
		// value" (non-nil pointer to ""). One pointer indirection is the
		// canonical Go idiom for optional-string args; substring-scanning the
		// raw JSON would false-positive on unrelated keys whose values happen
		// to contain "binary_path".
		var pre struct {
			Network    string  `json:"network"`
			BinaryPath *string `json:"binary_path"`
		}
		if len(args) > 0 {
			_ = json.Unmarshal(args, &pre)
		}
		if pre.Network != "" && pre.Network != "local" {
			// binary_path is a local-only override (it tweaks the local
			// chainbench.sh launch); ssh-remote startup is defined by the
			// node's provider_meta.start_cmd, so binary_path is ignored here.
			var nid struct {
				NodeID string `json:"node_id"`
			}
			_ = json.Unmarshal(args, &nid)
			return handleSSHNodeLifecycle(stateDir, pre.Network, nid.NodeID, "start", bus)
		}
		// binary_path is optional; reject explicit empty / relative paths.
		// Absent key (pre.BinaryPath == nil) → use profile default.
		if pre.BinaryPath != nil && *pre.BinaryPath == "" {
			return nil, NewInvalidArgs("args.binary_path must not be empty")
		}
		if pre.BinaryPath != nil && !strings.HasPrefix(*pre.BinaryPath, "/") {
			return nil, NewInvalidArgs(fmt.Sprintf(
				"args.binary_path must be an absolute path, got %q", *pre.BinaryPath,
			))
		}
		var binaryPath string
		if pre.BinaryPath != nil {
			binaryPath = *pre.BinaryPath
		}
		nodeID, nodeNum, err := resolveNodeID(stateDir, args)
		if err != nil {
			return nil, err
		}

		driver := local.NewDriver(chainbenchDir)
		result, err := driver.StartNode(context.Background(), nodeNum, binaryPath)
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
			var nid struct {
				NodeID string `json:"node_id"`
			}
			_ = json.Unmarshal(args, &nid)
			return handleSSHNodeLifecycle(stateDir, pre.Network, nid.NodeID, "restart", bus)
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
		// Restart does not currently propagate binary_path; the override is a
		// node.start-only arg in this sprint. Pass empty string to use the
		// profile's chain.binary_path.
		startRes, err := driver.StartNode(context.Background(), nodeNum, "")
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

// runSSHNodeCmd executes the provider_meta command named cmdKey (e.g.
// "stop_cmd") on an ssh-remote node and classifies the result.
//
//	NOT_SUPPORTED  — the command is not configured on this node.
//	UPSTREAM_ERROR — SSH/exec failure, or the command exited non-zero.
func runSSHNodeCmd(node *types.Node, cmdKey string) error {
	cmd, _ := node.ProviderMeta[cmdKey].(string)
	if cmd == "" {
		return NewNotSupported(fmt.Sprintf(
			"node %q has no provider_meta.%s configured", node.Id, cmdKey))
	}
	res, err := execSSHNode(context.Background(), node, cmd)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return NewUpstream(fmt.Sprintf(
			"%s exited %d: %s", cmdKey, res.ExitCode, truncStderr(res.Stderr)), nil)
	}
	return nil
}

// handleSSHNodeLifecycle runs a stop/start/restart on an ssh-remote node via its
// provider_meta commands, emitting the same bus events as the local path. A
// non-ssh-remote provider (e.g. "remote") yields NOT_SUPPORTED — the process
// capability is local/ssh-remote only.
func handleSSHNodeLifecycle(stateDir, network, nodeID, op string, bus *events.Bus) (map[string]any, error) {
	_, node, err := resolveNode(stateDir, network, nodeID)
	if err != nil {
		return nil, err
	}
	if node.Provider != types.NodeProviderSshRemote {
		return nil, NewNotSupported(fmt.Sprintf(
			"node.%s requires the process capability; provider %q does not provide it", op, node.Provider))
	}
	switch op {
	case "stop":
		if err := runSSHNodeCmd(&node, "stop_cmd"); err != nil {
			return nil, err
		}
		_ = bus.Publish(events.Event{Name: types.EventName("node.stopped"),
			Data: map[string]any{"node_id": node.Id, "reason": "manual"}})
		return map[string]any{"node_id": node.Id, "stopped": true}, nil
	case "start":
		if err := runSSHNodeCmd(&node, "start_cmd"); err != nil {
			return nil, err
		}
		_ = bus.Publish(events.Event{Name: types.EventName("node.started"),
			Data: map[string]any{"node_id": node.Id}})
		return map[string]any{"node_id": node.Id, "started": true}, nil
	case "restart":
		// Prefer a single restart_cmd; otherwise compose stop_cmd → start_cmd
		// with the same event ordering invariant as the local restart.
		if cmd, _ := node.ProviderMeta["restart_cmd"].(string); cmd != "" {
			if err := runSSHNodeCmd(&node, "restart_cmd"); err != nil {
				return nil, err
			}
			_ = bus.Publish(events.Event{Name: types.EventName("node.stopped"),
				Data: map[string]any{"node_id": node.Id, "reason": "restart"}})
			_ = bus.Publish(events.Event{Name: types.EventName("node.started"),
				Data: map[string]any{"node_id": node.Id}})
			return map[string]any{"node_id": node.Id, "restarted": true}, nil
		}
		if err := runSSHNodeCmd(&node, "stop_cmd"); err != nil {
			return nil, err
		}
		_ = bus.Publish(events.Event{Name: types.EventName("node.stopped"),
			Data: map[string]any{"node_id": node.Id, "reason": "restart"}})
		if err := runSSHNodeCmd(&node, "start_cmd"); err != nil {
			return nil, err
		}
		_ = bus.Publish(events.Event{Name: types.EventName("node.started"),
			Data: map[string]any{"node_id": node.Id}})
		return map[string]any{"node_id": node.Id, "restarted": true}, nil
	default:
		return nil, NewInvalidArgs(fmt.Sprintf("unknown lifecycle op %q", op))
	}
}

const (
	defaultTailLines = 50
	maxTailLines     = 1000
)

// newHandleNodeTailLog returns a Handler that reads the tail of a node's log
// file. Args: { "network": "name" (optional), "node_id": "nodeN",
// "lines": 50 (optional) }.
//
// Provider dispatch (fs capability):
//   - local: pure local file read (state.TailFile).
//   - ssh-remote: `tail -n <lines> -- <log_file>` over SSH (Sprint 5b.2),
//     where log_file comes from the node's provider_meta. A node without a
//     provider_meta.log_file does not provide fs → NOT_SUPPORTED.
//   - remote: no filesystem access → NOT_SUPPORTED.
//
// Returns {node_id, log_file, lines}.
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

		// --- local: pure file read (unchanged) ---
		if req.Network == "" || req.Network == "local" {
			nodeID, _, rerr := resolveNodeIDFromString(stateDir, req.NodeID)
			if rerr != nil {
				return nil, rerr
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
			return map[string]any{"node_id": nodeID, "log_file": logPath, "lines": tailed}, nil
		}

		// --- non-local: dispatch on provider ---
		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		if node.Provider != types.NodeProviderSshRemote {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.tail_log requires the fs capability; provider %q does not provide it", node.Provider))
		}
		logFile, _ := node.ProviderMeta["log_file"].(string)
		if logFile == "" {
			return nil, NewNotSupported(fmt.Sprintf(
				"node %q has no provider_meta.log_file; tail_log unavailable", node.Id))
		}
		cmd := fmt.Sprintf("tail -n %d -- %s", lines, shellQuote(logFile))
		res, err := execSSHNode(context.Background(), &node, cmd)
		if err != nil {
			return nil, err
		}
		if res.ExitCode != 0 {
			return nil, NewUpstream(fmt.Sprintf(
				"tail %s exited %d: %s", logFile, res.ExitCode, truncStderr(res.Stderr)), nil)
		}
		return map[string]any{"node_id": node.Id, "log_file": logFile, "lines": splitLines(res.Stdout)}, nil
	}
}

// shellQuote single-quotes s for safe interpolation into a remote shell command,
// escaping embedded single quotes. Used for operator-set paths (log_file) as a
// defensive measure even though they are not caller-controlled.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// splitLines splits command stdout into lines, trimming a single trailing
// newline and returning an empty (non-nil) slice for empty output — matching
// the []string shape state.TailFile returns for the local path.
func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}

// truncStderr trims and caps a command's stderr for inclusion in an error.
func truncStderr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 512 {
		return s[:512]
	}
	return s
}
