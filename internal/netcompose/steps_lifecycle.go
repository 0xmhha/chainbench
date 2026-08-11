package netcompose

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/engine"
)

// Lifecycle steps: init, start, stop, restart, rm, logs, health. They act on
// the node table the composition steps built, through the target's driver, and
// persist PIDs so a later step (or a re-run) can reach the same processes.

// binary resolves the node binary: the argument wins, else the workspace's.
func (w *Workspace) binary(arg string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	if w.state.Binary != "" {
		return w.state.Binary, nil
	}
	return "", fmt.Errorf("netcompose: a node binary is required (--binary, or set it at `net new`)")
}

// Init initializes each node's datadir from the built genesis (`<binary> init`),
// through the driver's Initializer capability.
func (w *Workspace) Init(ctx context.Context, binaryArg string) (string, error) {
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("netcompose: init: no node table — run `net allocate` first")
	}
	if w.state.GenesisPath == "" {
		return "", fmt.Errorf("netcompose: init: no genesis — run `net genesis` first")
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return "", err
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	initer, ok := t.Driver.(driver.Initializer)
	if !ok {
		return "", fmt.Errorf("netcompose: init: target driver cannot initialize datadirs")
	}
	gen, err := os.ReadFile(w.state.GenesisPath)
	if err != nil {
		return "", fmt.Errorf("netcompose: init: read genesis: %w", err)
	}
	for _, ns := range w.state.Nodes {
		spec := driverSpec(ns)
		spec.Binary = bin
		if err := initer.InitDatadir(ctx, spec, gen); err != nil {
			return "", fmt.Errorf("netcompose: init: node%d: %w", ns.Index, err)
		}
	}
	w.state.Binary = bin
	detail := fmt.Sprintf("%d datadir(s) initialized with %s", len(w.state.Nodes), bin)
	w.markStep("init", detail)
	return detail, nil
}

// Start launches every stopped node. Argv comes from the launchopts step when
// it ran; otherwise it is assembled here through the same single site
// (engine.NodeLaunchArgs) with no overrides.
func (w *Workspace) Start(ctx context.Context, binaryArg string) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("netcompose: start: no node table — run `net allocate` first")
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return "", err
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	preset, err := keys.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: start: %w", err)
	}
	started := 0
	for i, ns := range w.state.Nodes {
		if ns.PID > 0 {
			continue // already running; `net restart` bounces one node
		}
		spec := driverSpec(ns)
		spec.Binary = bin
		if len(spec.Args) == 0 {
			args, err := engine.NodeLaunchArgs(p, preset, spec, w.state.KeysDir, nil)
			if err != nil {
				return "", fmt.Errorf("netcompose: start: node%d: %w", ns.Index, err)
			}
			spec.Args = args
			w.state.Nodes[i].Args = args
		}
		h, err := t.Driver.Launch(ctx, spec)
		if err != nil {
			return "", fmt.Errorf("netcompose: start: node%d: %w", ns.Index, err)
		}
		w.state.Nodes[i].PID = h.PID
		started++
	}
	w.state.Binary = bin
	detail := fmt.Sprintf("%d node(s) started (%d already running)", started, len(w.state.Nodes)-started)
	w.markStep("start", detail)
	return detail, nil
}

// Stop terminates every running node by its recorded PID and clears the PIDs.
func (w *Workspace) Stop(ctx context.Context) (string, error) {
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	stopped := 0
	var errs []string
	for i, ns := range w.state.Nodes {
		if ns.PID <= 0 {
			continue
		}
		if err := t.Driver.Stop(ctx, driver.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
			errs = append(errs, fmt.Sprintf("node%d: %v", ns.Index, err))
			continue
		}
		w.state.Nodes[i].PID = 0
		stopped++
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("netcompose: stop: %s", strings.Join(errs, "; "))
	}
	detail := fmt.Sprintf("%d node(s) stopped", stopped)
	w.markStep("stop", detail)
	return detail, nil
}

// Restart bounces one node by index: stop (if running), relaunch with its
// recorded arming — the exact argv it started with.
func (w *Workspace) Restart(ctx context.Context, index int) (string, error) {
	ni := -1
	for i, ns := range w.state.Nodes {
		if ns.Index == index {
			ni = i
			break
		}
	}
	if ni < 0 {
		return "", fmt.Errorf("netcompose: restart: no node %d in the table", index)
	}
	bin, err := w.binary("")
	if err != nil {
		return "", err
	}
	t, err := w.state.Target.Resolve(w.env)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
	if ns.PID > 0 {
		if err := t.Driver.Stop(ctx, driver.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
			return "", fmt.Errorf("netcompose: restart: stop node%d: %w", ns.Index, err)
		}
		w.state.Nodes[ni].PID = 0
	}
	if len(ns.Args) == 0 {
		return "", fmt.Errorf("netcompose: restart: node%d has no recorded argv — run `net start` first", index)
	}
	spec := driverSpec(ns)
	spec.Binary = bin
	h, err := t.Driver.Launch(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("netcompose: restart: node%d: %w", ns.Index, err)
	}
	w.state.Nodes[ni].PID = h.PID
	detail := fmt.Sprintf("node%d restarted (pid %d)", index, h.PID)
	w.markStep("restart", detail)
	return detail, nil
}

// Rm removes the composed data plane (node datadirs, configs, genesis, logs)
// for a local target. Running nodes must be stopped first.
func (w *Workspace) Rm(ctx context.Context) (string, error) {
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 {
			return "", fmt.Errorf("netcompose: rm: node%d is running (pid %d) — run `net stop` first", ns.Index, ns.PID)
		}
	}
	if w.state.Target.IsRemote() {
		return "", fmt.Errorf("netcompose: rm: remote data-plane removal is not supported yet")
	}
	removed := 0
	for _, ns := range w.state.Nodes {
		for _, path := range []string{ns.DataDir, ns.ConfigPath} {
			if path == "" {
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				return "", fmt.Errorf("netcompose: rm: %s: %w", path, err)
			}
			removed++
		}
	}
	if w.state.GenesisPath != "" {
		if err := os.RemoveAll(w.state.GenesisPath); err != nil {
			return "", fmt.Errorf("netcompose: rm: %s: %w", w.state.GenesisPath, err)
		}
		removed++
	}
	_ = ctx
	w.state.GenesisPath = ""
	w.state.Nodes = nil
	detail := fmt.Sprintf("%d path(s) removed; node table cleared", removed)
	w.markStep("rm", detail)
	return detail, nil
}

// Logs returns the last n lines of one node's log (local target).
func (w *Workspace) Logs(index, n int) (string, error) {
	if w.state.Target.IsRemote() {
		return "", fmt.Errorf("netcompose: logs: remote log tailing is not supported yet")
	}
	for _, ns := range w.state.Nodes {
		if ns.Index != index {
			continue
		}
		b, err := os.ReadFile(ns.LogPath)
		if err != nil {
			return "", fmt.Errorf("netcompose: logs: %w", err)
		}
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if n > 0 && len(lines) > n {
			lines = lines[len(lines)-n:]
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("netcompose: logs: no node %d in the table", index)
}

// NodeHealth is one node's health probe result.
type NodeHealth struct {
	Index int    `json:"index"`
	PID   int    `json:"pid"`
	Block uint64 `json:"block"`
	Err   string `json:"error,omitempty"`
}

// Health probes every node's HTTP RPC for its latest block height. It does not
// mark a step — it is a read, re-runnable at any time.
func (w *Workspace) Health(ctx context.Context) ([]NodeHealth, error) {
	if len(w.state.Nodes) == 0 {
		return nil, fmt.Errorf("netcompose: health: no node table — run `net allocate` first")
	}
	out := make([]NodeHealth, len(w.state.Nodes))
	for i, ns := range w.state.Nodes {
		h := NodeHealth{Index: ns.Index, PID: ns.PID}
		c := rpc.Dial(fmt.Sprintf("http://127.0.0.1:%d", ns.HTTP))
		if bn, err := c.BlockNumber(ctx); err != nil {
			h.Err = err.Error()
		} else {
			h.Block = bn
		}
		out[i] = h
	}
	return out, nil
}
