package arch

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The §0 table was measured and the 240 lines under it were not.
//
// TestPackageTreeTotalsAreMeasured pins three rows, so those three stayed true
// while everything a reader actually consults went stale. Measured 2026-09-21:
// 32 of the 70 per-package figures disagreed with the toolchain, some by a lot
// — `consensus/upgrade` read 1,949 lines and holds 164, `dsl` read 1,192 and
// holds 1,889 — and two of the five section headings were wrong about how many
// packages they even cover.
//
// A wrong number in a document graded [측정] is worse than no number: it reads
// as something somebody checked. So the body is checked too.

var (
	// `├── home            52  [L0] ...`, `internal/resource  3,080  [L1] ...`,
	// or `├── (dsl)      1,192  ...` for the directory's own package.
	treeEntry = regexp.MustCompile(`^((?:[│ ]*[├└]── )?)(\(?[a-z][a-z0-9/._-]*\)?)(\s+)([0-9,]+)(\s)`)
	// A line naming the subtree the entries below it hang from: `internal/core/`.
	treeRoot = regexp.MustCompile(`^([a-z][a-z0-9/]*/)(\s|$)`)
	// `## 1. ` + "`internal/core`" + ` — 22패키지 16,108줄 · ...`
	treeSection = regexp.MustCompile(`^## \d+\. .*?(\d+)패키지 ([0-9,]+)줄`)
)

// treeLeaf is one package the document names, with the line count it claims.
type treeLeaf struct {
	line    int    // 1-based, for the message
	pkg     string // module-relative directory, e.g. internal/core/node
	claimed int
	section string
}

// readTreeLeaves walks the fenced blocks and resolves each entry to a package.
//
// An entry's name is relative to the nearest preceding root line, so the tree
// reads as a tree; a name in parentheses is the root's own package; a name that
// already carries its prefix stands alone. When that does not land on a real
// package the name is matched by suffix, which is how the one genuinely nested
// entry (chains/stablenet/govbind) resolves without the document having to
// spell the path twice.
func readTreeLeaves(t *testing.T, known map[string]bool) []treeLeaf {
	t.Helper()
	doc := readTreeDoc(t)
	var (
		out     []treeLeaf
		inBlock bool
		root    string
		section string
	)
	for i, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimSpace(line)
		}
		if strings.HasPrefix(line, "```") {
			inBlock, root = !inBlock, ""
			continue
		}
		if !inBlock {
			continue
		}
		if m := treeRoot.FindStringSubmatch(line); m != nil {
			root = strings.TrimSuffix(m[1], "/")
			continue
		}
		m := treeEntry.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[2]
		var pkg string
		switch {
		case strings.HasPrefix(name, "("):
			pkg = root
		case strings.HasPrefix(name, "internal/"), strings.HasPrefix(name, "cmd/"), strings.HasPrefix(name, "scripts/"):
			pkg = name
		case root != "":
			pkg = root + "/" + name
		default:
			pkg = name
		}
		if !known[pkg] {
			var found []string
			for k := range known {
				if strings.HasSuffix(k, "/"+name) && (root == "" || strings.HasPrefix(k, root)) {
					found = append(found, k)
				}
			}
			if len(found) == 1 {
				pkg = found[0]
			}
		}
		out = append(out, treeLeaf{line: i + 1, pkg: pkg, claimed: atoi(t, m[4]), section: section})
	}
	return out
}

// TestPackageTreeBodyIsMeasured holds every per-package figure to the toolchain.
func TestPackageTreeBodyIsMeasured(t *testing.T) {
	lines := measurePackages(t)
	known := map[string]bool{}
	for k := range lines {
		known[k] = true
	}
	leaves := readTreeLeaves(t, known)
	if len(leaves) == 0 {
		t.Fatalf("no package entries found in %s, so the parse is wrong rather than the tree empty", treeDoc)
	}
	seen := map[string]bool{}
	for _, l := range leaves {
		got, ok := lines[l.pkg]
		if !ok {
			t.Errorf("%s:%d names %q, which is not a package", treeDoc, l.line, l.pkg)
			continue
		}
		if seen[l.pkg] {
			t.Errorf("%s:%d names %s a second time", treeDoc, l.line, l.pkg)
		}
		seen[l.pkg] = true
		if got != l.claimed {
			t.Errorf("%s:%d: %s holds %d non-test lines, the tree says %d\n"+
				"Refresh every figure with: python3 scripts/refresh-package-tree.py", treeDoc, l.line, l.pkg, got, l.claimed)
		}
	}
	// The other direction: a tree that quietly drops a package reads as complete.
	var missing []string
	for pkg := range lines {
		if !seen[pkg] {
			missing = append(missing, pkg)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s names no entry for these packages:\n  %s", treeDoc, strings.Join(missing, "\n  "))
	}
	t.Logf("%d package figures measured", len(leaves))
}

// TestPackageTreeSectionsSumTheirOwnEntries checks each section heading against
// what that section lists, which is the claim a reader meets before the tree.
func TestPackageTreeSectionsSumTheirOwnEntries(t *testing.T) {
	lines := measurePackages(t)
	known := map[string]bool{}
	for k := range lines {
		known[k] = true
	}
	perSection := map[string]struct{ pkgs, lines int }{}
	for _, l := range readTreeLeaves(t, known) {
		n, ok := lines[l.pkg]
		if !ok {
			continue
		}
		s := perSection[l.section]
		s.pkgs, s.lines = s.pkgs+1, s.lines+n
		perSection[l.section] = s
	}
	checked := 0
	for _, line := range strings.Split(readTreeDoc(t), "\n") {
		m := treeSection.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		head := strings.TrimSpace(line)
		got, ok := perSection[head]
		if !ok {
			continue
		}
		checked++
		if got.pkgs != atoi(t, m[1]) {
			t.Errorf("%s: %q lists %d packages, its heading says %s", treeDoc, shorten(head), got.pkgs, m[1])
		}
		if got.lines != atoi(t, m[2]) {
			t.Errorf("%s: %q lists %d lines, its heading says %s", treeDoc, shorten(head), got.lines, m[2])
		}
	}
	if checked == 0 {
		t.Fatalf("no section headings carried a package/line summary, so the parse is wrong")
	}
	t.Logf("%d section headings measured", checked)
}

func shorten(s string) string {
	if i := strings.Index(s, "—"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// measurePackages counts non-test lines per package directory, module-relative.
func measurePackages(t *testing.T) map[string]int {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{.Dir}}", "./...")
	cmd.Dir = moduleRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("arch: go list failed (%v):\n%s", err, strings.TrimSpace(stderr.String()))
	}
	got := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		path, dir, ok := strings.Cut(line, "|")
		if !ok || !strings.HasPrefix(path, modulePath+"/") {
			continue
		}
		n, err := nonTestLines(dir)
		if err != nil {
			t.Fatalf("arch: %s: %v", path, err)
		}
		got[strings.TrimPrefix(path, modulePath+"/")] = n
	}
	return got
}

// readTreeDoc reads the package tree once per call site.
func readTreeDoc(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(treeDoc))
	if err != nil {
		t.Fatalf("arch: read %s: %v", treeDoc, err)
	}
	return string(b)
}
