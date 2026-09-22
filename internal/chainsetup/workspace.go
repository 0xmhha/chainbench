package chainsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The workspace: opening one, and resolving the machine a node runs on.
//
// The state it holds is in state.go, and the provenance of the values a launch
// uses in provenance.go.

type Workspace struct {
	// machines caches per-server opened accesses for the current command, so
	// a step over N nodes dials each machine once.
	machines map[string]*resource.Access
	// ledger is the persisted run record — which machine runs which binary,
	// under which command, as which pid. It is the source of truth for PIDs;
	// node.Record.PID is the in-memory view, synced from here on Open.
	ledger *process.Ledger
	comp   session.Composition
	state  State
	env    func(string) string
	now    func() time.Time
	// wcCache holds the parsed workspace-config for this command, loaded once
	// from state.WorkspaceConfig. nil until first use; the bool records that a
	// load was attempted so a workspace with none is not reloaded per node.
	wcCache  *resource.WorkspaceConfig
	wcLoaded bool
	// driver, when set, replaces every machine's process driver — the seam a
	// test uses to control nodes without an OS process, and a caller uses to
	// route control over another transport.
	driver func() (process.Driver, error)
}

// Open opens (creating if absent) the workspace at dir. now is injected for
// deterministic timestamps; nil uses time.Now.
func Open(dir string, now func() time.Time) (*Workspace, error) {
	w, err := open(dir, now)
	if err != nil {
		return nil, err
	}
	l, err := process.OpenLedger(dir)
	if err != nil {
		return nil, err
	}
	w.ledger = l
	for i, ns := range w.state.Nodes {
		if p, ok := l.Get(string(ns.NodeLabel())); ok {
			w.state.Nodes[i].PID = p.PID
			continue
		}
		// A workspace from before the ledger recorded pids in its own state;
		// seed the ledger once so the record moves without losing a process.
		if ns.PID > 0 {
			_ = l.Record(process.Proc{
				PID: ns.PID, Label: string(ns.NodeLabel()),
				Host: ns.Host, DataDir: ns.DataDir,
			})
		}
	}
	return w, nil
}

func open(dir string, now func() time.Time) (*Workspace, error) {
	if now == nil {
		now = time.Now
	}
	comp, err := session.OpenComposition(dir, now)
	if err != nil {
		return nil, err
	}
	ws := &Workspace{comp: comp, env: os.Getenv, now: now, state: State{Steps: map[string]Step{}}}
	found, err := comp.Load(&ws.state)
	if err != nil {
		return nil, err
	}
	if found && ws.state.FormatVersion != StateFormatVersion {
		return nil, fmt.Errorf(
			"chainsetup: the composition record in %s is format %d, and this build reads format %d — compose it again with `chain new`",
			dir, ws.state.FormatVersion, StateFormatVersion)
	}
	ws.state.FormatVersion = StateFormatVersion
	if ws.state.Steps == nil {
		ws.state.Steps = map[string]Step{}
	}
	return ws, nil
}

// Dir is the workspace's local control directory.
func (w *Workspace) Dir() string { return w.comp.Dir() }

// Acquire takes the workspace's lock for this run, reporting the previous
// holder and what it was. See session.Composition.Acquire.
func (w *Workspace) Acquire(command string) (*session.Held, session.Lock, session.LockState, error) {
	return w.comp.Acquire(command)
}

// Lock reports who holds the workspace without taking it.
func (w *Workspace) Lock() (session.Lock, session.LockState, error) { return w.comp.Lock() }

// SetEnv overrides the environment reader used when resolving a remote target
// (credentials). Nil is ignored; the default is os.Getenv.
func (w *Workspace) SetEnv(fn func(string) string) {
	if fn != nil {
		w.env = fn
	}
}

// SetDriver overrides the process driver every machine of this workspace
// controls its nodes through. Nil is ignored; the default is each machine's
// own process.
func (w *Workspace) SetDriver(fn func() (process.Driver, error)) {
	if fn != nil {
		w.driver = fn
	}
}

// State returns a copy of the current composition state.
func (w *Workspace) State() State { return w.state }

// SetStatePath records where the composition machine is, for the next Save.
//
// It takes the already-joined path rather than a state, because the path is
// what the machine computes and this package's record has no opinion about the
// shape of a machine's tree.
func (w *Workspace) SetStatePath(path string) { w.state.StatePath = path }

// keysBase is where a node's identity files (nodekey, keystore, password)
// live at launch, from the target's point of view: the local key set for a
// local target, or keys/ under the data root for a remote one — where the
// provision step ships them, and where the rendered config and launch argv
// then point. Baking the operator-side path into a remote config was how a
// remote node came to look for its nodekey on a machine it cannot see.
func (w *Workspace) keysBase() string {
	if w.state.Target.IsRemote() {
		return filepath.Join(w.state.Target.DataRoot, "keys")
	}
	return w.state.KeysDir
}

// ResolveTarget builds the live target for a step through the netmap module,
// the one dial-wiring point: the recorded server set, the docker-mode
// translation, and the login rules are bound there identically for every
// consumer, so a multi-step run cannot be half-mapped and this module cannot
// diverge from keyring or anyone else.
func (w *Workspace) ResolveTarget() (*resource.Access, error) {
	return w.opener().Open(w.state.Target)
}

// machineFor opens the machine ns runs on. A node spread across a set names its server-set
// entry and resolves through the netmap module like everything else; a node
// without one runs on the workspace's single target. Opened accesses are
// cached per entry for the life of this command.
func (w *Workspace) machineFor(ns node.Record) (*resource.Access, error) {
	key := ns.Server
	if w.machines == nil {
		w.machines = map[string]*resource.Access{}
	}
	if t, ok := w.machines[key]; ok {
		return t, nil
	}
	var (
		t   *resource.Access
		err error
	)
	if ns.Server == "" {
		t, err = w.ResolveTarget()
	} else {
		t, err = w.opener().Open(resource.Spec{
			Server: ns.Server,
			Host:   ns.Host, DataRoot: w.state.Target.DataRoot,
		})
	}
	if err != nil {
		return nil, err
	}
	if w.driver != nil {
		override, err := w.driver()
		if err != nil {
			return nil, err
		}
		// Copied: the opener may hand the same access to another workspace.
		t = &resource.Access{Spec: t.Spec, DataRoot: t.DataRoot, Files: t.Files, Driver: override}
	}
	w.machines[key] = t
	return t, nil
}

// eachMachine resolves every distinct machine the node table names, in node
// order, and calls fn once per machine with the nodes that live on it.
func (w *Workspace) eachMachine(fn func(t *resource.Access, nodes []node.Record) error) error {
	order := []string{}
	group := map[string][]node.Record{}
	for _, ns := range w.state.Nodes {
		if _, ok := group[ns.Server]; !ok {
			order = append(order, ns.Server)
		}
		group[ns.Server] = append(group[ns.Server], ns)
	}
	for _, key := range order {
		nodes := group[key]
		t, err := w.machineFor(nodes[0])
		if err != nil {
			return err
		}
		if err := fn(t, nodes); err != nil {
			return err
		}
	}
	return nil
}

// opener binds the workspace's recorded server set and docker choice to the
// netmap module's single wiring point.
func (w *Workspace) opener() resource.Opener {
	return resource.Opener{ServerSet: w.state.ServerSet, Docker: w.state.Docker, Env: w.env}
}

// markStep records that step finished, with the detail it reports.
//
// step must be a name one of the two lists declares — UpStepNames for a rung of
// the composition ladder, OpStepNames for an operation on a network that is
// already up. The two go into the same map and a reader picks the subset it
// means, so a name in neither list is recorded and then never read by anything.
// TestRecordedStepsAreDeclared holds every call here to that.
func (w *Workspace) markStep(step, detail string) {
	w.state.Steps[step] = w.comp.StepMark(detail)
}

// MarkStepFailed records that step was reached and did not finish.
//
// It is exported because the failure is noticed by the runner that drives the
// steps, not by the step itself: a step that fails returns an error and never
// reaches its own markStep call.
func (w *Workspace) MarkStepFailed(step string, cause error) {
	if w.state.Steps == nil {
		w.state.Steps = map[string]Step{}
	}
	w.state.Steps[step] = w.comp.StepEnd(w.comp.StepBegin(), "", cause)
}

// Save writes the composition state to the manifest.
func (w *Workspace) Save() error {
	if w.ledger != nil {
		if err := w.ledger.Save(); err != nil {
			return err
		}
	}
	return w.comp.Save(w.state)
}

// recordLaunch enters an already-launched node (a resume reattaching to a live
// pid) in the run ledger and syncs the view. Fresh launches go through
// process.LaunchAndRecord, which launches and records in one step; this records
// a pid the caller already holds.
func (w *Workspace) recordLaunch(i int, pid int, binary string) error {
	spec := process.SpecOf(w.state.Nodes[i])
	spec.Binary = binary
	if err := w.ledger.Record(process.ProcFor(spec, pid)); err != nil {
		return err
	}
	w.state.Nodes[i].PID = pid
	return nil
}

// recordSwap records a node the hardfork swapped in, superseding its prior
// entry so the pid and command that ran before the fork are kept as a revision
// rather than discarded. The old process is already stopped by the swap.
func (w *Workspace) recordSwap(i int, pid int, binary string) error {
	spec := process.SpecOf(w.state.Nodes[i])
	spec.Binary = binary
	if _, _, err := w.ledger.Supersede(process.ProcFor(spec, pid)); err != nil {
		return err
	}
	w.state.Nodes[i].PID = pid
	return nil
}

// clearPID removes a stopped node from the run ledger and syncs the view.
func (w *Workspace) clearPID(i int) {
	w.ledger.Clear(string(w.state.Nodes[i].NodeLabel()))
	w.state.Nodes[i].PID = 0
}

// localHost is the address a locally-composed node is reachable at.
const localHost = "127.0.0.1"

// RPCHost is the address this composition's nodes are reachable at: this
// machine for a local target, the SSH host for a remote one. Ports are the
// same either way — the allocator assigns them on the target.
func (w *Workspace) RPCHost() string {
	if w.state.Target.IsRemote() && w.state.Target.Host != "" {
		return w.state.Target.Host
	}
	return localHost
}

// NodeSet renders the composition as the chain-agnostic node model the rest of
// chainbench consumes (health probes, DSL runs, stop). It is the bridge that
// lets a composed network be used wherever a setup-launched one can: the two
// stacks persist different state, but every consumer downstream of them speaks
// NodeSet.
//
// PIDs are whatever the last lifecycle step recorded, so a node that has not
// been started reports 0 — the same convention as an attached node chainbench
// did not launch.
func (w *Workspace) NodeSet() node.NodeSet {
	host := w.RPCHost()
	// RPCURL is the address the harness dials, so it is the reachable one Health
	// and ChainEndpoints use — a docker node's URL is the mapped one, not the
	// container-internal address. Host keeps the node's own address, which is
	// what a display or record wants.
	ns := node.NodeSet{
		Chain:        w.state.Chain,
		Network:      w.state.Target.Describe(),
		Capabilities: w.state.Capabilities,
		Nodes:        make([]node.Node, 0, len(w.state.Nodes)),
	}

	for _, n := range w.state.Nodes {
		// A node's own recorded host wins: a set-wide pool puts each node on
		// a different address, which the target-level host cannot express.
		nodeHost := n.Host
		if nodeHost == "" {
			nodeHost = host
		}
		ns.Nodes = append(ns.Nodes, node.Node{
			Index:      n.Index,
			Role:       node.Role(n.Role),
			Host:       nodeHost,
			RPCURL:     rpcURLOf(w, n),
			MetricsURL: metricsURLOf(w, n),
			WSURL:      wsURLOf(w, n),
			// The record's embedded Endpoints, whole: copying fields one by one
			// is how the etcd port went missing between the plan and the
			// running network before.
			Ports: n.Endpoints,
			PID:   n.PID,
		})
	}
	return ns
}

// SetBinary records which node binary this workspace runs.
//
// A resume takes one from the command line for a node it is about to relaunch,
// and the launch reads it back from here — so the two have to be the same field
// rather than one passed alongside the other.
func (w *Workspace) SetBinary(path string) { w.state.Binary = path }
