package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// TestWithWorkspace_RecordsWhatAStepDidEvenWhenItFailed.
//
// A step that launches nodes and then fails has already changed the world:
// those processes hold ports whether the step reports success or not. Saving
// only on success threw their pids away with the error, and a pid nobody
// recorded is an orphan — `net stop` finds nothing and the next run fails with
// "address already in use".
//
// Measured before the fix: interrupting `net up` mid-bring-up left four wemix
// nodes running and the workspace holding four null pids.
func TestWithWorkspace_RecordsWhatAStepDidEvenWhenItFailed(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet"}); err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// The step allocates (which writes the node table) and then fails, standing
	// in for a start that launched nodes and was then interrupted.
	boom := errors.New("the step failed after starting something")
	_, err = withWorkspace(Deps{}, dir, func(ws *chainsetup.Workspace) (string, error) {
		if _, aerr := ws.Allocate(chainsetup.AllocateOpts{Validators: 2}); aerr != nil {
			return "", aerr
		}
		return "", boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the step's own error", err)
	}

	reopened, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if n := len(reopened.State().Nodes); n != 2 {
		t.Fatalf("the reopened workspace has %d node(s); what the step did was discarded with its error", n)
	}

	if _, err := os.Stat(filepath.Join(dir, "workspace.json")); err != nil {
		t.Fatalf("workspace file missing: %v", err)
	}
}
