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
// state across independent commands (`chain keys`, `chain start`, ...). Owning
// both here removes the third parallel state-store implementation the
// structure review measured (state / session / Workspace).

// chainRecordFile is the chain record at the directory root: what this chain
// was asked to be, what was composed for it, and how far the composition got.
//
// It is not named for the directory it sits in. The directory IS a workspace —
// runs/, the node data directories, genesis and the configs are all in it —
// but this one file is the record of a chain, and calling it workspace.json
// made every reader ask which of the two a given mention meant.
const chainRecordFile = "chain-record.json"

// legacyRecordFile is the name the chain record had before it was named for
// what it holds.
//
// It is still looked for. A build reads only the format version it writes and
// refuses any other by name (see chainsetup.StateFormatVersion); a rename with
// no lookup would turn that refusal into silence, because a directory holding
// only the old name reads as never composed and the next command would compose
// a second chain beside the first.
const legacyRecordFile = "workspace.json"

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

// Load reads the persisted state into out and reports whether a record was
// there to read. A composition that has never been saved loads nothing and
// reports false — the zero state is the starting point.
//
// The caller needs the two apart. A zero field in a record that exists means
// the record was written without it; the same zero in a record that does not
// exist means nothing at all, and a caller that cannot tell which reads the
// first case as the second.
func (c Composition) Load(out any) (bool, error) {
	b, err := os.ReadFile(filepath.Join(c.dir, chainRecordFile))
	if os.IsNotExist(err) {
		if legacyErr := refuseLegacyRecord(c.dir); legacyErr != nil {
			return false, legacyErr
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("session: read %s: %w", chainRecordFile, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return false, fmt.Errorf("session: parse %s: %w", chainRecordFile, err)
	}
	return true, nil
}

// Save writes the state to the composition's manifest.
func (c Composition) Save(state any) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal composition state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(c.dir, chainRecordFile), b, 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", chainRecordFile, err)
	}
	return nil
}

// StepMark stamps a completed step with the composition's clock.
func (c Composition) StepMark(detail string) Step {
	return Step{Done: true, Detail: detail, At: c.now().UTC().Format(time.RFC3339)}
}

// ChainRecordPath is where a composition's chain record lives under dir.
// Exported because session owns the artifact layout: a caller asking "is this
// directory a composition?" must not hard-code the file name.
func ChainRecordPath(dir string) string {
	return filepath.Join(dir, chainRecordFile)
}

// NoRecordError says dir holds no chain record. It says so differently when
// the directory still holds the name the record used to have, so an operator
// reading the refusal is told which file to look at rather than being told
// their composed directory is empty.
func NoRecordError(dir string) error {
	if err := refuseLegacyRecord(dir); err != nil {
		return err
	}
	return fmt.Errorf("session: %s holds no chain record (no %s)", dir, chainRecordFile)
}

// refuseLegacyRecord returns an error naming both files when dir holds only
// the old record, and nil otherwise.
func refuseLegacyRecord(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, legacyRecordFile)); err != nil {
		return nil
	}
	return fmt.Errorf("session: %s holds %s, which this build does not read — compose the chain again to get %s", dir, legacyRecordFile, chainRecordFile)
}
