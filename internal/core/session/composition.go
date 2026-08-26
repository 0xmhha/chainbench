package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Long-lived environment mode. A per-run artifact session
// (session.New) lives for one engine run; a Composition is the other lifetime
// this package owns: a persistent, step-composed environment that accumulates
// state across independent commands (`net keys`, `net start`, ...). Owning
// both here removes the third parallel state-store implementation the
// structure review measured (state / session / Workspace).

// compositionFile is the composition-state manifest at the directory root.
const compositionFile = "workspace.json"

// compositionDirPerm is the permission for a created composition directory.
const compositionDirPerm os.FileMode = 0o755

// Step records that one composition step ran, with a human-readable detail
// and the (injected) timestamp it completed.
type Step struct {
	Done   bool   `json:"done"`
	Detail string `json:"detail,omitempty"`
	At     string `json:"at,omitempty"`
}

// Composition is the persistence boundary of one long-lived environment: it owns
// the control directory, the state file, and step timestamps. The state
// payload's shape belongs to the caller (netcompose keeps its domain state);
// this type owns where and how it persists.
type Composition struct {
	dir string
	now func() time.Time
}

// OpenComposition opens (creating if absent) the composition directory. now is
// injected for deterministic timestamps; nil uses time.Now.
func OpenComposition(dir string, now func() time.Time) (Composition, error) {
	if dir == "" {
		return Composition{}, fmt.Errorf("session: composition dir is required")
	}
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(dir, compositionDirPerm); err != nil {
		return Composition{}, fmt.Errorf("session: mkdir %s: %w", dir, err)
	}
	return Composition{dir: dir, now: now}, nil
}

// Dir is the composition's control directory.
func (c Composition) Dir() string { return c.dir }

// Load reads the persisted state into out. A composition that has never been
// saved loads nothing and returns nil — the zero state is the starting point.
func (c Composition) Load(out any) error {
	b, err := os.ReadFile(filepath.Join(c.dir, compositionFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("session: read %s: %w", compositionFile, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("session: parse %s: %w", compositionFile, err)
	}
	return nil
}

// Save writes the state to the composition's manifest.
func (c Composition) Save(state any) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal composition state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(c.dir, compositionFile), b, 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", compositionFile, err)
	}
	return nil
}

// StepMark stamps a completed step with the composition's clock.
func (c Composition) StepMark(detail string) Step {
	return Step{Done: true, Detail: detail, At: c.now().UTC().Format(time.RFC3339)}
}

// CompositionFilePath is where a composition's state manifest lives under dir.
// Exported because session owns the artifact layout: a caller asking "is this
// directory a composition?" must not hard-code the file name.
func CompositionFilePath(dir string) string {
	return filepath.Join(dir, compositionFile)
}
