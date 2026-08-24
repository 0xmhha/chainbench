package netcompose

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/occupancy"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/target"
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
	// Before writing anything to the target. A datadir whose node is still
	// running is refused by the binary here anyway ("datadir already used by
	// another process"), but only after the step has begun and only about the
	// datadir; asking first says which ports, on which host, and whether they
	// are this workspace's own.
	if err := w.checkVacant(ctx, registry.Phase{}); err != nil {
		return "", err
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return "", err
	}
	t, err := w.resolveTarget()
	if err != nil {
		return "", err
	}
	initer, ok := t.Driver.(driver.Initializer)
	if !ok {
		return "", fmt.Errorf("netcompose: init: target driver cannot initialize datadirs")
	}
	// GenesisPath is a path on the target: the genesis step wrote it through
	// the target's file store, so it is read back the same way. A direct
	// os.ReadFile here worked only while the target was this machine.
	gen, err := t.Files.Read(ctx, w.state.GenesisPath)
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
	t, err := w.resolveTarget()
	if err != nil {
		return "", err
	}
	preset, err := keyring.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("netcompose: start: %w", err)
	}
	// The family orders the launch. A wbft network declares one phase and this
	// is the loop it always was; a wemix network starts its producer alone so
	// the etcd cluster can form, and the bootstrap runs in the gap before the
	// rest join. Launching everything at once produced a network that came up
	// and never agreed on anything.
	roles := make([]node.Role, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		roles = append(roles, node.Role(ns.Role))
	}
	phases := p.Family().BringUpPhases(roles)

	started := 0
	for _, phase := range phases {
		launched, err := w.startPhase(ctx, t, p, preset, bin, phase)
		if err != nil {
			return "", err
		}
		started += launched
		if len(phase.Actions) == 0 {
			continue
		}
		if err := w.runPhaseActions(ctx, bin, phase); err != nil {
			return "", err
		}
	}
	w.state.Binary = bin
	detail := fmt.Sprintf("%d node(s) started (%d already running)", started, len(w.state.Nodes)-started)
	w.markStep("start", detail)
	return detail, nil
}

// Stop terminates every running node by its recorded PID and clears the PIDs.
func (w *Workspace) Stop(ctx context.Context) (string, error) {
	t, err := w.resolveTarget()
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
	t, err := w.resolveTarget()
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
	m, err := w.addrMap()
	if err != nil {
		return nil, err
	}
	out := make([]NodeHealth, len(w.state.Nodes))
	for i, ns := range w.state.Nodes {
		h := NodeHealth{Index: ns.Index, PID: ns.PID}
		// A fleet places each node on its own address, so the probe asks the
		// node's recorded host, not the target-level one.
		host, port := ns.Host, ns.HTTP
		if host == "" {
			host = w.RPCHost()
		}
		if m != nil {
			host, port = m(host, port)
		}
		c := rpc.Dial(fmt.Sprintf("http://%s:%d", host, port))
		if bn, err := c.BlockNumber(ctx); err != nil {
			h.Err = err.Error()
		} else {
			h.Block = bn
		}
		out[i] = h
	}
	return out, nil
}

// startPhase launches one phase's nodes, or every stopped node when the phase
// names none. A node already running is left alone: `net restart` bounces one,
// and re-running `net start` should not double-launch the rest.
// checkVacant refuses to launch onto ports something is already listening on.
//
// Without it the collision is discovered by the node, which dies with "address
// already in use" partway through a bring-up, and the operator has to work out
// which of three situations they are in. This says which: a port held by a node
// this workspace recorded is its own leftover and `net stop` clears it; anything
// else belongs to something this workspace did not start, and guessing would be
// worse than refusing.
func (w *Workspace) checkVacant(ctx context.Context, phase registry.Phase) error {
	var addrs []occupancy.Addr
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 || !phaseHasNode(phase, ns.Index) {
			continue
		}
		host := nodeHost(ns)
		for purpose, port := range map[string]int{
			"p2p": ns.P2P, "etcd": ns.Etcd, "etcd-client": ns.EtcdClient,
			"http": ns.HTTP, "ws": ns.WS, "auth": ns.Auth, "metrics": ns.Metrics,
		} {
			addrs = append(addrs, occupancy.Addr{Host: host, Port: port, Node: ns.Index, Purpose: purpose})
		}
	}
	dial, err := w.probeDial()
	if err != nil {
		return err
	}
	busy := occupancy.Scan(ctx, addrs, dial)
	if len(busy) == 0 {
		return nil
	}
	mine := w.recordedLeftovers()
	lines := make([]string, 0, len(busy))
	var recoverable, byHand bool
	for _, b := range busy {
		who, ok := mine[b.Port]
		switch {
		case ok && who.pid > 0:
			recoverable = true
			lines = append(lines, fmt.Sprintf("  %s — this workspace's node%d (pid %d)", b, who.node, who.pid))
		case ok:
			byHand = true
			lines = append(lines, fmt.Sprintf("  %s — this workspace's node%d, but no pid was recorded", b, who.node))
		default:
			byHand = true
			lines = append(lines, fmt.Sprintf("  %s — not started by this workspace", b))
		}
	}
	var hints []string
	if recoverable {
		hints = append(hints, "`net stop --data-dir "+w.Dir()+"` stops the ones with a recorded pid")
	}
	if byHand {
		hints = append(hints, "the rest hold ports this workspace planned but cannot address — find and stop them by hand")
	}
	hint := strings.Join(hints, "; ")
	return fmt.Errorf("netcompose: start: %d port(s) are already in use:\n%s\n%s", len(busy), strings.Join(lines, "\n"), hint)
}

// probeDial reaches an address the way this workspace's own steps would. A
// docker-backed target publishes its containers on loopback ports, and probing
// the container's real address would find nothing and report every port free.
func (w *Workspace) probeDial() (occupancy.DialFunc, error) {
	m, err := w.addrMap()
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return func(ctx context.Context, hostPort string) (net.Conn, error) {
		host, portStr, serr := net.SplitHostPort(hostPort)
		if serr != nil {
			return nil, serr
		}
		port, _ := strconv.Atoi(portStr)
		h, p := m(host, port)
		d := net.Dialer{Timeout: 300 * time.Millisecond}
		return d.DialContext(ctx, "tcp", net.JoinHostPort(h, strconv.Itoa(p)))
	}, nil
}

// recordedLeftovers maps a port to what this workspace knows about the node
// that owns it, so a collision with our own earlier run reads as that rather
// than as a stranger.
//
// Every node in the table counts, not only the ones with a pid. A workspace
// that lost its pids — the interrupted run, the run whose state file was
// removed while its nodes kept running — still owns the layout, and telling the
// operator that their own ports belong to somebody else is the least useful
// thing this check could say.
func (w *Workspace) recordedLeftovers() map[int]owner {
	out := map[int]owner{}
	for _, ns := range w.state.Nodes {
		o := owner{node: ns.Index, pid: ns.PID}
		for _, port := range []int{ns.P2P, ns.Etcd, ns.EtcdClient, ns.HTTP, ns.WS, ns.Auth, ns.Metrics} {
			if port > 0 {
				out[port] = o
			}
		}
	}
	return out
}

// owner is what this workspace knows about the node that planned a port: which
// node it is, and whether a pid was ever recorded for it. The difference
// decides the remedy — a recorded pid can be stopped, and a missing one means
// the run that started it never got to write it down.
type owner struct {
	node int
	pid  int
}

func (w *Workspace) startPhase(ctx context.Context, t *target.Target, p registry.ChainPlugin, preset keyring.Preset, bin string, phase registry.Phase) (int, error) {
	if err := w.checkVacant(ctx, phase); err != nil {
		return 0, err
	}
	started := 0
	for i, ns := range w.state.Nodes {
		if ns.PID > 0 || !phaseHasNode(phase, ns.Index) {
			continue
		}
		spec := driverSpec(ns)
		spec.Binary = bin
		if len(spec.Args) == 0 {
			args, err := engine.NodeLaunchArgs(p, preset, spec, w.state.KeysDir, nil)
			if err != nil {
				return started, fmt.Errorf("netcompose: start: node%d: %w", ns.Index, err)
			}
			spec.Args = args
			w.state.Nodes[i].Args = args
		}
		h, err := t.Driver.Launch(ctx, spec)
		if err != nil {
			return started, fmt.Errorf("netcompose: start: node%d: %w", ns.Index, err)
		}
		w.state.Nodes[i].PID = h.PID
		started++
	}
	return started, nil
}

// runPhaseActions performs the bring-up steps a phase names, against the first
// node it launched. An action with no executor is an error, not a skip — the
// phase that named it expects it to have happened, and a bootstrap quietly
// skipped is a network that starts and then does nothing.
func (w *Workspace) runPhaseActions(ctx context.Context, bin string, phase registry.Phase) error {
	specs := make([]driver.NodeSpec, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		spec := driverSpec(ns)
		spec.Binary = bin
		specs = append(specs, spec)
	}
	plan := driver.Plan{DataRoot: w.state.Target.DataRoot, Nodes: specs}

	on, ok := phaseActionNode(w.state.Nodes, phase)
	if !ok {
		return fmt.Errorf("netcompose: start: phase %q names actions but launched no node to run them on", phase.Name)
	}
	exec := engine.WemixBootstrap{Binary: bin, KeysDir: w.state.KeysDir}
	for _, name := range phase.Actions {
		if err := exec.Action(ctx, name, plan, on); err != nil {
			return fmt.Errorf("netcompose: start: phase %q: %w", phase.Name, err)
		}
	}
	return nil
}

// phaseHasNode reports whether a phase covers a node. A phase naming no nodes
// covers all of them, which is what a single-phase family declares.
func phaseHasNode(phase registry.Phase, index int) bool {
	if len(phase.Nodes) == 0 {
		return true
	}
	for _, i := range phase.Nodes {
		if i == index {
			return true
		}
	}
	return false
}

// phaseActionNode is the node a phase's actions run against. A phase that
// names one gets it — the rest phase's join concerns the boot node, which it
// did not launch. Otherwise it is the first node the phase covers, which for a
// bootstrap phase is the producer that is alone.
func phaseActionNode(nodes []NodeState, phase registry.Phase) (node.Node, bool) {
	for _, ns := range nodes {
		if phase.ActionsOn > 0 {
			if ns.Index != phase.ActionsOn {
				continue
			}
		} else if !phaseHasNode(phase, ns.Index) {
			continue
		}
		return node.Node{Index: ns.Index, Role: node.Role(ns.Role), Host: nodeHost(ns), Ports: ns.Endpoints}, true
	}
	return node.Node{}, false
}
