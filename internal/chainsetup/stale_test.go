package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// listingCommander answers every command with the same output.
type listingCommander struct{ out string }

func (c listingCommander) Run(context.Context, string) (string, error) { return c.out, nil }

// TestCompositionsOn_OnlyCompositionIDsCount: what sits under the purpose
// directories is a composition only when its name is a composition id. A flat
// layout's node1, or anything else somebody put there, is never a candidate
// for removal.
func TestCompositionsOn_OnlyCompositionIDsCount(t *testing.T) {
	listing := "/data/node/85b701329116\n/data/runtime/85b701329116\n/data/logs/85b701329116\n" +
		"/data/node/node1\n/data/logs/keep-me\n/data/node/0123456789ab\n"
	found, err := compositionsOn(context.Background(), listingCommander{listing}, "/data", []string{"node", "runtime", "logs"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedKeys(found); !slices.Equal(got, []string{"0123456789ab", "85b701329116"}) {
		t.Errorf("compositions = %v", got)
	}
	if len(found["85b701329116"]) != 3 {
		t.Errorf("85b701329116 dirs = %v, want its node, runtime and logs directories", found["85b701329116"])
	}
}

// TestKnownCompositions_AWorkspaceUnderKeepUnderIsKept: a workspace kept outside
// ~/.chainbench is found by searching the directories the operator names, and
// its composition id is the one compose gave it.
func TestKnownCompositions_AWorkspaceUnderKeepUnderIsKept(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	base := t.TempDir()
	ws := filepath.Join(base, "manual", "basic-consensus")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, workspaceRecordFile), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	known, err := knownCompositions([]string{base})
	if err != nil {
		t.Fatal(err)
	}
	if !known[compositionID(ws)] || len(known) != 1 {
		t.Errorf("known = %v, want only %s", known, compositionID(ws))
	}
}
