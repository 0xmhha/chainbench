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
// SSH host); see target.go. Step functions use the Target's FileSink/Driver and
// never branch on local vs remote.
//
// Persistence belongs to core/session (Composition — the long-lived
// environment mode); this package owns only the domain state and the step
// functions. The CLI and MCP surfaces are thin wrappers over the app layer,
// which calls the step functions here, so both drive the exact same behavior.
package netcompose

import (
	"os"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// Step is a completed composition step (persistence model owned by session).
type Step = session.Step

// NodeState is one composed node's resolved assignment: its role, target-side
// paths, allocated ports, the assembled launch argv (once `launchopts` ran),
// and the live PID (once `start` ran; 0 = stopped).
type NodeState struct {
	Index      int      `json:"index"`
	Role       string   `json:"role"`
	DataDir    string   `json:"dataDir"`
	ConfigPath string   `json:"configPath"`
	LogPath    string   `json:"logPath"`
	P2P        int      `json:"p2p"`
	HTTP       int      `json:"http"`
	WS         int      `json:"ws"`
	Auth       int      `json:"auth"`
	Metrics    int      `json:"metrics,omitempty"`
	Args       []string `json:"args,omitempty"`
	PID        int      `json:"pid,omitempty"`
}

// State is the persisted composition state accumulated across step commands. It
// grows as later steps land (placements, genesis path, node table); each field
// is optional so a partially-composed workspace round-trips. It holds no
// secrets — remote credentials live only in the environment.
type State struct {
	Chain       string          `json:"chain"`
	Binary      string          `json:"binary,omitempty"`
	KeysDir     string          `json:"keysDir,omitempty"`
	Validators  int             `json:"validators,omitempty"`
	Target      TargetSpec      `json:"target"`
	GenesisPath string          `json:"genesisPath,omitempty"`
	Nodes       []NodeState     `json:"nodes,omitempty"`
	Steps       map[string]Step `json:"steps"`
}

// Workspace is an open composition workspace: the session-owned persistence
// (control directory + manifest + step stamps) plus the netcompose domain
// state the steps accumulate.
type Workspace struct {
	comp  session.Composition
	state State
	env   func(string) string
}

// Open opens (creating if absent) the workspace at dir. now is injected for
// deterministic timestamps; nil uses time.Now.
func Open(dir string, now func() time.Time) (*Workspace, error) {
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
	return ws, nil
}

// Dir is the workspace's local control directory.
func (w *Workspace) Dir() string { return w.comp.Dir() }

// SetEnv overrides the environment reader used when resolving a remote target
// (credentials). Nil is ignored; the default is os.Getenv.
func (w *Workspace) SetEnv(fn func(string) string) {
	if fn != nil {
		w.env = fn
	}
}

// State returns a copy of the current composition state.
func (w *Workspace) State() State { return w.state }

// markStep records that step ran with detail, stamping the completion time.
func (w *Workspace) markStep(step, detail string) {
	w.state.Steps[step] = w.comp.StepMark(detail)
}

// Save writes the composition state to the manifest.
func (w *Workspace) Save() error { return w.comp.Save(w.state) }
