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
// The CLI and MCP surfaces are thin wrappers over the step functions here, so
// both drive the exact same behavior.
package netcompose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// workspaceFile is the composition-state manifest at the data-dir root.
const workspaceFile = "workspace.json"

// dirPerm is the permission for the data directory.
const dirPerm os.FileMode = 0o755

// Step records that one composition step ran, with a human-readable detail and
// the (injected) timestamp it completed.
type Step struct {
	Done   bool   `json:"done"`
	Detail string `json:"detail,omitempty"`
	At     string `json:"at,omitempty"`
}

// State is the persisted composition state accumulated across step commands. It
// grows as later steps land (placements, genesis path, node table); each field
// is optional so a partially-composed workspace round-trips. It holds no
// secrets — remote credentials live only in the environment.
type State struct {
	Chain      string          `json:"chain"`
	Binary     string          `json:"binary,omitempty"`
	KeysDir    string          `json:"keysDir,omitempty"`
	Validators int             `json:"validators,omitempty"`
	Target     TargetSpec      `json:"target"`
	Steps      map[string]Step `json:"steps"`
}

// Workspace is an open composition workspace rooted at a local control directory.
type Workspace struct {
	dir   string
	state State
	now   func() time.Time
}

// Open opens (creating if absent) the workspace at dir. now is injected for
// deterministic timestamps; nil uses time.Now.
func Open(dir string, now func() time.Time) (*Workspace, error) {
	if dir == "" {
		return nil, fmt.Errorf("netcompose: data dir is required")
	}
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("netcompose: mkdir %s: %w", dir, err)
	}
	ws := &Workspace{dir: dir, now: now, state: State{Steps: map[string]Step{}}}
	if b, err := os.ReadFile(filepath.Join(dir, workspaceFile)); err == nil {
		if err := json.Unmarshal(b, &ws.state); err != nil {
			return nil, fmt.Errorf("netcompose: parse %s: %w", workspaceFile, err)
		}
		if ws.state.Steps == nil {
			ws.state.Steps = map[string]Step{}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("netcompose: read %s: %w", workspaceFile, err)
	}
	return ws, nil
}

// Dir is the workspace's local control directory.
func (w *Workspace) Dir() string { return w.dir }

// State returns a copy of the current composition state.
func (w *Workspace) State() State { return w.state }

// markStep records that step ran with detail, stamping the completion time.
func (w *Workspace) markStep(step, detail string) {
	w.state.Steps[step] = Step{Done: true, Detail: detail, At: w.now().UTC().Format(time.RFC3339)}
}

// Save writes the composition state to workspace.json.
func (w *Workspace) Save() error {
	b, err := json.MarshalIndent(w.state, "", "  ")
	if err != nil {
		return fmt.Errorf("netcompose: marshal state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(w.dir, workspaceFile), b, 0o644); err != nil {
		return fmt.Errorf("netcompose: write %s: %w", workspaceFile, err)
	}
	return nil
}
