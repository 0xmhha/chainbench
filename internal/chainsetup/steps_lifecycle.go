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
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
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

func (w *Workspace) network() (nodeconfig.Network, error) {
	p, err := w.plugin()
	if err != nil {
		return nodeconfig.Network{}, err
	}
	var chainID int64
	if w.state.Request != nil {
		chainID = w.state.Request.ChainID
	}
	return nodeconfig.NetworkOf(p, chainID), nil
}

// checkUniformNetworkID holds the assembled argv to one devp2p id.
//
// network() decides the number and every assembler takes it from there, so in
// principle they cannot disagree. This reads what was actually assembled
// instead, because the ways they can disagree are the ways that do not go
// through network(): a launch override that names the flag on one scope, or a
// fourth assembler written later. Both leave the input agreeing with itself.
//
// Nodes with no argv yet are skipped rather than failed: the check runs after
// each step that assembles, and a composition part-way through has some.
func (w *Workspace) checkUniformNetworkID() error {
	argv := map[string][]string{}
	for _, ns := range w.state.Nodes {
		if len(ns.Args) == 0 {
			continue
		}
		argv[string(ns.NodeLabel())] = ns.Args
	}
	if len(argv) == 0 {
		return nil
	}
	if err := nodeconfig.ValidateUniformNetworkID(argv); err != nil {
		return lifecycle.Mark(errBuildSplitNetwork, err)
	}
	return nil
}

// pluginFor resolves the chain one node runs: the one recorded for its binary
// when that binary is a different chain, otherwise the composition's.
//
// It is the third of these — binaryFor, genesisFor, pluginFor — and they ask the
// same question: which of this network's builds is this node. Keeping them the
// same shape is what stops one of them answering differently from the others.
// network is what every node of this composition shares, whichever binary it
// runs: the chain id it was composed with and the devp2p id that follows it.
//
// Every caller that builds a node's configuration takes it from here, so the
// config writer and the argv assembler cannot answer differently. They used to:
// the config writer read the node's own plugin and the assembler the
// composition's, which wrote one devp2p id into a successor's config file and
// passed another on its command line.
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

// errOpSomeStillUp: not every node came down. Every node is attempted before
// this is raised, so the message names each one that did not.
var errOpSomeStillUp = errors.New("not every node came down")

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
			return lifecycle.Mark(errInitTargetUnable,
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
				return nil, lifecycle.Mark(errInitGenesisUnreadable,
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
				return lifecycle.Mark(errInitDatadir,
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

// LaunchPlan is the checks a launch makes before it starts anything, and the
// phases the family says to start in.
//
// Split out of Start so the state that prepares and the states that walk the
// phases can be separate states. A family decides how many phases there are —
// wbft declares one, a poa network declares a boot plus one join per producer —
// so the count is not this package's to know.
func (w *Workspace) LaunchPlan(ctx context.Context, binaryArg string) (string, []registry.Phase, error) {
	if err := w.require("start"); err != nil {
		return "", nil, err
	}
	p, err := w.plugin()
	if err != nil {
		return "", nil, err
	}
	if len(w.state.Nodes) == 0 {
		return "", nil, fmt.Errorf("chainsetup: start: no node table — run `chain place` first")
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return "", nil, err
	}
	roles := make([]node.Role, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		roles = append(roles, node.Role(ns.Role))
	}
	phases := p.Family().BringUpPhases(roles)
	if err := w.checkUnmanaged(ctx, bin); err != nil {
		return "", nil, err
	}
	if err := w.checkPaths(ctx, bin); err != nil {
		return "", nil, err
	}
	return bin, phases, nil
}

// StartPhase launches one phase and says how many nodes it started.
func (w *Workspace) StartPhase(ctx context.Context, bin string, phase registry.Phase) (int, error) {
	p, err := w.plugin()
	if err != nil {
		return 0, err
	}
	// With accounts: a producer unlocks the account its keystore holds, which is
	// not always the address its nodekey derives.
	keys, err := preset.LoadKeyPresetWithAccounts(w.state.KeysDir)
	if err != nil {
		return 0, fmt.Errorf("chainsetup: start: %w", err)
	}
	return w.startPhase(ctx, p, keys, bin, phase)
}

// RunPhaseActions runs what a phase declares after its nodes are up.
func (w *Workspace) RunPhaseActions(ctx context.Context, bin string, phase registry.Phase) error {
	return w.runPhaseActions(ctx, bin, phase)
}

// FinishLaunch records the binary the network runs and the run itself.
func (w *Workspace) FinishLaunch(ctx context.Context, bin string, started int) (string, error) {
	w.state.Binary = bin
	detail := fmt.Sprintf("%d node(s) started (%d already running)", started, len(w.state.Nodes)-started)
	w.markStep("start", detail)
	rec, err := w.machineFor(w.state.Nodes[0])
	if err != nil {
		return "", err
	}
	if dir, rerr := w.recordRun(ctx, rec, bin); rerr == nil {
		detail += fmt.Sprintf("; run recorded at %s", dir)
	} else {
		// The record must never take the network it records down with it.
		detail += fmt.Sprintf("; run record failed: %v", rerr)
	}
	return detail, nil
}

// Start launches every stopped node. Argv comes from the launchopts step when
// it ran; otherwise it is assembled here through the same single site
// (nodeconfig.Argv) with no overrides.
func (w *Workspace) Start(ctx context.Context, binaryArg string) (StepOut, error) {
	bin, phases, err := w.LaunchPlan(ctx, binaryArg)
	if err != nil {
		return StepOut{}, err
	}
	started := 0
	for _, phase := range phases {
		launched, perr := w.StartPhase(ctx, bin, phase)
		if perr != nil {
			return StepOut{}, perr
		}
		started += launched
		if len(phase.Actions) > 0 {
			if aerr := w.RunPhaseActions(ctx, bin, phase); aerr != nil {
				return StepOut{}, aerr
			}
		}
	}
	detail, err := w.FinishLaunch(ctx, bin, started)
	if err != nil {
		return StepOut{}, err
	}
	return StepOut{Detail: detail}, nil
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
	stopped, attempts, errs := w.stopNodes(ctx, func(node.Record) bool { return true })
	if len(errs) > 0 {
		return "", lifecycle.Mark(errOpSomeStillUp,
			fmt.Errorf("chainsetup: stop: %d of %d node(s) stopped; %s",
				stopped, attempts, strings.Join(errs, "; ")))
	}
	detail := fmt.Sprintf("%d node(s) stopped", stopped)
	w.markStep("stop", detail)
	return detail, nil
}

// RollBackLaunch stops the nodes a failed launch started: every node that has a
// recorded pid now and was not running before the launch began.
//
// A launch that dies at node13 has already started node1..node12, and their
// pids are in the record. Nothing else takes them down — the run that failed
// hands back no network to stop — so they kept their ports and datadirs locked
// until someone noticed, and the next run on the same servers could not
// compose. Nodes that were running before the launch are left alone: they are
// not this launch's to stop.
func (w *Workspace) RollBackLaunch(ctx context.Context, wasRunning map[int]bool) (string, error) {
	stopped, attempts, errs := w.stopNodes(ctx, func(ns node.Record) bool { return !wasRunning[ns.Index] })
	if len(errs) > 0 {
		return "", fmt.Errorf("chainsetup: roll back launch: %d of %d node(s) stopped; %s",
			stopped, attempts, strings.Join(errs, "; "))
	}
	return fmt.Sprintf("%d node(s) this launch started were stopped", stopped), nil
}

// RunningNodes is the index of every node with a recorded pid.
func (w *Workspace) RunningNodes() map[int]bool {
	running := map[int]bool{}
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 {
			running[ns.Index] = true
		}
	}
	return running
}

// stopNodes stops, concurrently, every running node pick selects, and clears
// the pid of each one that went down. It returns how many stopped, how many
// were attempted, and one message per node that did not.
func (w *Workspace) stopNodes(ctx context.Context, pick func(node.Record) bool) (int, int, []string) {
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
		if ns.PID <= 0 || !pick(ns) {
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
	return stopped, attempts, errs
}

// Rm removes the composed data plane — node datadirs, configs, genesis, logs,
// and the composition's own directories — on every machine it was placed on.
// Running nodes must be stopped first.
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
	layout, err := w.layout()
	if err != nil {
		return "", fmt.Errorf("chainsetup: rm: %w", err)
	}
	genesisDone := map[string]bool{}
	for _, ns := range w.state.Nodes {
		acc, err := w.machineFor(ns)
		if err != nil {
			return "", fmt.Errorf("chainsetup: rm: node%d: %w", ns.Index, err)
		}
		for _, p := range []string{ns.DataDir, ns.ConfigPath, ns.LogPath} {
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
			// The composition's own directories go last, once the files in
			// them are gone: a composition isolated by id left an empty
			// node/<id>, runtime/<id> and logs/<id> on every server it ever
			// ran on, and a data root that saw a few hundred runs filled up.
			for _, p := range layout.CompositionDirs() {
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

// InitFailure is which of the init stage's failures this error is.
//
// The busy-port answer is the launch's state, not one of this block's. The
// check is the same one the launch makes, asked here before anything is
// written so that the refusal can name the ports and the host instead of
// leaving it to the binary to say "datadir already used"; what failed is the
// ports the launch needs, and it keeps that name whoever noticed.
//
// What still reaches the default is the binary's own refusal, and the two
// preconditions a bare `chain init` can still hit — inside a composition the
// transition table is what makes those unreachable.
func InitFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errInitTargetUnable):
		return lifecycle.ChainInitNodesFailTargetUnable
	case errors.Is(err, errInitGenesisUnreadable):
		return lifecycle.ChainInitNodesFailGenesisUnreadable
	case errors.Is(err, errInitDatadir):
		return lifecycle.ChainInitNodesFailDatadir
	case errors.Is(err, errLaunchPortBusy):
		return lifecycle.ChainLaunchNodesFailPortBusy
	}
	return lifecycle.FailStageUnclassified
}

// StopFailure is which state a failure of taking the network down is.
//
// Every node is attempted before the partial is raised, so the one state it has
// means "not all of them", and the message names which.
func StopFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errOpSomeStillUp):
		return lifecycle.ChainOpStopNodesFailSomeStillUp
	case errors.Is(err, errOpNoSuchNode):
		return lifecycle.ChainOpFailNoSuchNode
	case errors.Is(err, errOpPrecondition):
		return lifecycle.ChainOpFailPrecondition
	}
	return lifecycle.FailStageUnclassified
}
