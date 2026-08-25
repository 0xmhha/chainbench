// Package netcompose is the shared core behind the `chainbench net` step
// commands and their MCP mirrors: it composes a chain network for testing one
// customizable step at a time (keys, allocate, genesis, config, provision,
// init, start/stop, logs, test), persisting the accumulating state in a local
// data directory so steps run independently, re-run, and are inspectable.
//
// Two planes are kept separate. The CONTROL plane — the composition state in
// workspace.json (chain, keys, placements, node table, step-tracking) — always
// lives locally on the operator's machine. The DATA plane — genesis, configs,
// datadirs, logs — lives on a Target (this machine's filesystem, or a remote
// SSH host); see target.go. Step functions use the Target's FileStore/Driver and
// never branch on local vs remote.
//
// Persistence belongs to core/session (Composition — the long-lived
// environment mode); this package owns only the domain state and the step
// functions. The CLI and MCP surfaces are thin wrappers over the app layer,
// which calls the step functions here, so both drive the exact same behavior.
package netcompose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
	netmapmod "github.com/0xmhha/chainbench/internal/netmap"
)

// Step is a completed composition step (persistence model owned by session).
type Step = session.Step

// NodeState is one composed node's resolved assignment: its role, target-side
// paths, allocated ports, the assembled launch argv (once `launchopts` ran),
// and the live PID (once `start` ran; 0 = stopped).
type NodeState struct {
	Index int `json:"index"`
	// Label is the node's identity: the name its datadir, config and log file
	// carry, and the name an operator uses to address it. It is stored rather
	// than derived so that a name, once given, survives — the placement type
	// this replaced also carried a name, and it was thrown away.
	//
	// A workspace written before this field falls back to the conventional
	// label for its index, so nothing has to be migrated.
	Label string `json:"label,omitempty"`
	Role  string `json:"role"`
	// SyncMode is the geth sync mode this node's config renders. Validators are
	// always "full" — they must hold full state to seal — while an endpoint may
	// be switched to "snap" or "archive" so a large-gap re-sync exercises that
	// path. Empty means the config's own default.
	SyncMode string `json:"syncMode,omitempty"`
	// Host is the address this node is reachable at. It comes from the
	// allocator, so a remote placement records the server's address rather than
	// this machine's.
	Host       string `json:"host,omitempty"`
	DataDir    string `json:"dataDir"`
	ConfigPath string `json:"configPath"`
	LogPath    string `json:"logPath"`
	// Endpoints is embedded rather than copied field by field: its keys inline
	// into this object, so the persisted shape does not change, and a port can
	// no longer be dropped in a conversion. That is how the etcd port went
	// missing between the plan and the running network, and the first attempt
	// at restoring it dropped the port again in one of the three copies.
	node.Endpoints
	Args []string `json:"args,omitempty"`
	PID  int      `json:"pid,omitempty"`
}

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
	Target       machine.Spec    `json:"target"`
	GenesisPath  string          `json:"genesisPath,omitempty"`
	Nodes        []NodeState     `json:"nodes,omitempty"`
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
	// next to ServerSet. It is recorded once at `net new --docker` so a
	// multi-step run cannot be half-mapped, and it never changes what is
	// composed — genesis, static-nodes and the node table keep real addresses.
	Docker bool `json:"docker,omitempty"`
	// Capabilities is what the composed network advertises to capability-gated
	// test cases (chain manifest + ws + delayed-fork markers + overlay claims).
	// The genesis step derives it, since that is where the customizations that
	// change what the network can do are applied.
	Capabilities []string `json:"capabilities,omitempty"`
}

// Workspace is an open composition workspace: the session-owned persistence
// (control directory + manifest + step stamps) plus the netcompose domain
// state the steps accumulate.
type Workspace struct {
	// ledger is the persisted run record — which machine runs which binary,
	// under which command, as which pid. It is the source of truth for PIDs;
	// NodeState.PID is the in-memory view, synced from here on Open.
	ledger *process.Ledger
	comp   session.Composition
	state  State
	env    func(string) string
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
	comp, err := session.OpenComposition(dir, now)
	if err != nil {
		return nil, err
	}
	ws := &Workspace{comp: comp, env: os.Getenv, state: State{Steps: map[string]Step{}}}
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
func (w *Workspace) resolveTarget() (*machine.Access, error) {
	return w.opener().Open(w.state.Target)
}

// opener binds the workspace's recorded server set and docker choice to the
// netmap module's single wiring point.
func (w *Workspace) opener() netmapmod.Opener {
	return netmapmod.Opener{ServerSet: w.state.ServerSet, Docker: w.state.Docker, Env: w.env}
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
		Network:      string(w.state.Target.Kind),
		Capabilities: w.state.Capabilities,
		Nodes:        make([]node.Node, 0, len(w.state.Nodes)),
	}
	if ns.Network == "" {
		ns.Network = string(machine.KindLocal)
	}
	for _, n := range w.state.Nodes {
		// A node's own recorded host wins: a fleet placement puts each node on
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
			Ports: node.Endpoints{
				P2P: n.P2P, HTTP: n.HTTP, WS: n.WS, Auth: n.Auth, Metrics: n.Metrics,
			},
			PID: n.PID,
		})
	}
	return ns
}

// NodeLabel is the node's identity, falling back to the conventional label for
// workspaces written before the field existed.
func (n NodeState) NodeLabel() netmap.NodeLabel {
	if n.Label != "" {
		return netmap.NodeLabel(n.Label)
	}
	return netmap.LabelFor(n.Index)
}
