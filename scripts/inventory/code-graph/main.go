// Command code-graph extracts an AST-derived graph of the chainbench module:
// packages as nodes, imports as edges, and the exported symbols each edge
// actually references. It parses source with go/parser only — no build, no
// dependency on golang.org/x/tools — mirroring scripts/inventory/chain-flag-graph.
//
// Usage:
//
//	go run ./scripts/inventory/code-graph [module-root] > graph.json
//
// The output is the measurement behind docs/dev/architecture/code-graph.md;
// regenerate it after structural refactors instead of editing numbers by hand.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const modulePath = "github.com/0xmhha/chainbench"

// Package is one node of the graph.
type Package struct {
	// Path is the import path relative to the module root (e.g. "internal/engine").
	Path string `json:"path"`
	// Layer is the architectural layer the path maps to (see layerOf).
	Layer string `json:"layer"`
	// Files is the number of non-test .go files.
	Files int `json:"files"`
	// TestFiles is the number of _test.go files.
	TestFiles int `json:"testFiles"`
	// Lines is the total non-test source line count.
	Lines int `json:"lines"`
	// ExportedFuncs and ExportedTypes size the package's public surface.
	ExportedFuncs int `json:"exportedFuncs"`
	ExportedTypes int `json:"exportedTypes"`
	// FanIn / FanOut count module-internal importer / imported packages.
	FanIn  int `json:"fanIn"`
	FanOut int `json:"fanOut"`
}

// Edge is one import relation, annotated with the symbols it touches.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Refs maps referenced exported symbol -> use count, so the edge shows
	// which part of the target's surface the importer actually depends on.
	Refs map[string]int `json:"refs"`
}

// Graph is the tool's JSON output.
type Graph struct {
	Module   string    `json:"module"`
	Packages []Package `json:"packages"`
	Edges    []Edge    `json:"edges"`
	// Violations lists edges that break the declared layering (core must not
	// import upper layers).
	Violations []string `json:"violations"`
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	g, err := build(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// build walks the source trees and assembles the graph.
func build(root string) (*Graph, error) {
	pkgs := map[string]*Package{}
	edges := map[string]*Edge{} // key: from + "->" + to

	for _, tree := range []string{"cmd", "internal", "scripts", "tests"} {
		dir := filepath.Join(root, tree)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			rel, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			return scanFile(path, filepath.ToSlash(rel), pkgs, edges)
		})
		if err != nil {
			return nil, err
		}
	}

	g := &Graph{Module: modulePath}
	fanIn := map[string]int{}
	fanOut := map[string]int{}
	for _, e := range edges {
		fanOut[e.From]++
		fanIn[e.To]++
		g.Edges = append(g.Edges, *e)
	}
	for _, p := range pkgs {
		p.FanIn = fanIn[p.Path]
		p.FanOut = fanOut[p.Path]
		g.Packages = append(g.Packages, *p)
	}
	sort.Slice(g.Packages, func(i, j int) bool { return g.Packages[i].Path < g.Packages[j].Path })
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].From != g.Edges[j].From {
			return g.Edges[i].From < g.Edges[j].From
		}
		return g.Edges[i].To < g.Edges[j].To
	})
	g.Violations = violations(g.Edges)
	return g, nil
}

// scanFile parses one file and records package stats, import edges, and the
// selector references made through each import.
func scanFile(path, pkgPath string, pkgs map[string]*Package, edges map[string]*Edge) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	p, ok := pkgs[pkgPath]
	if !ok {
		p = &Package{Path: pkgPath, Layer: layerOf(pkgPath)}
		pkgs[pkgPath] = p
	}
	if strings.HasSuffix(path, "_test.go") {
		p.TestFiles++
		return nil // tests are consumers by design; keep the graph production-only
	}
	p.Files++
	p.Lines += fset.File(f.Pos()).LineCount()
	countExports(f, p)

	// alias -> module-internal import path
	aliases := map[string]string{}
	for _, imp := range f.Imports {
		ip, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !strings.HasPrefix(ip, modulePath+"/") {
			continue
		}
		rel := strings.TrimPrefix(ip, modulePath+"/")
		name := rel[strings.LastIndex(rel, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		aliases[name] = rel
		key := pkgPath + "->" + rel
		if _, ok := edges[key]; !ok {
			edges[key] = &Edge{From: pkgPath, To: rel, Refs: map[string]int{}}
		}
	}
	if len(aliases) == 0 {
		return nil
	}
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Obj != nil { // Obj != nil means a local binding shadows the alias
			return true
		}
		if to, ok := aliases[id.Name]; ok {
			edges[pkgPath+"->"+to].Refs[sel.Sel.Name]++
		}
		return true
	})
	return nil
}

// countExports tallies the exported surface of one file.
func countExports(f *ast.File, p *Package) {
	for _, d := range f.Decls {
		switch v := d.(type) {
		case *ast.FuncDecl:
			if v.Name.IsExported() && v.Recv == nil {
				p.ExportedFuncs++
			}
		case *ast.GenDecl:
			if v.Tok != token.TYPE {
				continue
			}
			for _, s := range v.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.IsExported() {
					p.ExportedTypes++
				}
			}
		}
	}
}

// layerOf maps a package path to its architectural layer, matching the layer
// model in docs/dev/architecture/software-architecture.md.
func layerOf(path string) string {
	switch {
	case strings.HasPrefix(path, "cmd/"):
		return "entry"
	case strings.HasPrefix(path, "internal/engine"),
		strings.HasPrefix(path, "internal/mcp"),
		strings.HasPrefix(path, "internal/dashboard"),
		strings.HasPrefix(path, "internal/testspec"),
		strings.HasPrefix(path, "internal/netcompose"),
		strings.HasPrefix(path, "internal/chainsetup"):
		return "orchestration"
	case strings.HasPrefix(path, "internal/chains"),
		strings.HasPrefix(path, "internal/consensus"),
		strings.HasPrefix(path, "internal/validatorset"),
		strings.HasPrefix(path, "internal/accounts"):
		return "domain"
	case strings.HasPrefix(path, "internal/core"):
		return "core"
	case strings.HasPrefix(path, "internal/keygen"),
		strings.HasPrefix(path, "internal/keymat"),
		strings.HasPrefix(path, "internal/serverset"),
		strings.HasPrefix(path, "internal/testkit"):
		return "support"
	case strings.HasPrefix(path, "tests/"):
		return "tests"
	case strings.HasPrefix(path, "scripts/"):
		return "scripts"
	default:
		return "other"
	}
}

// violations reports layering breaks: core packages importing any non-core,
// non-support module package.
func violations(edges []Edge) []string {
	var out []string
	for _, e := range edges {
		if layerOf(e.From) != "core" {
			continue
		}
		if l := layerOf(e.To); l != "core" && l != "support" && l != "domain" {
			out = append(out, e.From+" -> "+e.To+" ("+l+")")
		}
	}
	return out
}
