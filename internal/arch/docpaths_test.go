package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// docPathBudget is how many Go files the live documentation names that are not
// there. It only comes down.
//
// It is not zero because it was 36 when the check was written, and the check
// came first on purpose: a count is what tells you whether fixing them is an
// afternoon or a week. Lower it when references are fixed; never raise it.
const docPathBudget = 36

// docRoots are the documents a reader is expected to act on.
//
// docs/dev/archive/ and docs/research/ are left out, and the reason is the same
// one that governs a comment: what a record SAID at the time is not a false
// claim now. An archived design naming a file that has since been renamed is
// the archive working. A live guide doing it sends a reader to a path that is
// not there.
var docRoots = []string{"docs/dev", "docs/guide"}

var docSkip = []string{filepath.Join("docs", "dev", "archive")}

// goPathInProse matches a repository path to a Go file written in a document.
var goPathInProse = regexp.MustCompile(`\b((?:internal|cmd)/[a-z0-9_/-]+\.go)\b`)

// TestDocsDoNotNameFilesThatAreGone holds the live documents to the tree.
//
// It exists because they drifted and nothing said so. Splitting the verb layer
// out of chainsetup moved ten files into internal/chainsetup/verb/ and left 50
// references behind across the guides — one of them the chain-setup README's,
// which is where a reader starts. Nineteen more named files that earlier
// refactorings had deleted, some of them a year old.
//
// A comment that names a dead file is already caught (KindDeadFileCite). A
// document was not, and a document is what somebody reads first.
func TestDocsDoNotNameFilesThatAreGone(t *testing.T) {
	root := "../.."
	type miss struct{ path, doc string }
	var missing []miss
	seen := 0

	for _, r := range docRoots {
		err := filepath.Walk(filepath.Join(root, r), func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			for _, s := range docSkip {
				if strings.HasPrefix(rel, s) {
					return nil
				}
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			for _, m := range goPathInProse.FindAllStringSubmatch(string(b), -1) {
				seen++
				if _, err := os.Stat(filepath.Join(root, m[1])); err != nil {
					missing = append(missing, miss{m[1], rel})
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if seen == 0 {
		t.Fatal("no Go paths were found in the documents, so this proves nothing — the walk is broken")
	}

	sort.Slice(missing, func(i, j int) bool {
		if missing[i].path != missing[j].path {
			return missing[i].path < missing[j].path
		}
		return missing[i].doc < missing[j].doc
	})
	if len(missing) > docPathBudget {
		lines := make([]string, 0, len(missing))
		for _, m := range missing {
			lines = append(lines, "  "+m.doc+" names "+m.path)
		}
		t.Errorf("%d path(s) named in the live documents are not in the tree, over the budget of %d.\n"+
			"  Point them at what does that job now, or say what replaced it. %d of %d paths resolve.\n%s",
			len(missing), docPathBudget, seen-len(missing), seen, strings.Join(lines, "\n"))
	}
	if len(missing) < docPathBudget {
		t.Errorf("%d path(s) are missing and the budget is %d — lower it to %d, so the ceiling tracks the tree",
			len(missing), docPathBudget, len(missing))
	}
}
