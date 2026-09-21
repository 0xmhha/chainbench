package chainsetup

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// Running one piece of work against a workspace: open it, hold it, save it.
//
// It sits below the verbs because it is not only theirs — the reuse
// reconciliation takes a workspace the same way, and a helper living beside the
// verbs would be one the steps import upward.

// verbs, which do not, are in verbs_lifecycle.go.

func withWorkspace(d Deps, dataDir string, fn func(*Workspace) (string, error)) (string, error) {
	return inWorkspace(d, dataDir, fn)
}

// inWorkspace is withWorkspace for a step that reports more than a line.
//
// A step that decides something — which of three key sources it used, which of
// two targets it shipped to — has to be able to say so, and a string is what it
// says to a person rather than to the caller. The lock, the save and the way
// the two errors are joined are the same for both, so they are written once
// here and withWorkspace is the string case of it.
func inWorkspace[T any](d Deps, dataDir string, fn func(*Workspace) (T, error)) (T, error) {
	var zero T
	ws, err := Open(dataDir, d.Clock)
	if err != nil {
		return zero, err
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)

	// One run at a time per workspace. A second run would compose over the
	// first's half-built network and blame the collision on the chain; the
	// refusal names who holds it instead. A lock left by a run that died is
	// taken over — that run is gone, and its wreckage is what the operator is
	// here to clear — but never in silence.
	held, prev, state, lerr := ws.Acquire(d.command())
	if lerr != nil {
		return zero, lerr
	}
	defer func() { _ = held.Release() }()
	if state == session.LockStale {
		d.logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up; `chain status` shows what is there", prev.Describe())
	}

	out, stepErr := fn(ws)
	saveErr := ws.Save()
	switch {
	case stepErr != nil && saveErr != nil:
		return out, fmt.Errorf("%w (and the workspace could not be saved: %v — processes this step started may not be recorded)", stepErr, saveErr)
	case stepErr != nil:
		// The value comes back with the error. A step that fails partway is
		// still the authority on how far it got, and a caller that threw that
		// away had to guess — which is what the genesis stage was doing when it
		// reported a fork failure as if the genesis had never been built.
		return out, stepErr
	case saveErr != nil:
		return zero, saveErr
	}
	return out, nil
}
