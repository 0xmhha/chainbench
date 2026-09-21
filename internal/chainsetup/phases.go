package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/inspector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/preset"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Bring-up phases: the order a consensus family declares, and the actions that
// run between them.
//
// A wbft network declares one phase and the loop is the plain one. A wemix
// network starts its producer alone so the etcd cluster can form, then joins
// the rest one at a time — launching everything at once produced a network that
// came up and never agreed on anything.

// The kinds of failure the launch has. This is the block with the most detail
// states and the most ways to fail, because it is the one stage whose shape the
// family decides: a wbft network declares one phase, a poa network starts its
// producer alone to let the etcd cluster form and then joins the rest one at a
// time.
var (
	// errLaunchNoBinary: the binary is not set, not on the target, or not on
	// the target's PATH.
	errLaunchNoBinary = errors.New("the node binary is not on the target")
	// errLaunchPortBusy: a port the plan needs is already taken.
	errLaunchPortBusy = errors.New("a planned port is in use")
	// errLaunchOccupied: the binary is already running on the machine outside
	// this workspace.
	errLaunchOccupied = errors.New("the machine is already running this binary")
	// errLaunchNoKeystore: a producer has no keystore file to unlock.
	errLaunchNoKeystore = errors.New("a node has no keystore file")
	// errLaunchPhaseEmpty: a phase names actions and launched no node to run
	// them on.
	errLaunchPhaseEmpty = errors.New("a phase has nowhere to run its actions")
)

func (w *Workspace) startPhase(ctx context.Context, p registry.ChainPlugin, keys preset.Key, bin string, phase registry.Phase) (int, error) {
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
			args, err := nodeconfig.Argv(process.NodeConfig(np, keys, spec, w.state.KeysDir, staticNodes))
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
		return ofKind(errLaunchPhaseEmpty,
			fmt.Errorf("chainsetup: start: phase %q names actions but launched no node to run them on", phase.Name))
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
	return "", ofKind(errLaunchNoKeystore,
		fmt.Errorf("chainsetup: start: node%d has no keystore file in %s", index, dir))
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
	// Whether the binary was among the problems, and whether anything else was.
	// The combined message lists everything; the kind can only name one thing,
	// so it names the binary when that is the whole of it and says nothing when
	// the workspace is missing more than that — a state that said "no binary"
	// over a missing genesis would hide the bigger problem.
	binaryMissing, otherMissing := false, false
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
			binaryMissing = true
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
			otherMissing = true
		}
	}
	if len(lines) == 0 {
		return nil
	}
	err := fmt.Errorf("chainsetup: start: %d thing(s) the launch needs are missing on the target:\n%s\nrun the earlier steps (`chain genesis`, `chain config`, `chain init`) or check --binary",
		len(lines), strings.Join(uniq(lines), "\n"))
	if binaryMissing && !otherMissing {
		return ofKind(errLaunchNoBinary, err)
	}
	return err
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
		return ofKind(errLaunchNoBinary, fmt.Errorf("binary: none is set"))
	}
	if strings.ContainsRune(bin, '/') {
		ok, err := t.Files.Exists(ctx, bin)
		if err != nil {
			return fmt.Errorf("binary %s: %v", bin, err)
		}
		if !ok {
			return ofKind(errLaunchNoBinary, fmt.Errorf("binary %s: not on the target", bin))
		}
		return nil
	}
	path, ok, err := inspector.OnPath(ctx, t.Runner, bin)
	if err != nil {
		return fmt.Errorf("binary %s: %v", bin, err)
	}
	if !ok {
		return ofKind(errLaunchNoBinary,
			fmt.Errorf("binary %s: not on the target's PATH (name it in a workspace-config, or pass --binary with a path)", bin))
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
