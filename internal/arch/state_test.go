package arch

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// writesFiles is the set of packages the document allows to write files, and
// whether the document calls that placement correct.
type writesFiles map[string]verdict

// verdict is how the document judges a package's file writing.
type verdict struct {
	// Allowed is true for ✅ (this package owns that state) and ◐ (under
	// review); false for ❌, a violation the document already names.
	Allowed bool
	// Mark is the symbol as written, for the failure message.
	Mark string
}

// readWriters parses the file-writing table out of layers.md §5.
//
// Like the placement table, the document is the source: a copy here could
// disagree with it, and then the test would be enforcing something nobody
// wrote down.
func readWriters(t *testing.T) writesFiles {
	t.Helper()
	doc, err := readLayersDoc()
	if err != nil {
		t.Fatal(err)
	}
	sec, err := section(doc, "### 파일을 쓰는 패키지")
	if err != nil {
		t.Fatal(err)
	}

	out := writesFiles{}
	for _, line := range strings.Split(sec, "\n") {
		cell := firstCell(line)
		if cell == "" {
			continue
		}
		mark := verdictMark(line)
		if mark == "" {
			continue
		}
		for _, pkg := range packagesIn(cell) {
			out[pkg] = verdict{Allowed: mark != "❌", Mark: mark}
		}
	}
	if len(out) == 0 {
		t.Fatalf("arch: parsed no packages from %s §5 — has the table's shape changed?", layersDoc)
	}
	return out
}

// verdictMark reads the ✅/◐/❌ from a row's last cell.
func verdictMark(line string) string {
	for _, m := range []string{"✅", "◐", "❌"} {
		if strings.Contains(line, m) {
			return m
		}
	}
	return ""
}

// TestOnlyListedPackagesWriteFiles is the state-ownership rule. The control
// plane belongs to core/session and the data plane to the file interface; a package
// that writes files outside that is either a listed exception or a new one
// nobody decided on.
func TestOnlyListedPackagesWriteFiles(t *testing.T) {
	listed := readWriters(t)
	actual := packagesWritingFiles(t)

	var unlisted []string
	for _, pkg := range actual {
		if _, ok := listed[pkg]; !ok {
			unlisted = append(unlisted, pkg)
		}
	}
	sort.Strings(unlisted)
	if len(unlisted) > 0 {
		t.Errorf("these packages write files but %s §5 does not list them:\n  %s\n"+
			"Write through filestore.Store, or add the package to the table with a verdict.",
			layersDoc, strings.Join(unlisted, "\n  "))
	}

	// A listed package that no longer writes anything is a stale exception: it
	// keeps a violation on the books after the work to remove it was done.
	have := map[string]bool{}
	for _, p := range actual {
		have[p] = true
	}
	var stale []string
	for pkg, v := range listed {
		if !have[pkg] {
			stale = append(stale, pkg+" ("+v.Mark+")")
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("%s §5 lists packages that no longer write files:\n  %s\n"+
			"Remove them from the table — a stale exception hides the next real one.",
			layersDoc, strings.Join(stale, "\n  "))
	}
}

// packagesWritingFiles finds every internal package containing a direct file
// write, by reading the sources rather than by running them.
//
// The check is textual on purpose: an import-graph check would miss a package
// that writes through a helper, and a runtime check would only see the paths a
// test happened to exercise.
func packagesWritingFiles(t *testing.T) []string {
	t.Helper()
	const root = moduleRoot + "/internal"
	found := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !writesToDisk(string(b)) {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		found[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("arch: walk %s: %v", root, err)
	}

	out := make([]string, 0, len(found))
	for p := range found {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// writesToDisk reports whether source creates or writes a file, ignoring
// occurrences inside comments so a mention in prose is not a violation.
func writesToDisk(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		code, _, _ := strings.Cut(line, "//")
		if strings.Contains(code, "os.WriteFile(") || strings.Contains(code, "os.MkdirAll(") ||
			strings.Contains(code, "os.Create(") {
			return true
		}
	}
	return false
}
