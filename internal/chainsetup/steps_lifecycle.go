package chainsetup

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/inspector"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/resource"
	"sort"
	"sync"
	"time"
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

// genesisFor resolves the genesis one node initializes from: the one recorded
// for its binary when the composition built a separate document for that
// binary, otherwise the network's.
//
// It mirrors binaryFor deliberately. A node's binary and its genesis are the
// same question asked twice — which of the network's builds is this node — and
// two different answers to it is how they come apart.
func (w *Workspace) genesisFor(ns node.Record) string {
	if ns.Binary != "" {
		if p := w.state.GenesisPaths[ns.Binary]; p != "" {
			return p
		}
	}
	return w.state.GenesisPath
}

// pluginFor resolves the chain one node runs: the one recorded for its binary
// when that binary is a different chain, otherwise the composition's.
//
// It is the third of these — binaryFor, genesisFor, pluginFor — and they ask the
// same question: which of this network's builds is this node. Keeping them the
// same shape is what stops one of them answering differently from the others.
func (w *Workspace) pluginFor(ns node.Record) (registry.ChainPlugin, error) {
	if ns.Binary != "" {
		if id := w.state.BinaryChains[ns.Binary]; id != "" {
			return external.ResolveChain(id, "", "")
		}
	}
	return w.plugin()
}

// genesisPaths is every genesis document this composition wrote, deduplicated,
// with the network's first. Used by the steps that have to act on all of them:
// removing them, recording them, checking they are still there.
func (w *Workspace) genesisPaths() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	add(w.state.GenesisPath)
	for _, name := range slices.Sorted(maps.Keys(w.state.GenesisPaths)) {
		add(w.state.GenesisPaths[name])
	}
	return out
}

// Init initializes each node's datadir from the built genesis (`<binary> init`),
// through the driver's Initializer capability.
func (w *Workspace) Init(ctx context.Context, binaryArg string) (string, error) {
	if err := w.require("init"); err != nil {
		return "", err
	}
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
	inited := 0
	err = w.eachMachine(func(t *resource.Access, nodes []node.Record) error {
		initer, ok := t.Driver.(process.Initializer)
		if !ok {
			return fmt.Errorf("chainsetup: init: target driver cannot initialize datadirs")
		}
		// A path on the machine: the genesis step wrote it through each
		// machine's file store, so it is read back the same way. Read once per
		// document rather than once per node — a network of one binary has one
		// document and this is the same single read it always was.
		byPath := map[string][]byte{}
		readGenesis := func(p string) ([]byte, error) {
			if gen, ok := byPath[p]; ok {
				return gen, nil
			}
			gen, err := t.Files.Read(ctx, p)
			if err != nil {
				return nil, fmt.Errorf("chainsetup: init: read genesis %s: %w", p, err)
			}
			byPath[p] = gen
			return gen, nil
		}
		for _, ns := range nodes {
			// A running node's datadir is not re-initialized: reuse-if-matching
			// carries the pid of a node it leaves up, and re-initializing under a
			// live process would wipe the chain data it is serving.
			if ns.PID > 0 {
				continue
			}
			spec := process.SpecOf(ns)
			spec.Binary = w.binaryFor(ns, bin)
			// Clear the datadir first, so "init" means what it says.
			//
			// The binary refuses to init over a chain database that holds a
			// different genesis ("mismatching Boho fork block in database"),
			// which is exactly the case a rebuild is for: preflight says
			// "rebuild-all: genesis differs", the network is stopped, a new
			// genesis is written — and then init hands the old database to the
			// binary and the whole composition dies. Measured: 12 of 161 cases
			// declared a genesis overlay and none of them could run.
			//
			// Nothing else lives here. The genesis and the configs are shared
			// files at the workspace root, the identities are passed by path
			// from the key set (--nodekey), and a node that is still running is
			// skipped above — so what is removed is the chain this node built,
			// which is what a rebuild discards.
			if err := t.Files.Remove(ctx, ns.DataDir); err != nil {
				return fmt.Errorf("chainsetup: init: node%d: clear datadir: %w", ns.Index, err)
			}
			// The genesis this node's binary accepts, which is not always the
			// network's: two builds in one network need not take the same
			// document.
			gen, err := readGenesis(w.genesisFor(ns))
			if err != nil {
				return fmt.Errorf("chainsetup: init: node%d: %w", ns.Index, err)
			}
			if err := initer.InitDatadir(ctx, spec, gen); err != nil {
				return fmt.Errorf("chainsetup: init: node%d: %w", ns.Index, err)
			}
			inited++
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	w.state.Binary = bin
	detail := fmt.Sprintf("%d datadir(s) initialized with %s", inited, bin)
	if reused := len(w.state.Nodes) - inited; reused > 0 {
		detail += fmt.Sprintf(" (%d left running)", reused)
	}
	w.markStep("init", detail)
	return detail, nil
}

// Start launches every stopped node. Argv comes from the launchopts step when
// it ran; otherwise it is assembled here through the same single site
// (nodeconfig.Argv) with no overrides.
func (w *Workspace) Start(ctx context.Context, binaryArg string) (string, error) {
	if err := w.require("start"); err != nil {
		return "", err
	}
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
//
// Every node is attempted. A node whose machine cannot be resolved used to end
// the whole loop, so one unreachable server left every node after it running —
// and the caller was told "stop failed", which reads as "nothing stopped" when
// the truth was "some stopped, and I do not know which". Now each node's
// failure is collected and the rest are still stopped; the error names them all.
//
// The nodes are stopped concurrently because stopping one is mostly waiting:
// the driver sends SIGTERM and gives the process up to process.StopGrace to
// close its database. Done in sequence, five nodes take five grace periods —
// measured at 15s each, so a network that will not go down quietly held the
// next run's ports for over a minute. Done together they take one.
func (w *Workspace) Stop(ctx context.Context) (string, error) {
	type outcome struct {
		i   int
		err error
	}
	var (
		mu       sync.Mutex
		results  []outcome
		wg       sync.WaitGroup
		attempts int
	)
	for i, ns := range w.state.Nodes {
		if ns.PID <= 0 {
			continue
		}
		attempts++
		// Resolving the machine touches the workspace's memoized map, so it is
		// done here, one at a time, and only the wait runs concurrently.
		t, err := w.machineFor(ns)
		if err != nil {
			mu.Lock()
			results = append(results, outcome{i: i, err: fmt.Errorf("resolve machine: %w", err)})
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(i int, ns node.Record, t *resource.Access) {
			defer wg.Done()
			err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID})
			mu.Lock()
			results = append(results, outcome{i: i, err: err})
			mu.Unlock()
		}(i, ns, t)
	}
	wg.Wait()

	// Sorted so the same failure reads the same way twice: goroutines finish in
	// whatever order the processes happen to die.
	sort.Slice(results, func(a, b int) bool { return results[a].i < results[b].i })
	stopped := 0
	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Sprintf("node%d: %v", w.state.Nodes[r.i].Index, r.err))
			continue
		}
		w.clearPID(r.i)
		stopped++
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("chainsetup: stop: %d of %d node(s) stopped; %s",
			stopped, attempts, strings.Join(errs, "; "))
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
	// Purpose names the config fixture recorded in provenance.
	Purpose string
}

// SwapNode stops node index and relaunches it with a different binary and/or
// config, keeping the same datadir, genesis and argv — a per-node swap mid-test
// (so one network runs mixed binaries), not a rebuild. The pre-swap pid and
// command are kept as a ledger revision (recordSwap); the node's per-node
// binary and config provenance are updated so a later restart uses the swapped
// ones.
func (w *Workspace) SwapNode(ctx context.Context, opts SwapNodeOpts) (string, error) {
	index := opts.Index
	binary, config, purpose := opts.Binary, opts.Config, opts.Purpose
	if binary == "" && len(config) == 0 && len(opts.GenesisOverlay) == 0 {
		return "", fmt.Errorf("chainsetup: swap node%d needs a binary, a config change, or a genesis overlay", index)
	}
	ni, err := w.nodeAt(index)
	if err != nil {
		return "", err
	}
	ns := w.state.Nodes[ni]
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
	// Stop the running process but leave the ledger entry, so the relaunch
	// supersedes it and keeps the pre-swap pid/command as a revision.
	if ns.PID > 0 {
		if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID}); err != nil {
			return "", fmt.Errorf("chainsetup: swap node%d: stop: %w", index, err)
		}
	}
	if binary != "" {
		w.setNodeBinary(ni, binary)
	}
	if len(config) > 0 {
		if err := w.swapNodeConfig(ctx, ni, config, purpose); err != nil {
			return "", fmt.Errorf("chainsetup: swap node%d: %w", index, err)
		}
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
	detail := fmt.Sprintf("node%d swapped to %s (pid %d)", index, filepath.Base(spec.Binary), h.PID)
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
	if path := w.state.Binaries[binary]; path != "" {
		binary = path
	}
	key := "node" + strconv.Itoa(w.state.Nodes[ni].Index)
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

// Rm removes the composed data plane (node datadirs, configs, genesis, logs)
// for a local target. Running nodes must be stopped first.
func (w *Workspace) Rm(ctx context.Context) (string, error) {
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 {
			return "", fmt.Errorf("chainsetup: rm: node%d is running (pid %d) — run `chain stop` first", ns.Index, ns.PID)
		}
	}
	// Removal goes through the target's file store, the same boundary that wrote
	// these paths. That is what makes a remote data plane removable at all: this
	// used to call os.RemoveAll directly and so could only ever clear the local
	// machine, which is why the remote case was a refusal rather than a branch.
	//
	// Every path is confined to the target's data root before it is deleted. The
	// store applies its own coarse guard underneath; this is the precise one,
	// made here because this is the layer that knows what the root is.
	removed := 0
	remove := func(acc *resource.Access, p string) error {
		if p == "" {
			return nil
		}
		if err := filestore.CheckWithin(acc.DataRoot, p); err != nil {
			return fmt.Errorf("chainsetup: rm: %w", err)
		}
		if err := acc.Files.Remove(ctx, p); err != nil {
			return fmt.Errorf("chainsetup: rm: %s: %w", p, err)
		}
		removed++
		return nil
	}
	// The genesis lives on every machine the network was placed on, so it is
	// cleared once per machine rather than once — a set, because two nodes on
	// the same server share the file and removing it twice is not an error but
	// is a second round trip.
	genesisDone := map[string]bool{}
	for _, ns := range w.state.Nodes {
		acc, err := w.machineFor(ns)
		if err != nil {
			return "", fmt.Errorf("chainsetup: rm: node%d: %w", ns.Index, err)
		}
		for _, p := range []string{ns.DataDir, ns.ConfigPath} {
			if err := remove(acc, p); err != nil {
				return "", err
			}
		}
		if !genesisDone[ns.Server] {
			for _, p := range w.genesisPaths() {
				if err := remove(acc, p); err != nil {
					return "", err
				}
			}
			genesisDone[ns.Server] = true
		}
	}
	w.state.GenesisPath = ""
	w.state.GenesisPaths = nil
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
		// A node's log lives on its machine, and may be root-owned there, so it is
		// read through the machine (remote or local) and elevated through sudo
		// where the login user cannot reach it and the server set permits it.
		b, err := t.ReadMaybeElevated(ctx, ns.LogPath)
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

// LogExcerpt returns the first head lines and the last tail lines of one node's
// log, with a line in between saying how much was left out.
//
// The tail alone is not enough for the failure this exists to explain. A node
// that refuses its genesis, or cannot bind a port, says so in its first few
// lines and then exits; a node that dies after an hour says so in its last. At
// one block per second a geth-family node writes several lines a second, so a
// 200-line tail is the last half minute — which is exactly the window that does
// NOT contain a startup failure.
//
// A log shorter than head+tail is returned whole: eliding nothing is not worth
// a marker saying so.
func (w *Workspace) LogExcerpt(ctx context.Context, index, head, tail int) (string, error) {
	full, err := w.Logs(ctx, index, 0)
	if err != nil {
		return "", err
	}
	return excerpt(full, head, tail), nil
}

// excerpt keeps both ends of s and says what it dropped.
func excerpt(s string, head, tail int) string {
	if head <= 0 && tail <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= head+tail {
		return s
	}
	out := make([]string, 0, head+tail+1)
	out = append(out, lines[:head]...)
	out = append(out, fmt.Sprintf("... %d line(s) elided ...", len(lines)-head-tail))
	out = append(out, lines[len(lines)-tail:]...)
	return strings.Join(out, "\n")
}

// livePIDs asks each node's machine whether its recorded pid is still a
// process, for the nodes that have one.
//
// Best effort, and deliberately silent about its own failures: this answers
// "what is running", and a machine that cannot be reached has not told us the
// node is gone. Absent from the map means "not asked or could not ask", which
// a caller must not read as "dead" — the map only ever carries answers.
func (w *Workspace) livePIDs(ctx context.Context) map[int]bool {
	out := map[int]bool{}
	for _, ns := range w.state.Nodes {
		if ns.PID <= 0 {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			continue
		}
		insp, ok := t.Driver.(process.ProcessInspector)
		if !ok {
			continue
		}
		alive, err := insp.PIDAlive(ctx, ns.PID)
		if err != nil {
			continue
		}
		out[ns.Index] = alive
	}
	return out
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
	out := make([]NodeHealth, len(w.state.Nodes))
	for i, ns := range w.state.Nodes {
		h := NodeHealth{Index: ns.Index, PID: ns.PID}
		// A network spread across a set places each node on its own address, so the probe asks the
		// node's recorded host, not the target-level one; the resource layer turns
		// that into the address this machine actually dials.
		url, err := w.nodeHTTPURL(ns)
		if err != nil {
			return nil, err
		}
		c := rpc.Dial(url)
		if bn, err := c.BlockNumber(ctx); err != nil {
			h.Err = err.Error()
		} else {
			h.Block = bn
		}
		out[i] = h
	}
	return out, nil
}

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

// checkVacant refuses to launch onto ports something is already listening on.
//
// Without it the collision is discovered by the node, which dies with "address
// already in use" partway through a bring-up, and the operator has to work out
// which of three situations they are in. This says which: a port held by a node
// this workspace recorded is its own leftover and `chain stop` clears it; anything
// else belongs to something this workspace did not start, and guessing would be
// worse than refusing.
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
		who, ok := mine[portKey{host: b.Host, port: b.Port}]
		switch {
		case ok && who.pid > 0:
			recoverable = true
			lines = append(lines, fmt.Sprintf("  %s — this workspace's node%d (pid %d)", b, who.node, who.pid))
		case ok:
			// This workspace planned the address and never recorded a pid for
			// it, so whatever is listening is not something it started. Saying
			// "this workspace's node%d" claimed the opposite, and the operator
			// who believed it went looking in the wrong composition.
			byHand = true
			lines = append(lines, fmt.Sprintf("  %s — planned for this workspace's node%d, but it started nothing there: another composition holds it", b, who.node))
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
func (w *Workspace) recordedLeftovers() map[portKey]owner {
	out := map[portKey]owner{}
	for _, ns := range w.state.Nodes {
		o := owner{node: ns.Index, pid: ns.PID}
		host := nodeHost(ns)
		for _, port := range []int{ns.P2P, ns.Etcd, ns.EtcdClient, ns.HTTP, ns.WS, ns.Auth, ns.Metrics} {
			if port > 0 {
				out[portKey{host: host, port: port}] = o
			}
		}
	}
	return out
}

// portKey addresses a planned port the way the scan reports a busy one: by host
// and port, never by port alone.
//
// A port number is not an identity here. Spread across a server set, every
// server runs its slot-1 node on the same numbers — 8601, 30301 — so a map
// keyed on the number alone answers "who planned 8601?" with whichever node was
// written last, and a busy port on one server gets reported as a node on
// another. Same collapse as the running-node map had (MON-010), one file over.
type portKey struct {
	host string
	port int
}

// owner is what this workspace knows about the node that planned a port: which
// node it is, and whether a pid was ever recorded for it. The difference
// decides the remedy — a recorded pid can be stopped, and a missing one means
// the run that started it never got to write it down.
type owner struct {
	node int
	pid  int
}

// startPhase launches one phase's nodes, or every stopped node when the phase
// names none. A node already running is left alone: `chain restart` bounces one,
// and re-running `chain start` should not double-launch the rest.
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
			// This node's own chain, which is not always the composition's: a
			// network can run two builds, and the chain is what says which flag
			// vocabulary the binary accepts and which RPC namespace it serves.
			np, perr := w.pluginFor(ns)
			if perr != nil {
				return started, fmt.Errorf("chainsetup: start: node%d: %w", ns.Index, perr)
			}
			args, err := nodeconfig.Argv(process.NodeConfig(np, preset, spec, w.state.KeysDir, staticNodes))
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
	plan := w.phasePlan(bin)

	on, ok := phaseActionNode(w.state.Nodes, phase)
	if !ok {
		return fmt.Errorf("chainsetup: start: phase %q names actions but launched no node to run them on", phase.Name)
	}
	// No Binary override: the executor already prefers the plan's own entry for
	// the node it runs on, and that is the node's binary. Naming one here
	// overrode it with the network's single binary, which is wrong the moment
	// the network runs more than one — and it runs more than one on purpose,
	// for a swap and for a handoff across a fork. The socket the bootstrap
	// attaches to is derived from the binary, so a node running the other one
	// was waited for at a path it never creates.
	exec := poa.Bootstrap{KeysDir: w.state.KeysDir}
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

// phasePlan is the launch plan a phase's actions run against: every node, each
// with the binary IT runs.
//
// Per node, not per network. A network can run more than one binary on purpose
// — a case swaps a node onto another build, a handoff puts the pre-fork and
// post-fork binaries in one network from genesis — and a bring-up action that
// assumed one of them addressed the others through the wrong binary.
func (w *Workspace) phasePlan(bin string) process.Plan {
	specs := make([]process.NodeSpec, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		spec := process.SpecOf(ns)
		spec.Binary = w.binaryFor(ns, bin)
		specs = append(specs, spec)
	}
	return process.Plan{DataRoot: w.state.Target.DataRoot, GenesisPath: w.state.GenesisPath, Nodes: specs}
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
		// The binary is asked for separately because a name is not a path: a
		// bare name is whatever the target's PATH resolves, and stating it
		// would report a binary the launch will find as missing.
		if err := checkBinary(ctx, t, bin); err != nil {
			lines = append(lines, "  "+err.Error())
		}
		want := []inspector.Path{
			{Path: w.genesisFor(ns), Purpose: "genesis"},
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
	return fmt.Errorf("chainsetup: start: %d thing(s) the launch needs are missing on the target:\n%s\nrun the earlier steps (`chain genesis`, `chain config`, `chain init`) or check --binary",
		len(lines), strings.Join(uniq(lines), "\n"))
}

// checkBinary reports the binary as missing when the target cannot produce it,
// whether it was named as a path or as a command.
//
// It is the one pre-launch check that cannot be a file lookup. A workspace-
// config places the binary under the data root and the answer is a path; with
// no workspace-config the name is the target's to resolve on PATH, and asking
// the file store about it stats it against the working directory and answers
// no for a binary the launch would have found.
func checkBinary(ctx context.Context, t *resource.Access, bin string) error {
	if bin == "" {
		return fmt.Errorf("binary: none is set")
	}
	if strings.ContainsRune(bin, '/') {
		ok, err := t.Files.Exists(ctx, bin)
		if err != nil {
			return fmt.Errorf("binary %s: %v", bin, err)
		}
		if !ok {
			return fmt.Errorf("binary %s: not on the target", bin)
		}
		return nil
	}
	path, ok, err := inspector.OnPath(ctx, t.Runner, bin)
	if err != nil {
		return fmt.Errorf("binary %s: %v", bin, err)
	}
	if !ok {
		return fmt.Errorf("binary %s: not on the target's PATH (name it in a workspace-config, or pass --binary with a path)", bin)
	}
	_ = path
	return nil
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
