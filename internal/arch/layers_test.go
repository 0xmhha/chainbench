package arch

import (
	"bytes"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// placement is the layer each package belongs to, read from the document.
type placement map[string]int

// readPlacement parses the module placement table out of layers.md §3.
//
// The table is the source of truth. Restating it here would create a second
// copy that can disagree with the document, which is exactly how the previous
// two measurements got the answer wrong.
func readPlacement(t *testing.T) placement {
	t.Helper()
	doc, err := readLayersDoc()
	if err != nil {
		t.Fatal(err)
	}
	sec, err := section(doc, "## 3. 모듈 배치")
	if err != nil {
		t.Fatal(err)
	}

	out := placement{}
	layer := -1
	for _, line := range strings.Split(sec, "\n") {
		if m := layerHeading.FindStringSubmatch(line); m != nil {
			layer = int(m[1][0] - '0')
			continue
		}
		if layer < 0 {
			continue
		}
		for _, pkg := range packagesIn(firstCell(line)) {
			out[pkg] = layer
		}
	}
	if len(out) == 0 {
		t.Fatalf("arch: parsed no packages from %s §3 — has the table's shape changed?", layersDoc)
	}
	return out
}

// edge is one import from src to dst, both relative to internal/.
type edge struct{ src, dst string }

// importGraph lists every internal package and the internal packages it
// imports, as the toolchain sees them.
func importGraph(t *testing.T) (pkgs []string, edges []edge) {
	t.Helper()
	// Run from the module root: `go list ./...` is relative to the working
	// directory, and from this package it would see only this package.
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \" \"}}", "./internal/...")
	cmd.Dir = moduleRoot
	// The toolchain's own complaint is the useful part: an import cycle, a
	// build failure, or a missing dependency all surface here, and "exit status
	// 1" alone would send a reader looking in the wrong place.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("arch: go list failed (%v):\n%s", err, strings.TrimSpace(stderr.String()))
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		path, imports, ok := strings.Cut(line, "|")
		if !ok || !strings.HasPrefix(path, modulePrefix) {
			continue
		}
		src := strings.TrimPrefix(path, modulePrefix)
		pkgs = append(pkgs, src)
		for _, imp := range strings.Fields(imports) {
			if strings.HasPrefix(imp, modulePrefix) {
				edges = append(edges, edge{src, strings.TrimPrefix(imp, modulePrefix)})
			}
		}
	}
	return pkgs, edges
}

// TestEveryPackageIsPlaced is the half that has been got wrong before: a check
// that quietly ignores what it does not recognise reports success for code
// nobody looked at.
func TestEveryPackageIsPlaced(t *testing.T) {
	place := readPlacement(t)
	pkgs, _ := importGraph(t)

	var missing []string
	for _, p := range pkgs {
		if _, ok := place[p]; !ok {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("these packages are not in %s §3, so nothing decides which layer they belong to:\n  %s\n"+
			"Add each to the table for its layer, or explain in the document why it is exempt.",
			layersDoc, strings.Join(missing, "\n  "))
	}

	// The reverse: a table entry naming a package that no longer exists means
	// the document is describing code that is gone.
	have := map[string]bool{}
	for _, p := range pkgs {
		have[p] = true
	}
	var ghosts []string
	for p := range place {
		if !have[p] {
			ghosts = append(ghosts, p)
		}
	}
	sort.Strings(ghosts)
	if len(ghosts) > 0 {
		t.Errorf("%s §3 names packages that do not exist:\n  %s", layersDoc, strings.Join(ghosts, "\n  "))
	}
}

// TestNoUpwardDependency is the rule the layering exists for: a lower layer
// must not know about a higher one, which is what lets L3 and L4 work against
// an interface while the chain implementations sit above them.
func TestNoUpwardDependency(t *testing.T) {
	place := readPlacement(t)
	_, edges := importGraph(t)

	var bad []string
	for _, e := range edges {
		src, ok1 := place[e.src]
		dst, ok2 := place[e.dst]
		if !ok1 || !ok2 {
			continue // reported by TestEveryPackageIsPlaced
		}
		if dst > src {
			bad = append(bad, fmtEdge(e, src, dst))
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Errorf("a lower layer imports a higher one:\n  %s\n"+
			"Either the import is wrong, or the placement in %s is.",
			strings.Join(bad, "\n  "), layersDoc)
	}
}

func fmtEdge(e edge, src, dst int) string {
	return "L" + string(rune('0'+src)) + " " + e.src + "  ->  L" + string(rune('0'+dst)) + " " + e.dst
}
