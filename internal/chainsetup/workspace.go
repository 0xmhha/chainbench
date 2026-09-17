// Workspace and the state it records. The package comment is in doc.go.
//
// Persistence belongs to core/session (Composition, the long-lived environment
// mode); this file owns the domain state and the accessors the steps use.

package chainsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// StateFormatVersion is the shape of the composition record this build reads
// and writes.
//
// It exists because the record had no version at all. A field could be renamed
// or its meaning changed and an older record would still decode — the missing
// field reads as a zero, and a zero is a legitimate value for most of them, so
// the composition came up describing itself wrongly rather than refusing. There
// is no migration path on purpose: this track keeps one current shape, and a
// record from another version is a workspace to compose again.
//
// Raise it whenever a field changes meaning, is removed, or is renamed.
const StateFormatVersion = 1

// Step is a completed composition step (persistence model owned by session).
type Step = session.Step

// State is the persisted composition state accumulated across step commands. It
// grows as later steps land (placements, genesis path, node table); each field
// is optional so a partially-composed workspace round-trips. It holds no
// secrets — a server-set placement reads its login from the server-set file at
// resolve time, and a directly named target reads the environment.
type State struct {
	// FormatVersion is the shape this record was written in. A build reads only
	// the version it writes: this track removes old-format readers rather than
	// keeping one per era, so a record from another version is refused by name
	// instead of being half-understood.
	//
	// It is first because it decides whether the rest means anything.
	FormatVersion int `json:"formatVersion"`

	Chain string `json:"chain"`
	// CompositionID is a stable identifier for this composition, set once at
	// `new` and kept across resume and binary swap. It names the composition's
	// own node data directories, runtime files, and logs so two compositions on
	// one data root do not collide; it is derived from the workspace directory,
	// not the run time or a pid, so a resume keeps the same one.
	CompositionID string `json:"compositionId,omitempty"`
	// ManifestPath and TemplatePath name an external, project-supplied chain
	// manifest. When set they win over Chain, so a workspace composed for a
	// project's own chain resolves the same plugin on every later step.
	ManifestPath string `json:"manifestPath,omitempty"`
	TemplatePath string `json:"templatePath,omitempty"`
	Binary       string `json:"binary,omitempty"`
	KeysDir      string `json:"keysDir,omitempty"`
	// BPCount is how many bp nodes the placement resolved to. The genesis step
	// sizes the validator set from it: the producers ARE the validator set, and
	// counting the placements rather than the request is what makes a topology
	// decide it.
	BPCount     int           `json:"bp,omitempty"`
	Target      resource.Spec `json:"target"`
	GenesisPath string        `json:"genesisPath,omitempty"`
	// LaunchInputs is what each launch input hashed to when this workspace
	// wrote it, keyed by path on the target.
	//
	// It exists so deploy can tell "the file is there" from "the file is the
	// one we built". Without it deploy checked only for presence and reported
	// the inputs "reused, not rewritten", which is true of a genesis someone
	// edited and of a config left by a previous composition — and the nodes
	// then launch from it. A hash costs nothing to record and turns a silent
	// wrong launch into a refusal that names the file.
	LaunchInputs map[string]string `json:"launchInputs,omitempty"`
	Nodes        []node.Record     `json:"nodes,omitempty"`
	Steps        map[string]Step   `json:"steps"`
	// Peering is the peer graph the composition wires ("mesh" default,
	// "proxied" for bp <-> pn <-> en). Empty means mesh, so a workspace written
	// before the field keeps the graph it was composed with.
	Peering string `json:"peering,omitempty"`
	// Bootnode is the 1-based index of the topology's bootnode, or 0 when the
	// layout came from plain counts. Informational: every composed node lists
	// every other as a static node, so peering does not depend on it.
	Bootnode int `json:"bootnode,omitempty"`
	// PortSource names where the port plan came from (a server set entry,
	// or the built-in defaults), so an operator reading the state never has to
	// guess why a node listens where it does.
	PortSource string `json:"portSource,omitempty"`
	// ServerSet is the server-set file the placement came from, recorded so
	// later steps resolve the same file — and, in docker mode, find the
	// localmap next to it.
	ServerSet string `json:"serverSet,omitempty"`
	// WorkspaceConfig is the environment file (--workspace-config) this
	// composition was set up with, recorded so later steps and a resume resolve
	// portable file references under the same data root and purpose directories.
	WorkspaceConfig string `json:"workspaceConfig,omitempty"`
	// Docker records that this composition treats its servers as local docker
	// containers: the harness's own dials are translated through the localmap
	// next to ServerSet. It is recorded once at `chain new --docker` so a
	// multi-step run cannot be half-mapped, and it never changes what is
	// composed — genesis, static-nodes and the node table keep real addresses.
	Docker bool `json:"docker,omitempty"`
	// Capabilities is what the composed network advertises to capability-gated
	// test cases (chain manifest + ws + delayed-fork markers + overlay claims).
	// The genesis step derives it, since that is where the customizations that
	// change what the network can do are applied.
	Capabilities []string `json:"capabilities,omitempty"`
	// ConfigSet holds per-scope config-knob overrides, keyed by scope: "all"
	// for every node, "node<N>" for one. Each value is a list of dot-path
	// "key=value" strings applied at config render (all first, then the node's
	// own — node wins). Stored key-agnostically so the surface and the storage
	// do not change when the set of overridable knobs grows.
	ConfigSet map[string][]string `json:"configSet,omitempty"`
	// ConfigProvenance records, per node, the overrides that shaped its config
	// and the checksum of the config that resulted, so a run can show which
	// config each node got and that it read back intact. The config step
	// recomputes it each time it runs — a fresh config is a new revision.
	ConfigProvenance []ConfigProvenance `json:"configProvenance,omitempty"`
	// LaunchSet holds per-scope launch-argv overrides, keyed by scope: "all" for
	// every node, a role ("bp"/"en") for that role, "node<N>" for one.
	// Each value is a list of "key" (boolean flag) or "key=value" applied at
	// argv assembly, most-general-first (all, then role, then node — node wins).
	LaunchSet map[string][]string `json:"launchSet,omitempty"`
	// Binaries maps a per-node binary name (as topology entries reference it)
	// to its resolved path. Empty means every node runs the single Binary. It
	// is how one network runs mixed builds concurrently.
	Binaries map[string]string `json:"binaries,omitempty"`
	// Request is what `chain up` was asked to compose, recorded at the new
	// step so a run that dies before the results exist can be resumed from
	// what it was asked, not re-asked. Its DataDir is left empty: the
	// workspace's location is where this file is. It is the one fact of a
	// composition that is otherwise nowhere on disk (F1).
	Request *NetUpIn `json:"request,omitempty"`
}

// Workspace is an open composition workspace: the session-owned persistence
// (control directory + manifest + step stamps) plus the netcompose domain
// state the steps accumulate.
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

// resolveTarget builds the live target for a step through the netmap module,
// the one dial-wiring point: the recorded server set, the docker-mode
// translation, and the login rules are bound there identically for every
// consumer, so a multi-step run cannot be half-mapped and this module cannot
// diverge from keyring or anyone else.
func (w *Workspace) resolveTarget() (*resource.Access, error) {
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
		t, err = w.resolveTarget()
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

// applyConfigOverrides applies the workspace's config-knob overrides to one
// node's spec, most-general-first, so the narrowest scope wins. Each entry is a
// dot-path "key=value"; an unknown key or a malformed entry is an error, never
// a silent no-op.
func (w *Workspace) applyConfigOverrides(spec *nodeconfig.Spec, role node.Role, index int) error {
	for _, kv := range w.configOverridesFor(role, index) {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			return fmt.Errorf("config override %q must be key=value", kv)
		}
		if err := nodeconfig.ApplyConfigOverride(spec, key, value); err != nil {
			return err
		}
	}
	return nil
}

// configOverridesFor returns the config overrides that apply to one node,
// most-general-first: "all", then the node's role, then the node itself, so the
// narrowest scope wins.
//
// It is the single source of which overrides shape a node's config —
// applyConfigOverrides applies them and the config step records them as
// provenance, so the two never diverge. A node reads only its own "node<N>"
// scope, so one node's override never leaks into another's config.
func (w *Workspace) configOverridesFor(role node.Role, index int) []string {
	var out []string
	for _, scope := range node.ScopeFor(role, index) {
		out = append(out, w.state.ConfigSet[scope]...)
	}
	return out
}

// ConfigProvenance is one REVISION of one node's config: the overrides applied to
// it, the checksum ("sha256:<hex>") of the config that resulted, and when. Fixture
// names a mid-test config swap (config-<test-purpose>), empty for the initial
// compose.
//
// Revisions accumulate. The word was already in this file's comments — "a fresh
// config is a new revision" — while the writer replaced the node's entry, so a
// swapNode erased the config the node had been composed with. What a run is asked
// afterwards is "which config did node N have, and since when", and one entry with
// no time answers neither half. A fresh compose clears the list; a swap appends.
type ConfigProvenance struct {
	Node      int      `json:"node"`
	Fixture   string   `json:"fixture,omitempty"`
	Overrides []string `json:"overrides,omitempty"`
	Checksum  string   `json:"checksum"`
	// At is when this revision was applied (RFC3339, UTC). The requirement asks
	// for the node AND the time, and without it two revisions of one node's
	// config cannot be ordered — which is the only question a swap makes anyone
	// ask.
	At string `json:"at,omitempty"`
}

// recordLaunchSet stores launch-argv overrides under a scope ("all", a role,
// or "node<N>"). Each entry is validated as a launch override up front, so a
// bad knob is refused where it is set rather than at argv assembly.
//
// A scope holds one entry per key. Setting a key it already has replaces that
// entry where it stands rather than appending a second one: argv assembly is
// last-write-wins, so two entries for one key mean the same node either way,
// and the record is what says what was ASKED for. It used to append
// unconditionally, so composing the same declaration twice over one workspace
// wrote the knob twice and a workspace reused all week grew a line per run.
// Replacing in place keeps the order a reader sees stable across runs.
func (w *Workspace) recordLaunchSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !node.ValidScope(scope) {
		return fmt.Errorf("launch scope %q must be %s", scope, node.ScopeWords())
	}
	if _, err := ParseOverrides(sets); err != nil {
		return err
	}
	if w.state.LaunchSet == nil {
		w.state.LaunchSet = map[string][]string{}
	}
	w.state.LaunchSet[scope] = holdOnePerKey(w.state.LaunchSet[scope], sets)
	return nil
}

// holdOnePerKey folds new overrides into what a scope already holds, keeping
// one entry per key. A key already there is replaced where it stands; a new one
// is appended.
//
// An override is "key=value" or a bare key, and the key is the text before the
// first "=" — the same split ParseOverrides makes, so the two agree on what
// counts as one key. Replacing in place rather than appending keeps the order a
// reader sees stable no matter how many times a workspace is recomposed.
func holdOnePerKey(held, sets []string) []string {
	at := make(map[string]int, len(held))
	for i, kv := range held {
		key, _, _ := strings.Cut(kv, "=")
		at[key] = i
	}
	for _, kv := range sets {
		key, _, _ := strings.Cut(kv, "=")
		if j, ok := at[key]; ok {
			held[j] = kv
			continue
		}
		at[key] = len(held)
		held = append(held, kv)
	}
	return held
}

// launchOverridesFor returns the launch-argv overrides for one node, folding the
// scopes that apply to it most-general-first: "all", then the node's role, then
// the node itself. Later entries win at assembly (nodeconfig.Argv override
// layer is last-write-wins), so a node override beats a role override beats all.
func (w *Workspace) launchOverridesFor(role string, index int) []string {
	var out []string
	for _, scope := range node.ScopeFor(node.Role(role), index) {
		out = append(out, w.state.LaunchSet[scope]...)
	}
	return out
}

// recordConfigSet stores config overrides under a scope, one entry per key, the
// same way recordLaunchSet does and for the same reason: render is
// last-write-wins, so a second entry for one key means the same node either way
// and only makes the record grow every time the workspace is recomposed.
//
// Both halves are validated up front, so a bad override is refused where it is
// set rather than at render. The scope was not checked at all before: a typo
// stored values under a key nothing reads, and the node it was meant for came
// up with a config that silently lacked them.
func (w *Workspace) recordConfigSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !node.ValidScope(scope) {
		return fmt.Errorf("config scope %q must be %s", scope, node.ScopeWords())
	}
	var probe nodeconfig.Spec
	for _, kv := range sets {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			return fmt.Errorf("config override %q must be key=value", kv)
		}
		if err := nodeconfig.ApplyConfigOverride(&probe, key, value); err != nil {
			return err
		}
	}
	if w.state.ConfigSet == nil {
		w.state.ConfigSet = map[string][]string{}
	}
	w.state.ConfigSet[scope] = holdOnePerKey(w.state.ConfigSet[scope], sets)
	return nil
}

// markStep records that step ran with detail, stamping the completion time.
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
	// and NetEndpoints use — a docker node's URL is the mapped one, not the
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
			// The record's embedded Endpoints, whole: copying fields one by one
			// is how the etcd port went missing between the plan and the
			// running network before.
			Ports: n.Endpoints,
			PID:   n.PID,
		})
	}
	return ns
}
