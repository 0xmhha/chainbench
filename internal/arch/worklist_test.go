package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// worklistDoc is the single source of truth for what is left to do.
const worklistDoc = "../../docs/dev/chainbench-worklist.md"

// openWork is how many items §0 lists. It is pinned so the list cannot drift
// silently in either direction: an item finished without being struck, or one
// added without being counted, both show up here.
//
// Why the list exists at all, and why counting the document instead does not
// work: the tracker has used two marker vocabularies — the ☐/◐/☑ its own preamble
// declares, and the GitHub-style "- [ ]" its later sections use. On 2026-09-12 a
// status pass reported "nothing open" three times running and was wrong every
// time, because it counted "- [ ]" and ◐ and missed eleven ☐. A second attempt to
// count mechanically failed differently: the markers appear in prose, mid-cell and
// as a cell's first token, so a rule tight enough to skip the prose also skipped a
// real item whose table row had wrapped.
//
// So the document stopped being counted and started carrying one canonical list.
//
// The honest limit: this cannot catch an item added only to the narrative, deep in
// 2,900 lines. What it catches is the canonical list going stale — which is the
// failure that actually happened.
const openWork = 10

var (
	openItem    = regexp.MustCompile(`(?m)^- \[ \] `)
	rowToken    = regexp.MustCompile("`([^`]+)` 행")
	sectionRef  = regexp.MustCompile(`§(\d+[a-z]?)`)
	sectionDecl = regexp.MustCompile(`(?m)^#{2,3} (\d+[a-z]?)\.`)
)

func worklist(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(worklistDoc))
	if err != nil {
		t.Fatalf("read %s: %v", worklistDoc, err)
	}
	return string(b)
}

// openSection returns §0 — from its heading to the next one.
func openSection(t *testing.T, doc string) string {
	t.Helper()
	const head = "\n## 0. 열린 작업"
	i := strings.Index(doc, head)
	if i < 0 {
		t.Fatalf("%s has no \"## 0. 열린 작업\" section — the canonical list is where open work lives", worklistDoc)
	}
	rest := doc[i+len(head):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// TestWorklistOpenWorkIsListed pins the canonical list's size. A number that has
// to be edited deliberately is what makes finishing something visible.
func TestWorklistOpenWorkIsListed(t *testing.T) {
	sec := openSection(t, worklist(t))
	got := len(openItem.FindAllString(sec, -1))
	switch {
	case got > openWork:
		t.Errorf("§0 lists %d open items, up from %d — record the new work here AND lower nothing, or raise the constant on purpose", got, openWork)
	case got < openWork:
		t.Errorf("§0 lists %d open items, down from %d — lower the constant so it keeps tracking reality", got, openWork)
	}
	t.Logf("§0 lists %d open items", got)
}

// TestWorklistOpenWorkPointsSomewhere keeps each entry attached to its evidence.
// An item whose §-reference names a section that does not exist is a ghost: it
// reads as tracked and leads nowhere, which is how the tracker rotted in the first
// place.
func TestWorklistOpenWorkPointsSomewhere(t *testing.T) {
	doc := worklist(t)
	declared := map[string]bool{}
	for _, m := range sectionDecl.FindAllStringSubmatch(doc, -1) {
		declared[m[1]] = true
	}
	if len(declared) == 0 {
		t.Fatal("no numbered sections found, so the parse is wrong rather than the document flat")
	}
	for _, line := range strings.Split(openSection(t, doc), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "- [ ] ") {
			continue
		}
		refs := sectionRef.FindAllStringSubmatch(line, -1)
		if len(refs) == 0 {
			// An item may point at a file instead of a section; require one or
			// the other, because an item with neither cannot be followed.
			if !strings.Contains(line, "](") && !strings.Contains(line, "`internal/") {
				t.Errorf("this open item names neither a section nor a file, so nobody can find its evidence:\n  %s", strings.TrimSpace(line))
			}
			continue
		}
		for _, r := range refs {
			if !declared[r[1]] {
				t.Errorf("open item points at §%s, which this document does not declare:\n  %s", r[1], strings.TrimSpace(line))
				continue
			}
			// A section that exists is not yet the right section. When the item
			// also names the row it means (`10+` 행), that row has to be inside
			// it — otherwise the reference reads as precise and sends the reader
			// to the wrong table, which is worse than a vague one. Caught S2
			// pointing at §1l when its row is in §1g.
			if row := rowToken.FindStringSubmatch(line); row != nil {
				if !sectionContains(doc, r[1], row[1]) {
					t.Errorf("open item points at §%s but its row %q is not there:\n  %s", r[1], row[1], strings.TrimSpace(line))
				}
			}
		}
	}
}

// sectionContains reports whether token appears inside section §name — the span
// from that heading to the next numbered one.
func sectionContains(doc, name, token string) bool {
	lines := strings.Split(doc, "\n")
	start := -1
	for i, l := range lines {
		m := sectionDecl.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		if m[1] == name {
			start = i
			continue
		}
		if start >= 0 {
			return strings.Contains(strings.Join(lines[start:i], "\n"), token)
		}
	}
	if start < 0 {
		return false
	}
	return strings.Contains(strings.Join(lines[start:], "\n"), token)
}
