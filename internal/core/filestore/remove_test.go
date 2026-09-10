package filestore_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// Removal is the one thing this package does that cannot be undone, and the
// same guard runs in front of a local os.RemoveAll and a remote `rm -rf` on
// somebody's server. These tests are the guard's contract.

// TestCheckRemovable_RefusesTheCatastrophicShapes covers the paths that are
// wrong regardless of who asked. Each of them is what a path built from an
// empty or unresolved variable collapses into.
func TestCheckRemovable_RefusesTheCatastrophicShapes(t *testing.T) {
	for name, p := range map[string]string{
		"empty":             "",
		"blank":             "   ",
		"filesystem root":   "/",
		"root with slash":   "//",
		"root via dots":     "/data/..",
		"one segment":       "/data",
		"one segment slash": "/data/",
		"relative":          "data/chainbench/runtime",
		"bare relative":     "runtime",
		"dot":               ".",
		"env var":           "/data/$HOME/runtime",
		"tilde":             "~/chainbench",
		"glob":              "/data/chainbench/*",
		"question glob":     "/data/chainbench/node?",
	} {
		t.Run(name, func(t *testing.T) {
			if err := filestore.CheckRemovable(p); err == nil {
				t.Fatalf("CheckRemovable(%q) allowed a delete that must never happen", p)
			}
		})
	}
}

// TestCheckRemovable_AllowsARealCompositionPath keeps the guard from being so
// strict it refuses the thing it exists to permit.
func TestCheckRemovable_AllowsARealCompositionPath(t *testing.T) {
	for _, p := range []string{
		"/data/chainbench",
		"/data/chainbench/runtime/c55a956b4463/node1",
		"/data/chainbench/runtime/c55a956b4463/genesis.json",
		"/tmp/cbws.abc/node1",
		"/data/my chain/node 1",
	} {
		if err := filestore.CheckRemovable(p); err != nil {
			t.Errorf("CheckRemovable(%q) refused a legitimate path: %v", p, err)
		}
	}
}

// TestCheckWithin_ConfinesToTheTargetRoot is the precise half: a path that
// passes the coarse guard can still be somebody else's directory.
func TestCheckWithin_ConfinesToTheTargetRoot(t *testing.T) {
	const root = "/data/chainbench"
	ok := []string{
		"/data/chainbench/runtime/abc/node1",
		"/data/chainbench/keys",
		"/data/chainbench/runtime/abc/../abc/genesis.json", // cleans to inside
	}
	for _, p := range ok {
		if err := filestore.CheckWithin(root, p); err != nil {
			t.Errorf("CheckWithin(%q, %q) refused a path inside the root: %v", root, p, err)
		}
	}
	bad := map[string]string{
		"escapes upward":  "/data/chainbench/../etc",
		"sibling root":    "/data/chainbench-other/node1",
		"unrelated":       "/etc/passwd",
		"the root itself": "/data/chainbench",
		"root with slash": "/data/chainbench/",
		"no root given":   "/data/chainbench/node1",
	}
	for name, p := range bad {
		useRoot := root
		if name == "no root given" {
			useRoot = ""
		}
		if err := filestore.CheckWithin(useRoot, p); err == nil {
			t.Errorf("%s: CheckWithin(%q, %q) allowed a delete outside the composition", name, useRoot, p)
		}
	}
}

// TestLocalRemove_DeletesATreeAndIsIdempotent pins the behavior the interface
// promises: a directory goes whole, and a second call is not an error, because
// clearing a composition twice must not fail.
func TestLocalRemove_DeletesATreeAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "runtime", "abc", "node1")
	if err := os.MkdirAll(filepath.Join(tree, "geth", "chaindata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "geth", "chaindata", "CURRENT"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var s filestore.Store = filestore.Local{}
	if err := s.Remove(context.Background(), tree); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Fatalf("the tree survived: %v", err)
	}
	if err := s.Remove(context.Background(), tree); err != nil {
		t.Errorf("removing an absent path should succeed, got %v", err)
	}
}

// TestLocalRemove_AppliesTheGuard proves the backstop runs inside the store and
// not only at the caller: an implementation that trusted its caller would be
// the one an unchecked path reached.
func TestLocalRemove_AppliesTheGuard(t *testing.T) {
	var s filestore.Store = filestore.Local{}
	err := s.Remove(context.Background(), "/")
	if err == nil {
		t.Fatal("Local.Remove accepted the filesystem root")
	}
	if !strings.Contains(err.Error(), "filesystem root") {
		t.Errorf("the refusal should name what it refused: %v", err)
	}
}
