package home_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/home"
)

// TestRoot_IsNotRelativeToWhereYouStand is the whole point.
//
// The defaults this package replaced were "keys/default" and "chainbench-out",
// both relative: a ring created in one directory was invisible from another,
// and sessions landed wherever the operator happened to be. A promised
// location that moves with the working directory is not a promised location.
func TestRoot_IsNotRelativeToWhereYouStand(t *testing.T) {
	root, err := home.Root()
	if err != nil {
		t.Skipf("no home directory on this machine: %v", err)
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("Root() = %q, which is relative — the default would follow the working directory", root)
	}
	h, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	if want := filepath.Join(h, home.Dir); root != want {
		t.Fatalf("Root() = %q, want %q", root, want)
	}
}

// TestUnder_KeepsTheKindsApart: the three kinds of asset share one root and
// keep their own place under it, so an operator can find any of them without
// remembering which command wrote it.
func TestUnder_KeepsTheKindsApart(t *testing.T) {
	keys, err := home.KeySets()
	if err != nil {
		t.Skip("no home directory")
	}
	sessions, err := home.Sessions()
	if err != nil {
		t.Skip("no home directory")
	}
	root, _ := home.Root()
	for name, got := range map[string]string{"keys": keys, "sessions": sessions} {
		if !strings.HasPrefix(got, root+string(filepath.Separator)) {
			t.Errorf("%s = %q, which is not under the root %q", name, got, root)
		}
	}
	if keys == sessions {
		t.Error("key sets and sessions resolve to the same directory")
	}
}
