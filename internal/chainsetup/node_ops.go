package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/hardfork"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/node"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/process"

	"time"

	"github.com/0xmhha/chainbench/internal/resource"
)

// Verbs that act on one node rather than the network: start, stop, restart and
// swap.
//
// A swap is the interesting one. It relaunches a node on a different binary,
// which means its genesis and its config have to be the ones that binary
// expects — a node configured for one build and launched on another is the
// failure this path exists to prevent.

// The kinds of failure acting on one node has.
var (
	// errOpNoSuchNode: the index named is not in the node table.
	errOpNoSuchNode = errors.New("no such node in the table")
	// errOpNothingToReplace: a swap that names no binary, no config change and
	// no genesis overlay has nothing to do.
	errOpNothingToReplace = errors.New("the swap asks for no change")
)

// nodeAt finds a node's position in the table by its index.
func (w *Workspace) nodeAt(index int) (int, error) {
	for i, ns := range w.state.Nodes {
		if ns.Index == index {
			return i, nil
		}
	}
	return -1, lifecycle.Mark(errOpNoSuchNode, fmt.Errorf("chainsetup: no node %d in the table", index))
}

// StopNode stops one node by index and archives its execution before clearing
// its pid; the node keeps its
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
	w.ledger.Retire(string(ns.NodeLabel()))
	w.state.Nodes[ni].PID = 0
	detail := fmt.Sprintf("node%d stopped", index)
	w.markStep("stop-node", detail)
	return detail, nil
}

// StartNode relaunches one stopped node with its recorded arming — the exact
// argv it started with — and records the new pid. A node that is already
// running is refused rather than doubled.
func (w *Workspace) StartNode(ctx context.Context, index int) (string, error) {
	if err := w.allowNode("StartNode", index); err != nil {
		return "", err
	}
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
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

// ResetNode stops one non-producing node and initialises its datadir again, so
// it holds only the genesis block. The node keeps its arming and is left down;
// a later StartNode brings it up at head 0.
//
// That is the only state in which a node grants snap sync: every client turns
// a snap request into full sync once its head is above 0. Stopping a node that
// has run is not enough, since it has already imported blocks. A producer is
// refused: wiping it takes a sealer out of the quorum.
func (w *Workspace) ResetNode(ctx context.Context, index int) (string, error) {
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	if node.Is(node.Role(w.state.Nodes[ni].Role), node.RoleBP) {
		return "", fmt.Errorf("chainsetup: reset node%d: a block producer cannot be reset", index)
	}
	if _, err := w.StopNode(ctx, index); err != nil {
		return "", fmt.Errorf("chainsetup: reset: %w", err)
	}
	ns := w.state.Nodes[ni]
	bin, err := w.binary("")
	if err != nil {
		return "", err
	}
	t, err := w.machineFor(ns)
	if err != nil {
		return "", err
	}
	initer, ok := t.Driver.(process.Initializer)
	if !ok {
		return "", lifecycle.Mark(errInitTargetUnable,
			fmt.Errorf("chainsetup: reset node%d: target driver cannot initialize datadirs", index))
	}
	spec := process.SpecOf(ns)
	spec.Binary = w.binaryFor(ns, bin)
	if err := reinitDatadir(ctx, t, initer, spec, func() ([]byte, error) {
		gen, err := t.Files.Read(ctx, w.genesisFor(ns))
		if err != nil {
			return nil, lifecycle.Mark(errInitGenesisUnreadable, fmt.Errorf("read genesis: %w", err))
		}
		return gen, nil
	}); err != nil {
		return "", fmt.Errorf("chainsetup: reset node%d: %w", index, err)
	}
	detail := fmt.Sprintf("node%d reset to its genesis and left stopped", index)
	w.markStep("reset-node", detail)
	return detail, nil
}

// SwapNodeOpts is what one node is relaunched with. Every field is optional on
// its own, but at least one must be set — a swap that changes nothing is a
// restart, and saying so is clearer than doing it silently.
type SwapNodeOpts struct {
	// Index selects the node, 1-based.
	Index int
	// Binary is the path to relaunch on (empty keeps the current one).
	Binary string
	// Config is key=value config overrides applied before relaunch.
	Config []string
	// GenesisOverlay is a genesis JSON fragment deep-merged into the network's
	// genesis and re-applied to THIS node's datadir. It is how a test gives one
	// node a genesis the rest of the network does not have — the shape a
	// "this genesis must be rejected" case needs, since a bad genesis given to
	// the whole network fails composition rather than the test.
	GenesisOverlay []byte
	// Args are flags appended to the node's recorded command line before the
	// relaunch. The command line is built in the network chain's flag dialect,
	// which knows nothing of a flag only the binary swapped onto takes (an
	// independent implementation's own flags, or one that changes how it
	// starts); the case that swaps onto that binary names them. They stay on
	// the record, so a later restart or swap keeps them.
	Args []string
	// Purpose names the config fixture recorded in provenance.
	Purpose string
}

// SwapNode stops node index and relaunches it with a different binary and/or
// config, keeping the same datadir and genesis. Generated config changes also
// update the corresponding launch arguments — a per-node swap mid-test
// (so one network runs mixed binaries), not a rebuild. The pre-swap pid and
// command are kept as a ledger revision (recordSwap); the node's per-node
// binary and config provenance are updated so a later restart uses the swapped
// ones.
func (w *Workspace) SwapNode(ctx context.Context, opts SwapNodeOpts) (detail string, err error) {
	index := opts.Index
	binary, config, purpose := opts.Binary, opts.Config, opts.Purpose
	if binary == "" && len(config) == 0 && len(opts.GenesisOverlay) == 0 && len(opts.Args) == 0 {
		return "", lifecycle.Mark(errOpNothingToReplace,
			fmt.Errorf("chainsetup: swap node%d needs a binary, a config change, a genesis overlay, or args", index))
	}
	if err := w.allowNode("SwapNode", index); err != nil {
		return "", err
	}
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
	bin, err := w.binary("")
	if err != nil {
		return "", err
	}
	t, err := w.machineFor(ns)
	if err != nil {
		return "", err
	}
	// Keep the prior ledger entry until a successful replacement supersedes it.
	// A failed replacement archives that stopped entry instead of leaving a
	// stale PID for later controls to target.
	if ns.PID > 0 {
		if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
			return "", fmt.Errorf("chainsetup: swap node%d: stop: %w", index, err)
		}
		w.state.Nodes[ni].PID = 0
		defer func() {
			if err != nil {
				w.ledger.Retire(string(ns.NodeLabel()))
			}
		}()
	}
	if binary != "" {
		w.setNodeBinary(ni, binary)
	}
	if len(config) > 0 {
		if err := w.swapNodeConfig(ctx, ni, config, purpose); err != nil {
			return "", fmt.Errorf("chainsetup: swap node%d: %w", index, err)
		}
	}
	if len(opts.Args) > 0 {
		w.state.Nodes[ni].Args = append(append([]string{}, w.state.Nodes[ni].Args...), opts.Args...)
	}
	ns = w.state.Nodes[ni]
	spec := process.SpecOf(ns)
	spec.Binary = w.binaryFor(ns, bin)
	if len(opts.GenesisOverlay) > 0 {
		if err := w.reinitNodeGenesis(ctx, t, spec, opts.GenesisOverlay); err != nil {
			return "", fmt.Errorf("chainsetup: swap node%d: %w", index, err)
		}
	}
	h, err := t.Driver.Launch(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("chainsetup: swap node%d: launch: %w", index, err)
	}
	if err := w.recordSwap(ni, h.PID, spec.Binary); err != nil {
		return "", fmt.Errorf("chainsetup: swap node%d: %w", index, err)
	}
	detail = fmt.Sprintf("node%d swapped to %s (pid %d)", index, filepath.Base(spec.Binary), h.PID)
	w.markStep("swap-node", detail)
	return detail, nil
}

// reinitNodeGenesis re-initializes one node's datadir with the network genesis
// deep-merged with overlay. It is the per-node half of Init: same driver call,
// same genesis source, one node instead of all of them.
//
// A genesis the binary refuses fails here, which is what an expect:"fail" swap
// is looking for — the node never launches and the error names the reason.
func (w *Workspace) reinitNodeGenesis(ctx context.Context, t *resource.Access, spec process.NodeSpec, overlay []byte) error {
	if w.state.GenesisPath == "" {
		return fmt.Errorf("no genesis — run `chain genesis` first")
	}
	initer, ok := t.Driver.(process.Initializer)
	if !ok {
		return fmt.Errorf("target driver cannot initialize datadirs")
	}
	base, err := t.Files.Read(ctx, w.state.GenesisPath)
	if err != nil {
		return fmt.Errorf("read genesis: %w", err)
	}
	merged, err := genesis.MergeOverride(base, overlay)
	if err != nil {
		return fmt.Errorf("merge genesis overlay: %w", err)
	}
	if err := initer.InitDatadir(ctx, spec, merged); err != nil {
		return fmt.Errorf("init datadir with overlaid genesis: %w", err)
	}
	return nil
}

// defaultBinaryName is the name a declaration gives the binary every node
// runs unless it names another (dsl.BinaryDefault).
const defaultBinaryName = "default"

// setNodeBinary registers binary under a per-node key and points node ni at it,
// so binaryFor resolves the swapped binary for this and any later launch.
//
// binary may be a path or a name the declaration gave one ("upgrade",
// "mismatch"), because that is what a case writes: it says which of the
// binaries the env declared a node should swap onto, not where that binary
// lives. A name is resolved here, once, so everything downstream holds a path.
// Storing the name instead handed it to exec, and a case that swapped onto
// "upgrade" died with `exec: "upgrade": executable file not found in $PATH`
// while the declaration said plainly what upgrade meant.
func (w *Workspace) setNodeBinary(ni int, binary string) {
	if w.state.Binaries == nil {
		w.state.Binaries = map[string]string{}
	}
	name := binary
	key := "node" + strconv.Itoa(w.state.Nodes[ni].Index)
	if binary == defaultBinaryName && w.state.Binaries[binary] == "" {
		// "default" is what the rest of the network runs: the node drops its
		// own entry and launches the network binary again.
		delete(w.state.Binaries, key)
		delete(w.state.BinaryChains, key)
		w.state.Nodes[ni].Binary = ""
		return
	}
	if path := w.state.Binaries[binary]; path != "" {
		binary = path
	}
	w.state.Binaries[key] = binary
	// The chain travels with the name. Without this a swap onto another build
	// kept its path and lost which chain it is, so the node relaunched with the
	// other build's flag vocabulary.
	if id := w.state.BinaryChains[name]; id != "" {
		if w.state.BinaryChains == nil {
			w.state.BinaryChains = map[string]string{}
		}
		w.state.BinaryChains[key] = id
	}
	w.state.Nodes[ni].Binary = key
}

// swapNodeConfig appends config overrides for node ni, re-renders and writes its
// config through the shared writeNodeConfig path (so the swap produces the same
// config and provenance a compose would), and records the config-<purpose>
// fixture in provenance. An unknown override key fails here, not at node boot.
func (w *Workspace) swapNodeConfig(ctx context.Context, ni int, config []string, purpose string) error {
	p, err := w.plugin()
	if err != nil {
		return err
	}
	preset, placed, peering, pubkey, err := w.peerPlan(p)
	if err != nil {
		return err
	}
	scope := "node" + strconv.Itoa(w.state.Nodes[ni].Index)
	if w.state.ConfigSet == nil {
		w.state.ConfigSet = map[string][]string{}
	}
	w.state.ConfigSet[scope] = append(w.state.ConfigSet[scope], config...)
	np, err := w.pluginFor(w.state.Nodes[ni])
	if err != nil {
		return err
	}
	prov, err := w.writeNodeConfig(ctx, np, preset, placed, peering, pubkey, w.state.Nodes[ni], purpose)
	if err != nil {
		return err
	}
	net, err := w.network()
	if err != nil {
		return err
	}
	ns := w.state.Nodes[ni]
	staticNodes, err := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
	if err != nil {
		return err
	}
	overrides, err := ParseOverrides(w.launchOverridesFor(ns.Role, ns.Index))
	if err != nil {
		return err
	}
	args, err := w.nodeLaunchArgs(ns, process.NodeConfig(np, net, preset, process.SpecOf(ns), w.keysBase(), staticNodes), overrides)
	if err != nil {
		return err
	}
	w.state.Nodes[ni].Args = args
	w.addConfigProvenance(prov)
	return nil
}

// addConfigProvenance appends a revision to the node's config history.
//
// It used to replace the node's entry, which lost the config the node was
// composed with the moment a swapNode gave it another one — and the state's own
// comment called each write "a new revision" while keeping exactly one. The
// requirement is to preserve the fixture, the overrides, the resulting config and
// the node with its time; a list that overwrites preserves only the last of them.
//
// A fresh compose clears the list first (see the config step), so this grows only
// with the swaps a run actually made.
func (w *Workspace) addConfigProvenance(prov ConfigProvenance) {
	if prov.At == "" {
		prov.At = w.now().UTC().Format(time.RFC3339)
	}
	w.state.ConfigProvenance = append(w.state.ConfigProvenance, prov)
}

// Hardfork swaps every node onto binary at the plan's fork, continuing the
// same chain data, and records the new pids, binary and chain.
func (w *Workspace) Hardfork(ctx context.Context, plan hardfork.SwapPlan, binary string) (node.NodeSet, error) {
	if err := w.allow("Hardfork"); err != nil {
		return node.NodeSet{}, err
	}
	specs := make([]process.NodeSpec, 0, len(w.state.Nodes))
	for _, rec := range w.state.Nodes {
		spec := process.SpecOf(rec)
		spec.Binary = w.state.Binary
		specs = append(specs, spec)
	}
	// One driver relaunches every node: the swap runs on the machine the
	// network's nodes share. (A network spread across a server set would need
	// a per-node driver; the plan executes over one.)
	t, err := w.machineFor(w.state.Nodes[0])
	if err != nil {
		return node.NodeSet{}, err
	}
	ns, err := plan.Execute(ctx, t.Driver, specs, binary)
	if err != nil {
		return ns, err
	}
	for _, n := range ns.Nodes {
		for i, rec := range w.state.Nodes {
			if rec.Index != n.Index {
				continue
			}
			if err := w.recordSwap(i, n.PID, binary); err != nil {
				return ns, fmt.Errorf("chainsetup: hardfork: node%d: %w", n.Index, err)
			}
		}
	}
	w.state.Chain = plan.ToChain
	w.state.Binary = binary
	w.markStep("hardfork", fmt.Sprintf("%s -> %s at block %d on %s", plan.FromChain, plan.ToChain, plan.Block, binary))
	return ns, nil
}

// NodeOpFailure is which state a failure of acting on one node is.
//
// The borrowed ones are the point. Deciding which binary to launch fails the
// way the launch stage fails, and re-initializing one node's datadir fails the
// way init does — the work is the same work, so it keeps the same name rather
// than getting a second one under this block.
func NodeOpFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errOpNoSuchNode):
		return lifecycle.ChainOpFailNoSuchNode
	case errors.Is(err, errOpNothingToReplace):
		return lifecycle.ChainOpReplaceNodeFailNothingAsked
	case errors.Is(err, errOpPrecondition):
		return lifecycle.ChainOpFailPrecondition
	case errors.Is(err, errLaunchNoBinary):
		return lifecycle.ChainLaunchNodesFailNoBinary
	case errors.Is(err, errLaunchPortBusy):
		return lifecycle.ChainLaunchNodesFailPortBusy
	case errors.Is(err, errInitTargetUnable):
		return lifecycle.ChainInitNodesFailTargetUnable
	case errors.Is(err, errInitGenesisUnreadable):
		return lifecycle.ChainInitNodesFailGenesisUnreadable
	}
	return ConfigFailure(err)
}
