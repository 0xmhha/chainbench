package arch

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// treeDoc is the document that answers "what packages are there, and what does
// each one support", relative to this package.
const treeDoc = "../../docs/dev/architecture/package-tree.md"

// This file exists because a count written by hand goes stale silently. The
// layers document's §3 heading said "43개 전수" while internal/ held 48: the
// table below it was complete (TestEveryPackageIsPlaced proves that every
// release), so nothing failed, and the only thing wrong was the number a reader
// sees first. A stale count is worse than no count — it reads as a measurement.
//
// So the totals in both documents are measured here instead of trusted. What is
// pinned is only what a reader would otherwise have to believe: how many
// packages there are and how many lines they hold, per group.

// group is one row of the package-tree §0 table: a path prefix and the totals
// claimed for it.
type group struct {
	prefix string // "internal/", "cmd/", "scripts/"; "" means every package
	pkgs   int
	lines  int
}

var (
	// `internal/` | 48 | 47,583
	treeRow = regexp.MustCompile("^\\| (?:\\*\\*)?`?([a-z/]*)`?(?:\\*\\*)?[^|]*\\| (?:\\*\\*)?([0-9,]+)(?:\\*\\*)?[^|]*\\| (?:\\*\\*)?([0-9,]+)")
	// ## 3. 모듈 배치 (43개 전수 — ...)
	placementCount = regexp.MustCompile(`## 3\. 모듈 배치 \((\d+)개 전수`)
	// The "합계" row names no path; it stands for every package.
	totalRow = regexp.MustCompile(`^\| \*\*합계\*\*`)
)

// measureGroups counts packages and non-test lines per top-level group, as the
// toolchain sees them.
//
// Non-test lines means every .go file in the package's directory that is not a
// _test.go file — not GoFiles, which would drop whatever a build constraint
// excludes on this platform and make the number depend on where it was run.
func measureGroups(t *testing.T) map[string]group {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{.Dir}}", "./...")
	cmd.Dir = moduleRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("arch: go list failed (%v):\n%s", err, strings.TrimSpace(stderr.String()))
	}

	got := map[string]group{}
	add := func(prefix string, lines int) {
		g := got[prefix]
		g.prefix, g.pkgs, g.lines = prefix, g.pkgs+1, g.lines+lines
		got[prefix] = g
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		path, dir, ok := strings.Cut(line, "|")
		if !ok || !strings.HasPrefix(path, modulePath+"/") {
			continue
		}
		rel := strings.TrimPrefix(path, modulePath+"/")
		n, err := nonTestLines(dir)
		if err != nil {
			t.Fatalf("arch: %s: %v", rel, err)
		}
		add("", n)
		// Every directory prefix, not just the first segment: the document is
		// free to name a group at whatever depth reads best
		// ("scripts/inventory/" rather than "scripts/").
		for i, c := range rel {
			if c == '/' {
				add(rel[:i+1], n)
			}
		}
	}
	return got
}

// nonTestLines counts the lines of every non-test .go file in dir.
func nonTestLines(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return 0, err
		}
		total += bytes.Count(b, []byte("\n"))
		if len(b) > 0 && b[len(b)-1] != '\n' {
			total++
		}
	}
	return total, nil
}

// readTreeGroups parses the §0 table out of the package-tree document.
func readTreeGroups(t *testing.T) []group {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(treeDoc))
	if err != nil {
		t.Fatalf("arch: read %s: %v", treeDoc, err)
	}
	sec, err := section(string(b), "## 0. 전수")
	if err != nil {
		t.Fatal(err)
	}

	var out []group
	for _, line := range strings.Split(sec, "\n") {
		m := treeRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		prefix := m[1]
		if totalRow.MatchString(line) {
			prefix = ""
		} else if prefix == "" {
			continue // a header or separator row, not a group
		}
		out = append(out, group{prefix: prefix, pkgs: atoi(t, m[2]), lines: atoi(t, m[3])})
	}
	if len(out) == 0 {
		t.Fatalf("arch: parsed no rows from %s §0 — has the table's shape changed?", treeDoc)
	}
	return out
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		t.Fatalf("arch: %q is not a number: %v", s, err)
	}
	return n
}

func name(prefix string) string {
	if prefix == "" {
		return "합계 (every package)"
	}
	return prefix
}

// TestPackageTreeTotalsAreMeasured checks every row of the package-tree §0
// table against the toolchain. The table is what a reader believes before
// reading 240 lines of tree, so it is the one place a wrong number costs most.
func TestPackageTreeTotalsAreMeasured(t *testing.T) {
	got := measureGroups(t)
	for _, want := range readTreeGroups(t) {
		g, ok := got[want.prefix]
		if !ok {
			t.Errorf("%s §0 claims a group %q that holds no packages", treeDoc, name(want.prefix))
			continue
		}
		if g.pkgs != want.pkgs {
			t.Errorf("%s §0: %s has %d packages, the table says %d", treeDoc, name(want.prefix), g.pkgs, want.pkgs)
		}
		if g.lines != want.lines {
			t.Errorf("%s §0: %s has %d non-test lines, the table says %d", treeDoc, name(want.prefix), g.lines, want.lines)
		}
	}
}

// TestPackageTreeCoversEveryGroup is the other direction: a table that lists
// only the groups whose numbers still happen to be right would pass the test
// above while hiding a whole group. It asks only about top-level groups — a row
// may name a group at any depth, and "scripts/inventory/" covers "scripts/".
func TestPackageTreeCoversEveryGroup(t *testing.T) {
	rows := readTreeGroups(t)
	covered := func(top string) bool {
		for _, g := range rows {
			if strings.HasPrefix(g.prefix, top) {
				return true
			}
		}
		return false
	}
	for prefix := range measureGroups(t) {
		if prefix == "" || strings.Count(prefix, "/") != 1 {
			continue // "" is the total, deeper prefixes are not top-level groups
		}
		if !covered(prefix) {
			t.Errorf("%s §0 does not list %s, which holds packages", treeDoc, name(prefix))
		}
	}
}

// TestLayersPlacementCountIsMeasured pins the number in the layers §3 heading,
// which is the one that went stale and started this file.
func TestLayersPlacementCountIsMeasured(t *testing.T) {
	doc, err := readLayersDoc()
	if err != nil {
		t.Fatal(err)
	}
	m := placementCount.FindStringSubmatch(doc)
	if m == nil {
		t.Fatalf("arch: %s has no \"## 3. 모듈 배치 (N개 전수\" heading — the shape changed, so the count is no longer checked", layersDoc)
	}
	want := atoi(t, m[1])
	got := measureGroups(t)["internal/"].pkgs
	if got != want {
		t.Errorf("%s §3 heading says %d packages, internal/ holds %d", layersDoc, want, got)
	}
}
