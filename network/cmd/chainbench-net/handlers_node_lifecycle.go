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
