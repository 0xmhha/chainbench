package chainsetup

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/inspector"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Lifecycle steps: init, start, stop, restart, rm, logs, health. They act on
// the node table the composition steps built, through the target's driver, and
// persist PIDs so a later step (or a re-run) can reach the same processes.

// binary resolves the node binary: the argument wins, else the workspace's.
// binaryFor resolves the binary one node runs: its per-node binary from the
// binaries map when the topology assigned one, otherwise the fallback (the
// composition's single binary). A workspace with no per-node binaries always
// returns the fallback, so its behavior is unchanged.
func (w *Workspace) binaryFor(ns node.Record, fallback string) string {
	if ns.Binary != "" {
		if path := w.state.Binaries[ns.Binary]; path != "" {
			return path
		}
	}
	return fallback
}

func (w *Workspace) binary(arg string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	if w.state.Binary != "" {
		return w.state.Binary, nil
	}
	return "", fmt.Errorf("chainsetup: a node binary is required (--binary, or set it at `chain new`)")
}

// Init initializes each node's datadir from the built genesis (`<binary> init`),
// through the driver's Initializer capability.
func (w *Workspace) Init(ctx context.Context, binaryArg string) (string, error) {
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("chainsetup: init: no node table — run `chain place` first")
	}
	if w.state.GenesisPath == "" {
		return "", fmt.Errorf("chainsetup: init: no genesis — run `chain genesis` first")
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
	err = w.eachMachine(func(t *resource.Access, nodes []node.Record) error {
		initer, ok := t.Driver.(process.Initializer)
		if !ok {
			return fmt.Errorf("chainsetup: init: target driver cannot initialize datadirs")
		}
		// GenesisPath is a path on the machine: the genesis step wrote it
		// through each machine's file store, so it is read back the same way.
		gen, err := t.Files.Read(ctx, w.state.GenesisPath)
		if err != nil {
			return fmt.Errorf("chainsetup: init: read genesis: %w", err)
		}
		for _, ns := range nodes {
			spec := process.SpecOf(ns)
			spec.Binary = w.binaryFor(ns, bin)
			if err := initer.InitDatadir(ctx, spec, gen); err != nil {
				return fmt.Errorf("chainsetup: init: node%d: %w", ns.Index, err)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	w.state.Binary = bin
	detail := fmt.Sprintf("%d datadir(s) initialized with %s", len(w.state.Nodes), bin)
	w.markStep("init", detail)
	return detail, nil
}

// Start launches every stopped node. Argv comes from the launchopts step when
// it ran; otherwise it is assembled here through the same single site
// (nodeconfig.Argv) with no overrides.
func (w *Workspace) Start(ctx context.Context, binaryArg string) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if len(w.state.Nodes) == 0 {
		return "", fmt.Errorf("chainsetup: start: no node table — run `chain place` first")
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return "", err
	}
	preset, err := store.LoadPreset(w.state.KeysDir)
	if err != nil {
		return "", fmt.Errorf("chainsetup: start: %w", err)
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

	if err := w.checkUnmanaged(ctx, bin); err != nil {
		return "", err
	}
	if err := w.checkPaths(ctx, bin); err != nil {
		return "", err
	}
	started := 0
	for _, phase := range phases {
		launched, err := w.startPhase(ctx, p, preset, bin, phase)
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
	rec, err := w.machineFor(w.state.Nodes[0])
	if err != nil {
		return "", err
	}
	if dir, err := w.recordRun(ctx, rec, bin); err == nil {
		detail += fmt.Sprintf("; run recorded at %s", dir)
	} else {
		// The record must never take the network it records down with it.
		detail += fmt.Sprintf("; run record failed: %v", err)
	}
	return detail, nil
}

// Stop terminates every running node by its recorded PID and clears the PIDs.
func (w *Workspace) Stop(ctx context.Context) (string, error) {
	stopped := 0
	var errs []string
	for i, ns := range w.state.Nodes {
		if ns.PID <= 0 {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			return "", err
		}
		if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
			errs = append(errs, fmt.Sprintf("node%d: %v", ns.Index, err))
			continue
		}
		w.clearPID(i)
		stopped++
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("chainsetup: stop: %s", strings.Join(errs, "; "))
	}
	detail := fmt.Sprintf("%d node(s) stopped", stopped)
	w.markStep("stop", detail)
	return detail, nil
}

// nodeAt finds a node's position in the table by its index.
func (w *Workspace) nodeAt(index int) (int, error) {
	for i, ns := range w.state.Nodes {
		if ns.Index == index {
			return i, nil
		}
	}
	return -1, fmt.Errorf("chainsetup: no node %d in the table", index)
}

// StopNode stops one node by index and clears its pid; the node keeps its
// resource and its datadir, so a later StartNode brings the same node back.
// A node that is not running is left as it is.
func (w *Workspace) StopNode(ctx context.Context, index int) (string, error) {
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
	if ns.PID <= 0 {
		return fmt.Sprintf("node%d was not running", index), nil
	}
	t, err := w.machineFor(ns)
	if err != nil {
		return "", err
	}
	if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
		return "", fmt.Errorf("chainsetup: stop node%d: %w", ns.Index, err)
	}
	w.clearPID(ni)
	detail := fmt.Sprintf("node%d stopped", index)
	w.markStep("stop-node", detail)
	return detail, nil
}

// StartNode relaunches one stopped node with its recorded arming — the exact
// argv it started with — and records the new pid. A node that is already
// running is refused rather than doubled.
func (w *Workspace) StartNode(ctx context.Context, index int) (string, error) {
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
	if ns.PID > 0 {
		return "", fmt.Errorf("chainsetup: node%d is already running (pid %d)", index, ns.PID)
	}
	if len(ns.Args) == 0 {
		return "", fmt.Errorf("chainsetup: node%d has no recorded argv — run `chain start` first", index)
	}
	bin, err := w.binary("")
	if err != nil {
		return "", err
	}
	t, err := w.machineFor(ns)
	if err != nil {
		return "", err
	}
	spec := process.SpecOf(ns)
	spec.Binary = w.binaryFor(ns, bin)
	h, err := process.LaunchAndRecord(ctx, t.Driver, w.ledger, spec)
	if err != nil {
		return "", fmt.Errorf("chainsetup: start node%d: %w", ns.Index, err)
	}
	w.state.Nodes[ni].PID = h.PID
	detail := fmt.Sprintf("node%d started (pid %d)", index, h.PID)
	w.markStep("start-node", detail)
	return detail, nil
}

// Restart bounces one node by index: stop (if running), then relaunch with
// its recorded arming.
func (w *Workspace) Restart(ctx context.Context, index int) (string, error) {
	if _, err := w.StopNode(ctx, index); err != nil {
		return "", fmt.Errorf("chainsetup: restart: %w", err)
	}
	detail, err := w.StartNode(ctx, index)
	if err != nil {
		return "", fmt.Errorf("chainsetup: restart: %w", err)
	}
	detail = fmt.Sprintf("node%d restarted", index) + strings.TrimPrefix(detail, fmt.Sprintf("node%d started", index))
	w.markStep("restart", detail)
	return detail, nil
}

// SwapNode stops node index and relaunches it on binary, keeping the same
// datadir, genesis and argv — a per-node binary swap mid-test (so one network
// runs mixed binaries), not a rebuild. The pre-swap pid and command are kept as
// a ledger revision (recordSwap), and the node's per-node binary is updated so
// a later restart uses the swapped one. binary is a path.
func (w *Workspace) SwapNode(ctx context.Context, index int, binary string) (string, error) {
	if binary == "" {
		return "", fmt.Errorf("chainsetup: swap node%d needs a binary", index)
	}
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
	if len(ns.Args) == 0 {
		return "", fmt.Errorf("chainsetup: node%d has no recorded argv — run `chain start` first", index)
	}
	t, err := w.machineFor(ns)
	if err != nil {
		return "", err
	}
	// Stop the running process but leave the ledger entry, so the relaunch
	// supersedes it and keeps the pre-swap pid/command as a revision.
	if ns.PID > 0 {
		if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
			return "", fmt.Errorf("chainsetup: swap node%d: stop: %w", index, err)
		}
	}
	w.setNodeBinary(ni, binary)
	ns = w.state.Nodes[ni]
	spec := process.SpecOf(ns)
	spec.Binary = w.binaryFor(ns, "")
	h, err := t.Driver.Launch(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("chainsetup: swap node%d: launch: %w", index, err)
	}
	if err := w.recordSwap(ni, h.PID, spec.Binary); err != nil {
		return "", fmt.Errorf("chainsetup: swap node%d: %w", index, err)
	}
	detail := fmt.Sprintf("node%d swapped to %s (pid %d)", index, filepath.Base(spec.Binary), h.PID)
	w.markStep("swap-node", detail)
	return detail, nil
}

// setNodeBinary registers binary under a per-node key and points node ni at it,
// so binaryFor resolves the swapped binary for this and any later launch.
func (w *Workspace) setNodeBinary(ni int, binary string) {
	if w.state.Binaries == nil {
		w.state.Binaries = map[string]string{}
	}
	key := "node" + strconv.Itoa(w.state.Nodes[ni].Index)
	w.state.Binaries[key] = binary
	w.state.Nodes[ni].Binary = key
}

// Rm removes the composed data plane (node datadirs, configs, genesis, logs)
// for a local target. Running nodes must be stopped first.
func (w *Workspace) Rm(ctx context.Context) (string, error) {
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 {
			return "", fmt.Errorf("chainsetup: rm: node%d is running (pid %d) — run `chain stop` first", ns.Index, ns.PID)
		}
	}
	if w.state.Target.IsRemote() {
		return "", fmt.Errorf("chainsetup: rm: remote data-plane removal is not supported yet")
	}
	removed := 0
	for _, ns := range w.state.Nodes {
		for _, path := range []string{ns.DataDir, ns.ConfigPath} {
			if path == "" {
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				return "", fmt.Errorf("chainsetup: rm: %s: %w", path, err)
			}
			removed++
		}
	}
	if w.state.GenesisPath != "" {
		if err := os.RemoveAll(w.state.GenesisPath); err != nil {
			return "", fmt.Errorf("chainsetup: rm: %s: %w", w.state.GenesisPath, err)
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

// Logs returns the last n lines of one node's log. The log lives on the
// target, so it is read through the target's file store — the same boundary that
// wrote it — which is what makes a remote node's log one call instead of a
// branch. (The collector's live tail has its own byte-offset reader; this is
// the step surface's one-shot read.)
func (w *Workspace) Logs(ctx context.Context, index, n int) (string, error) {
	for _, ns := range w.state.Nodes {
		if ns.Index != index {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			return "", err
		}
		b, err := t.Files.Read(ctx, ns.LogPath)
		if err != nil {
			return "", fmt.Errorf("chainsetup: logs: %w", err)
		}
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if n > 0 && len(lines) > n {
			lines = lines[len(lines)-n:]
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("chainsetup: logs: no node %d in the table", index)
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
		return nil, fmt.Errorf("chainsetup: health: no node table — run `chain place` first")
	}
	m, err := w.opener().AddrMap()
	if err != nil {
		return nil, err
	}
	out := make([]NodeHealth, len(w.state.Nodes))
	for i, ns := range w.state.Nodes {
		h := NodeHealth{Index: ns.Index, PID: ns.PID}
		// A network spread across a set places each node on its own address, so the probe asks the
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
// names none. A node already running is left alone: `chain restart` bounces one,
// and re-running `chain start` should not double-launch the rest.
// checkVacant refuses to launch onto ports something is already listening on.
//
// Without it the collision is discovered by the node, which dies with "address
// already in use" partway through a bring-up, and the operator has to work out
// which of three situations they are in. This says which: a port held by a node
// this workspace recorded is its own leftover and `chain stop` clears it; anything
// else belongs to something this workspace did not start, and guessing would be
// worse than refusing.
// Preflight is the check-only entry: the same pre-launch inspection Start
// runs, callable without composing anything. It answers "may a network of
// this shape start here right now?" with the refusal Start would give — port
// occupancy plus unmanaged copies of the binary already on the resource.
func (w *Workspace) Preflight(ctx context.Context, binaryArg string) error {
	if len(w.state.Nodes) == 0 {
		return fmt.Errorf("chainsetup: preflight: no node table — run `chain place` first")
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return err
	}
	if err := w.checkUnmanaged(ctx, bin); err != nil {
		return err
	}
	return w.checkVacant(ctx, registry.Phase{})
}

// checkUnmanaged asks the machine (through the driver's inspector) whether
// the binary about to be launched is already running OUTSIDE the run ledger.
// A pid the ledger knows is this workspace's and is handled per node; a pid
// it does not know belongs to someone — another workspace, an operator's
// hand-started node — and composing on top of it is refused by name.
func (w *Workspace) checkUnmanaged(ctx context.Context, bin string) error {
	name := filepath.Base(bin)
	return w.eachMachine(func(t *resource.Access, _ []node.Record) error {
		return w.checkUnmanagedOn(ctx, t, name)
	})
}

// checkUnmanagedOn is checkUnmanaged for one resource.
func (w *Workspace) checkUnmanagedOn(ctx context.Context, t *resource.Access, name string) error {
	insp, ok := t.Driver.(process.ProcessInspector)
	if !ok {
		return nil
	}
	pids, err := insp.FindBinary(ctx, name)
	if err != nil {
		return fmt.Errorf("chainsetup: process check: %w", err)
	}
	known := map[int]bool{}
	for _, p := range w.ledger.Recorded() {
		known[p.PID] = true
	}
	var strays []string
	for _, pid := range pids {
		if !known[pid] {
			strays = append(strays, strconv.Itoa(pid))
		}
	}
	if len(strays) > 0 {
		return fmt.Errorf("chainsetup: %s is already running on the machine outside this workspace (pid %s) — stop it, or compose on a different server",
			name, strings.Join(strays, ", "))
	}
	return nil
}

func (w *Workspace) checkVacant(ctx context.Context, phase registry.Phase) error {
	var addrs []inspector.Addr
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 || !phaseHasNode(phase, ns.Index) {
			continue
		}
		host := nodeHost(ns)
		for purpose, port := range map[string]int{
			"p2p": ns.P2P, "etcd": ns.Etcd, "etcd-client": ns.EtcdClient,
			"http": ns.HTTP, "ws": ns.WS, "auth": ns.Auth, "metrics": ns.Metrics,
		} {
			addrs = append(addrs, inspector.Addr{Host: host, Port: port, Node: ns.Index, Purpose: purpose})
		}
	}
	busy, err := w.scanPorts(ctx, addrs)
	if err != nil {
		return err
	}
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
		hints = append(hints, "`chain stop --workspace-dir "+w.Dir()+"` stops the ones with a recorded pid")
	}
	if byHand {
		hints = append(hints, "the rest hold ports this workspace planned but cannot address — find and stop them by hand")
	}
	hint := strings.Join(hints, "; ")
	return fmt.Errorf("chainsetup: start: %d port(s) are already in use:\n%s\n%s", len(busy), strings.Join(lines, "\n"), hint)
}

// scanPorts asks whether the plan's ports are taken, from where the
// answer is true. A local target asks this machine's kernel (inspector.Scan's
// bind probe). A remote target is asked ON the target through the driver's
// PortProber: probing from here lies in both directions — a loopback-bound
// listener on the server is invisible from outside, and a docker-published
// port is "open" from here even when nothing inside the container listens,
// because the publish forwarder itself accepts the connection (measured: an
// idle set reported every node port busy).
func (w *Workspace) scanPorts(ctx context.Context, addrs []inspector.Addr) ([]inspector.Addr, error) {
	if !w.state.Target.IsRemote() {
		return inspector.Ports(ctx, addrs, nil), nil
	}
	byHost := map[string][]int{}
	for _, a := range addrs {
		if a.Port > 0 {
			byHost[a.Host] = append(byHost[a.Host], a.Port)
		}
	}
	// Each host is probed BY ITS OWN machine (the probe lies from anywhere
	// else); the node table says which machine owns which address.
	proberFor := func(host string) (process.PortProber, error) {
		for _, ns := range w.state.Nodes {
			if nodeHost(ns) != host {
				continue
			}
			t, err := w.machineFor(ns)
			if err != nil {
				return nil, err
			}
			p, ok := t.Driver.(process.PortProber)
			if !ok {
				return nil, nil
			}
			return p, nil
		}
		return nil, nil
	}
	var busy []inspector.Addr
	for host, ports := range byHost {
		prober, err := proberFor(host)
		if err != nil {
			return nil, err
		}
		if prober == nil {
			// A machine whose driver cannot probe reports nothing rather
			// than guessing from the wrong side; the launch finds a
			// collision the old way ("address already in use").
			continue
		}
		open, err := prober.ProbePorts(ctx, host, ports)
		if err != nil {
			return nil, fmt.Errorf("chainsetup: port probe on %s: %w", host, err)
		}
		taken := map[int]bool{}
		for _, p := range open {
			taken[p] = true
		}
		for _, a := range addrs {
			if a.Host == host && taken[a.Port] {
				busy = append(busy, a)
			}
		}
	}
	return busy, nil
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

func (w *Workspace) startPhase(ctx context.Context, p registry.ChainPlugin, preset keyring.Preset, bin string, phase registry.Phase) (int, error) {
	if err := w.checkVacant(ctx, phase); err != nil {
		return 0, err
	}
	started := 0
	for i, ns := range w.state.Nodes {
		if ns.PID > 0 || !phaseHasNode(phase, ns.Index) {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			return started, err
		}
		spec := process.SpecOf(ns)
		spec.Binary = w.binaryFor(ns, bin)
		if len(spec.Args) == 0 {
			_, placed, peering, pubkey, perr := w.peerPlan(p)
			if perr != nil {
				return started, fmt.Errorf("chainsetup: start: %w", perr)
			}
			staticNodes, perr := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
			if perr != nil {
				return started, fmt.Errorf("chainsetup: start: node%d peers: %w", ns.Index, perr)
			}
			args, err := nodeconfig.Argv(process.NodeConfig(p, preset, spec, w.state.KeysDir, staticNodes))
			if err != nil {
				return started, fmt.Errorf("chainsetup: start: node%d: %w", ns.Index, err)
			}
			spec.Args = args
			w.state.Nodes[i].Args = args
		}
		h, err := process.LaunchAndRecord(ctx, t.Driver, w.ledger, spec)
		if err != nil {
			return started, fmt.Errorf("chainsetup: start: node%d: %w", ns.Index, err)
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
	specs := make([]process.NodeSpec, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		spec := process.SpecOf(ns)
		spec.Binary = bin
		specs = append(specs, spec)
	}
	plan := process.Plan{DataRoot: w.state.Target.DataRoot, Nodes: specs}

	on, ok := phaseActionNode(w.state.Nodes, phase)
	if !ok {
		return fmt.Errorf("chainsetup: start: phase %q names actions but launched no node to run them on", phase.Name)
	}
	exec := poa.Bootstrap{Binary: bin, KeysDir: w.state.KeysDir}
	// A remote target runs the bootstrap where the node is: the binary, its IPC
	// socket, the governance config and the keystore all live on the target, so
	// route the runner and the file probes through that node's access — the same
	// transport init and start use — and point the keys at where they shipped.
	if w.state.Target.IsRemote() {
		keystore, err := bootKeystoreOnTarget(w.state.KeysDir, w.keysBase(), on.Index)
		if err != nil {
			return fmt.Errorf("chainsetup: start: phase %q: %w", phase.Name, err)
		}
		exec.KeysDir = w.keysBase()
		exec.BootKeystore = keystore
		// Each node's bootstrap commands run on its own machine: a spread network
		// puts the boot node and every joiner on a different host, so resolve the
		// runner and file store per node index through the same access init/start
		// use.
		exec.Access = func(index int) (poa.Runner, filestore.Store, error) {
			rec, ok := recordByIndex(w.state.Nodes, index)
			if !ok {
				return nil, nil, fmt.Errorf("no node%d in the table", index)
			}
			access, err := w.machineFor(rec)
			if err != nil {
				return nil, nil, err
			}
			cmdr, ok := access.Driver.(process.Commander)
			if !ok {
				return nil, nil, fmt.Errorf("node%d's target cannot run a command", index)
			}
			return poa.Runner(commanderRunner(cmdr)), access.Files, nil
		}
	}
	for _, name := range phase.Actions {
		if err := exec.Action(ctx, name, plan, on); err != nil {
			return fmt.Errorf("chainsetup: start: phase %q: %w", phase.Name, err)
		}
	}
	return nil
}

// recordByIndex returns the node record with the given 1-based index.
func recordByIndex(nodes []node.Record, index int) (node.Record, bool) {
	for _, ns := range nodes {
		if ns.Index == index {
			return ns, true
		}
	}
	return node.Record{}, false
}

// bootKeystoreOnTarget returns the boot node's keystore file as a path on the
// target. The keystore's name is generated with the key and shipped unchanged,
// so it is read from the local set and re-rooted at the target keys directory —
// the store the bootstrap probes with cannot list a directory to find it.
func bootKeystoreOnTarget(localKeysDir, targetKeysDir string, index int) (string, error) {
	dir := filepath.Join(localKeysDir, fmt.Sprintf("node%d", index), "keystore")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("chainsetup: start: read keystore for node%d: %w", index, err)
	}
	for _, e := range ents {
		if !e.IsDir() {
			return path.Join(targetKeysDir, fmt.Sprintf("node%d", index), "keystore", e.Name()), nil
		}
	}
	return "", fmt.Errorf("chainsetup: start: node%d has no keystore file in %s", index, dir)
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
func phaseActionNode(nodes []node.Record, phase registry.Phase) (node.Node, bool) {
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

// checkPaths asks each node's machine whether what the launch is about to
// read is there: the binary, the genesis, and every stopped node's datadir and
// config. A missing file fails here with its name, rather than one step later
// inside a launch with "no such file" and nothing about which file.
func (w *Workspace) checkPaths(ctx context.Context, bin string) error {
	var lines []string
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			return err
		}
		want := []inspector.Path{
			{Path: bin, Purpose: "binary"},
			{Path: w.state.GenesisPath, Purpose: "genesis"},
			{Path: ns.DataDir, Node: ns.Index, Purpose: "datadir"},
			{Path: ns.ConfigPath, Node: ns.Index, Purpose: "config"},
		}
		missing, err := inspector.Paths(ctx, t.Files, want)
		if err != nil {
			return fmt.Errorf("chainsetup: start: %w", err)
		}
		for _, m := range missing {
			line := "  " + m.String()
			if ns.Server != "" {
				line += " on " + ns.Server
			}
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return fmt.Errorf("chainsetup: start: %d path(s) the launch needs are missing on the target:\n%s\nrun the earlier steps (`chain genesis`, `chain config`, `chain init`) or check --binary",
		len(lines), strings.Join(uniq(lines), "\n"))
}

// uniq drops repeated lines, keeping first occurrence order — the binary and
// genesis are checked once per node and would otherwise be listed once each.
func uniq(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
