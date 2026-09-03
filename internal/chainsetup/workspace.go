// Package netcompose is the shared core behind the `chainbench net` step
// commands and their MCP mirrors: it composes a chain network for testing one
// customizable step at a time (keys, allocate, genesis, config, provision,
// init, start/stop, logs, test), persisting the accumulating state in a local
// data directory so steps run independently, re-run, and are inspectable.
//
// Two planes are kept separate. The CONTROL plane — the composition state in
// workspace.json (chain, keys, placements, node table, step-tracking) — always
// lives locally on the operator's resource. The DATA plane — genesis, configs,
// datadirs, logs — lives on a Target (this machine's filesystem, or a remote
// SSH host); see target.go. Step functions use the Target's FileStore/Driver and
// never branch on local vs remote.
//
// Persistence belongs to core/session (Composition — the long-lived
// environment mode); this package owns only the domain state and the step
// functions. The CLI and MCP surfaces are thin wrappers over the app layer,
// which calls the step functions here, so both drive the exact same behavior.
package chainsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// nodeScopeRE matches a per-node override scope key ("node1", "node12"), the
// storage form both config and launch overrides share.
var nodeScopeRE = regexp.MustCompile(`^node[1-9][0-9]*$`)

// Step is a completed composition step (persistence model owned by session).
type Step = session.Step

// State is the persisted composition state accumulated across step commands. It
// grows as later steps land (placements, genesis path, node table); each field
// is optional so a partially-composed workspace round-trips. It holds no
// secrets — a server-set placement reads its login from the server-set file at
// resolve time, and a directly named target reads the environment.
type State struct {
	Chain string `json:"chain"`
	// ManifestPath and TemplatePath name an external, project-supplied chain
	// manifest. When set they win over Chain, so a workspace composed for a
	// project's own chain resolves the same plugin on every later step.
	ManifestPath string          `json:"manifestPath,omitempty"`
	TemplatePath string          `json:"templatePath,omitempty"`
	Binary       string          `json:"binary,omitempty"`
	KeysDir      string          `json:"keysDir,omitempty"`
	Validators   int             `json:"validators,omitempty"`
	Target       resource.Spec   `json:"target"`
	GenesisPath  string          `json:"genesisPath,omitempty"`
	Nodes        []node.Record   `json:"nodes,omitempty"`
	Steps        map[string]Step `json:"steps"`
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
	// LegacyServerSet reads the field's pre-rename key so a workspace composed
	// before the rename keeps its recorded path. It is migrated into ServerSet
	// on open and never written back.
	LegacyServerSet string `json:"serverConfig,omitempty"`
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
	// every node, a role ("bp"/"en"/"boot") for that role, "node<N>" for one.
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
	if err := comp.Load(&ws.state); err != nil {
		return nil, err
	}
	if ws.state.Steps == nil {
		ws.state.Steps = map[string]Step{}
	}
	if ws.state.ServerSet == "" && ws.state.LegacyServerSet != "" {
		ws.state.ServerSet = ws.state.LegacyServerSet
	}
	ws.state.LegacyServerSet = ""
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
// node's spec: the "all" scope first, then that node's own scope (so a node
// override wins). Each entry is a dot-path "key=value"; an unknown key or a
// malformed entry is an error, never a silent no-op.
func (w *Workspace) applyConfigOverrides(spec *nodeconfig.Spec, index int) error {
	for _, kv := range w.configOverridesFor(index) {
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
// most-general-first: "all" then "node<N>" (node wins). It is the single source
// of which overrides shape a node's config — applyConfigOverrides applies them
// and the config step records them as provenance, so the two never diverge. A
// node reads only its own "node<N>" scope, so one node's override never leaks
// into another's config.
func (w *Workspace) configOverridesFor(index int) []string {
	var out []string
	for _, scope := range []string{"all", fmt.Sprintf("node%d", index)} {
		out = append(out, w.state.ConfigSet[scope]...)
	}
	return out
}

// ConfigProvenance is one node's config record: the overrides applied to it and
// the checksum ("sha256:<hex>") of the config that resulted.
type ConfigProvenance struct {
	Node      int      `json:"node"`
	Overrides []string `json:"overrides,omitempty"`
	Checksum  string   `json:"checksum"`
}

// recordLaunchSet stores launch-argv overrides under a scope ("all", a role,
// or "node<N>"), appending to what that scope already holds so repeated calls
// accumulate. Each entry is validated as a launch override up front, so a bad
// knob is refused where it is set rather than at argv assembly.
func (w *Workspace) recordLaunchSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
	}
	if !validLaunchScope(scope) {
		return fmt.Errorf("launch scope %q must be \"all\", a role (bp|validator, en|endpoint, boot), or \"node<N>\"", scope)
	}
	if _, err := ParseOverrides(sets); err != nil {
		return err
	}
	if w.state.LaunchSet == nil {
		w.state.LaunchSet = map[string][]string{}
	}
	w.state.LaunchSet[scope] = append(w.state.LaunchSet[scope], sets...)
	return nil
}

// launchOverridesFor returns the launch-argv overrides for one node, folding the
// scopes that apply to it most-general-first: "all", then the node's role, then
// the node itself. Later entries win at assembly (nodeconfig.Argv override
// layer is last-write-wins), so a node override beats a role override beats all.
func (w *Workspace) launchOverridesFor(role string, index int) []string {
	var out []string
	roleScope := ""
	if r, err := node.NormalizeRole(role); err == nil {
		roleScope = string(r)
	}
	for _, scope := range []string{"all", roleScope, fmt.Sprintf("node%d", index)} {
		if scope == "" {
			continue
		}
		out = append(out, w.state.LaunchSet[scope]...)
	}
	return out
}

// validLaunchScope reports whether scope is a launch scope the workspace applies:
// "all", a role token, or "node<N>". A role is stored normalized (bp/en/boot).
func validLaunchScope(scope string) bool {
	switch scope {
	case "all", string(node.RoleBP), string(node.RoleValidator),
		string(node.RoleEN), string(node.RoleEndpoint), string(node.RoleBoot):
		return true
	}
	return nodeScopeRE.MatchString(scope)
}

// recordConfigSet stores config overrides under a scope ("all" or "node<N>"),
// appending to what that scope already holds so repeated --set calls accumulate.
// It validates each entry against the knob contract up front, so a bad override
// is refused at the point it is set rather than at render.
func (w *Workspace) recordConfigSet(scope string, sets []string) error {
	if len(sets) == 0 {
		return nil
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
	w.state.ConfigSet[scope] = append(w.state.ConfigSet[scope], sets...)
	return nil
}

// markStep records that step ran with detail, stamping the completion time.
func (w *Workspace) markStep(step, detail string) {
	w.state.Steps[step] = w.comp.StepMark(detail)
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

// recordLaunch enters a launched node in the run ledger and syncs the view.
func (w *Workspace) recordLaunch(i int, pid int, binary string) error {
	ns := w.state.Nodes[i]
	if err := w.ledger.Record(process.Proc{
		PID: pid, Label: string(ns.NodeLabel()), Binary: filepath.Base(binary),
		Command: strings.Join(append([]string{binary}, ns.Args...), " "),
		Host:    nodeHost(ns), DataDir: ns.DataDir,
	}); err != nil {
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
			Index:  n.Index,
			Role:   node.Role(n.Role),
			Host:   nodeHost,
			RPCURL: fmt.Sprintf("http://%s:%d", nodeHost, n.HTTP),
			// The record's embedded Endpoints, whole: copying fields one by one
			// is how the etcd port went missing between the plan and the
			// running network before.
			Ports: n.Endpoints,
			PID:   n.PID,
		})
	}
	return ns
}
