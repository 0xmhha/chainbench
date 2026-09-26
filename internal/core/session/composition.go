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

// StepState is how a composition step ended.
//
// It is a string in the record because a record is read by people, and a
// number there would need this file open beside it to mean anything. It is not
// StepResult, which is the outcome of one TEST step (a tx hash, a receipt); the
// two words sit in the same package and mean different things.
type StepState string

const (
	// StepRunning is written when a step begins. A record whose last step is
	// still running is a run that died inside it — which is the one thing the
	// record could not say before, because a step was only ever written after
	// it succeeded.
	StepRunning StepState = "running"
	// StepDone is a step that finished its work.
	StepDone StepState = "done"
	// StepFailed is a step that was reached and did not finish. Err says why.
	StepFailed StepState = "failed"
	// StepReused is a step that had nothing to do because what it would have
	// produced was already there. It is not StepDone: a reader asking "did this
	// run build the genesis?" has to be able to tell "yes" from "it was already
	// built", and an absent step from a step that was skipped on purpose.
	StepReused StepState = "reused"
)

// Step records one composition step: that it was reached, how it ended, and
// when. Detail is the line an operator reads.
//
// Done is kept because records written before Result existed have it, and a
// reader of those records still has to work. New writes set both.
type Step struct {
	Done   bool      `json:"done"`
	Result StepState `json:"result,omitempty"`
	Detail string    `json:"detail,omitempty"`
	Err    string    `json:"err,omitempty"`
	// Failed is the failure state the failing stage's classifier named for Err
	// (for example ChainLaunchNodesFailPortBusy). Empty when the step did not
	// fail, or failed before its stage could classify it.
	Failed string `json:"failed,omitempty"`
	// StartedAt and At are RFC3339 UTC. A step that is still running has the
	// first and not the second.
	StartedAt string `json:"startedAt,omitempty"`
	At        string `json:"at,omitempty"`
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
//
// Atomically, through the same helper every other record in this package uses.
// A direct write truncates the file and then fills it, so a process that dies
// in between leaves a half-written record — and this record is what says a
// workspace is composed at all. A reader finding it truncated reads a
// composition that is not there, which is worse than finding the previous one:
// the previous one was at least true a moment ago.
func (c Composition) Save(state any) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal composition state: %w", err)
	}
	if err := WriteFileAtomic(filepath.Join(c.dir, chainRecordFile), b, 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", chainRecordFile, err)
	}
	return nil
}

// StepMark stamps a completed step with the composition's clock.
func (c Composition) StepMark(detail string) Step {
	now := c.now().UTC().Format(time.RFC3339)
	return Step{Done: true, Result: StepDone, Detail: detail, StartedAt: now, At: now}
}

// StepBegin marks a step as reached but not finished. The caller overwrites it
// with StepEnd; a record left holding this is a run that died inside the step.
func (c Composition) StepBegin() Step {
	return Step{Result: StepRunning, StartedAt: c.now().UTC().Format(time.RFC3339)}
}

// StepEnd closes a step begun with StepBegin, keeping its start time.
func (c Composition) StepEnd(begun Step, detail string, err error) Step {
	out := begun
	out.Detail = detail
	out.At = c.now().UTC().Format(time.RFC3339)
	if err != nil {
		out.Result = StepFailed
		out.Err = err.Error()
		return out
	}
	out.Done = true
	out.Result = StepDone
	return out
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
