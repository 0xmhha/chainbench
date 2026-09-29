package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// docPathBudget is how many repository paths the live documentation names that
// are not there. It only comes down.
//
// It was 36 when the check was written and the check came first on purpose: a
// count is what tells you whether fixing them is an afternoon or a week. It
// reached 0, and on 2026-09-29 the matcher widened from Go files to any path in
// our tree, which put it back to 107 for an afternoon. Lower it when references
// are fixed; never raise it.
const docPathBudget = 0

// docNoteWindow is how much of a document is read looking for its path note. A
// note past this is not near the title, so it does not exempt the body.
const docNoteWindow = 4000

// pathNote and pathFix are the two spellings this repository uses for that
// declaration.
const (
	pathNote = "경로 안내"
	pathFix  = "경로 정정"
)

// docRoots are the documents a reader is expected to act on.
//
// docs/research/ is left out, and the reason is the same one that governs a
// comment: what a record SAID at the time is not a false claim now. A frozen
// analysis naming a file that has since been renamed is the record working. A
// live guide doing it sends a reader to a path that is not there.
//
// docs/dev/archive/ used to be skipped for the same reason. It was retired on
// 2026-09-24 and its contents live in git history, so there is nothing left
// to skip.
var docRoots = []string{"docs/dev", "docs/guide"}

// goPathInProse matches a repository path written in a document: a Go file, a
// package directory, a test case, a preset, a script.
//
// It started at Go files under internal/ and cmd/ only. Widening it found 107
// more on 2026-09-29 — case trees that had been consolidated into tests/tc/,
// preset directories that moved, a guide pointing at tests/tc/samples/ the day
// after it was emptied. A path to a directory sends a reader nowhere just as a
// path to a file does.
// The path has to start the token. Without that, `wemix/scripts/genesis-template.json`
// in a go-wemix checkout reads as our `scripts/...`, and `netmap/internal/serverset`
// reads as our `internal/serverset`. Another repository's tree is not ours to hold.
// A match ending in `_` is a stem in a diagram (`tests/001_<id>/`), not a path.
var goPathInProse = regexp.MustCompile(`(^|[\s` + "`" + `(|\[])((?:internal|cmd|tests|presets|scripts|env|examples|web)/[A-Za-z0-9_./-]*[A-Za-z0-9])`)

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
			b, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			// A document may declare, once at the top, that its paths are the
			// tree as it stood when the measurement was taken. That is how a
			// frozen graph or a 2,900-line tracker stays honest without
			// rewriting the record line by line, and this repository already
			// writes it that way. The note has to be near the title, so it is
			// read before the body, not buried after it.
			if head := string(b); len(head) > docNoteWindow {
				head = head[:docNoteWindow]
			} else if strings.Contains(head, pathNote) || strings.Contains(head, pathFix) {
				return nil
			}
			if h := string(b); len(h) > docNoteWindow &&
				(strings.Contains(h[:docNoteWindow], pathNote) || strings.Contains(h[:docNoteWindow], pathFix)) {
				return nil
			}
			for _, line := range strings.Split(string(b), "\n") {
				var hits []string
				for _, m := range goPathInProse.FindAllStringIndex(line, -1) {
					whole := line[m[0]:m[1]]
					path := strings.TrimLeft(whole, " \t`([|")
					// What follows decides whether this is a path or the stem of
					// one: `tests/001_<id>/` in a tree diagram is a shape.
					if m[1] < len(line) {
						switch line[m[1]] {
						case '_', '<':
							continue
						}
					}
					m := []string{whole, "", path}
					// An elided path (tests/tc/.../case.json) names a shape, not
					// a file, so there is nothing to resolve.
					if strings.Contains(m[2], "...") {
						continue
					}
					hits = append(hits, m[2])
				}
				if len(hits) == 0 {
					continue
				}
				seen += len(hits)
				// A line that names a path that is gone AND one that is there
				// is a record of the move, and naming the old one is the
				// point. So is a blockquote: this repository corrects a
				// document by quoting the correction above the text it
				// corrects, and the correction has to be able to say what the
				// text used to say.
				live, dead := 0, []string{}
				for _, h := range hits {
					if _, err := os.Stat(filepath.Join(root, h)); err == nil {
						live++
					} else {
						dead = append(dead, h)
					}
				}
				if live > 0 || strings.HasPrefix(strings.TrimSpace(line), ">") {
					continue
				}
				for _, d := range dead {
					missing = append(missing, miss{d, rel})
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
