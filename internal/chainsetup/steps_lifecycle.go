package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"

	"sort"
	"sync"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/preset"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Lifecycle steps: init, start, stop. They act on the node table the
// composition steps built, through the target's driver, and persist PIDs so a
// later step — or a re-run — reaches the same processes.
//
// Single-node verbs (restart, swap) are in node_ops.go, observation (logs,
// health) in observe.go, and the checks that run before a launch in
// occupancy.go. They stay in this package: the split is for the reader.

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

// genesisConfigFor resolves the genesis one node reads from its config file:
// the one recorded for its binary when the composition carried a fork there,
// and otherwise nothing — the ordinary node's config says nothing about the
// genesis, because it initialized from the genesis document like every other.
//
// It mirrors genesisFor, and for the same reason: a node's binary decides both,
// and two shapes of answer is how they come apart.
func (w *Workspace) genesisConfigFor(ns node.Record) string {
	if ns.Binary != "" {
		return w.state.GenesisConfigPaths[ns.Binary]
	}
	return ""
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

// genesisPaths is every genesis this composition wrote, deduplicated, with the
// network's first: the documents nodes initialize from, then the configs a
// build reads its genesis from instead. Used by the steps that have to act on
// all of them: removing them, recording them, checking they are still there.
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
	for _, name := range slices.Sorted(maps.Keys(w.state.GenesisConfigPaths)) {
		add(w.state.GenesisConfigPaths[name])
	}
	return out
}

// The kinds of failure the init stage has.
//
// What is deliberately not one of them is the binary's own refusal:
// InitDatadir covers "the binary is not there", "it will not take this genesis"
// and "the datadir is in use", and which of those it was is the driver's answer
// rather than this step's. Naming one of these three over it would say
// something the error does not.
var (
	// errInitTargetUnable: the target's driver cannot initialize a datadir.
	errInitTargetUnable = errors.New("the target cannot initialize a datadir")
	// errInitGenesisUnreadable: the genesis this node needs cannot be read back
	// from the machine the genesis stage wrote it to.
	errInitGenesisUnreadable = errors.New("the genesis cannot be read from the target")
	// errInitDatadir: the datadir could not be cleared, so "init" would not
	// mean what it says.
	errInitDatadir = errors.New("the datadir cannot be cleared")
)

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
			return ofKind(errInitTargetUnable,
				fmt.Errorf("chainsetup: init: target driver cannot initialize datadirs"))
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
				return nil, ofKind(errInitGenesisUnreadable,
					fmt.Errorf("chainsetup: init: read genesis %s: %w", p, err))
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
				return ofKind(errInitDatadir,
					fmt.Errorf("chainsetup: init: node%d: clear datadir: %w", ns.Index, err))
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
func (w *Workspace) Start(ctx context.Context, binaryArg string) (StepOut, error) {
	if err := w.require("start"); err != nil {
		return StepOut{}, err
	}
	p, err := w.plugin()
	if err != nil {
		return StepOut{}, err
	}
	if len(w.state.Nodes) == 0 {
		return StepOut{}, fmt.Errorf("chainsetup: start: no node table — run `chain place` first")
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return StepOut{}, err
	}
	// With accounts: a producer unlocks the account its keystore holds, which is
	// not always the address its nodekey derives.
	preset, err := preset.LoadKeyPresetWithAccounts(w.state.KeysDir)
	if err != nil {
		return StepOut{}, fmt.Errorf("chainsetup: start: %w", err)
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
		return StepOut{}, err
	}
	if err := w.checkPaths(ctx, bin); err != nil {
		return StepOut{}, err
	}
	// The walk through the phases, as states. A family decides how many there
	// are — wbft declares one, a poa network declares a boot plus one join per
	// producer — so the count is not this package's to know, and the path is
	// built as the loop runs rather than assumed in front of it.
	//
	// The state says what the launch was doing; how many times it has been
	// through says which phase. A launch that dies in the third join reports
	// three PhaseLaunching and stops there, which is the thing the loop alone
	// could not say: the record used to hold "start" and nothing else.
	var passed []lifecycle.Status
	started := 0
	for _, phase := range phases {
		passed = append(passed, lifecycle.ChainLaunchNodesPhaseLaunching)
		launched, err := w.startPhase(ctx, p, preset, bin, phase)
		if err != nil {
			return StepOut{Passed: passed}, err
		}
		started += launched
		if len(phase.Actions) > 0 {
			passed = append(passed, lifecycle.ChainLaunchNodesPhaseActions)
			if err := w.runPhaseActions(ctx, bin, phase); err != nil {
				return StepOut{Passed: passed}, err
			}
		}
		passed = append(passed, lifecycle.ChainLaunchNodesPhaseDone)
	}
	w.state.Binary = bin
	detail := fmt.Sprintf("%d node(s) started (%d already running)", started, len(w.state.Nodes)-started)
	w.markStep("start", detail)
	rec, err := w.machineFor(w.state.Nodes[0])
	if err != nil {
		return StepOut{}, err
	}
	if dir, err := w.recordRun(ctx, rec, bin); err == nil {
		detail += fmt.Sprintf("; run recorded at %s", dir)
	} else {
		// The record must never take the network it records down with it.
		detail += fmt.Sprintf("; run record failed: %v", err)
	}
	return StepOut{Detail: detail, Passed: passed}, nil
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

func (w *Workspace) Rm(ctx context.Context) (string, error) {
	if err := w.allow("Rm"); err != nil {
		return "", err
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
	w.state.GenesisConfigPaths = nil
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
