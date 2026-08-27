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
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
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

// Symbol is one top-level declaration: what it is, where it lives, how big it
// is, and the one line of doc that says why. Package-level nodes answer "what
// depends on what"; a refactor also asks "who owns this concern" and "how many
// places compute the same thing", and those are answered by grouping symbols.
type Symbol struct {
	Pkg  string `json:"pkg"`
	File string `json:"file"`
	Line int    `json:"line"`
	// Kind is func, method, type, const, or var.
	Kind string `json:"kind"`
	Name string `json:"name"`
	// Recv is the receiver type for a method, without pointer or type params.
	Recv string `json:"recv,omitempty"`
	// Sig is the rendered parameter and result list for a func or method.
	Sig string `json:"sig,omitempty"`
	// Doc is the first line of the doc comment, empty when undocumented.
	Doc      string `json:"doc,omitempty"`
	Exported bool   `json:"exported"`
	// Lines spans the declaration, so size shows without opening the file.
	Lines int `json:"lines"`
}

// Graph is the tool's JSON output.
type Graph struct {
	Module   string    `json:"module"`
	Packages []Package `json:"packages"`
	Edges    []Edge    `json:"edges"`
	// Violations lists edges that break the declared layering (core must not
	// import upper layers).
	Violations []string `json:"violations"`
	// Symbols is the per-declaration inventory, emitted only with -symbols
	// because it is an order of magnitude larger than the package graph.
	Symbols []Symbol `json:"symbols,omitempty"`
}

func main() {
	symbols := flag.Bool("symbols", false, "also emit the per-declaration symbol inventory")
	flag.Parse()
	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	g, err := build(root, *symbols)
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

// build walks the source trees and assembles the graph. withSymbols also
// collects the per-declaration inventory.
func build(root string, withSymbols bool) (*Graph, error) {
	pkgs := map[string]*Package{}
	edges := map[string]*Edge{} // key: from + "->" + to
	var syms []Symbol

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
			return scanFile(path, filepath.ToSlash(rel), pkgs, edges, withSymbols, &syms)
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
	sort.Slice(syms, func(i, j int) bool {
		if syms[i].Pkg != syms[j].Pkg {
			return syms[i].Pkg < syms[j].Pkg
		}
		if syms[i].File != syms[j].File {
			return syms[i].File < syms[j].File
		}
		return syms[i].Line < syms[j].Line
	})
	g.Symbols = syms
	return g, nil
}

// scanFile parses one file and records package stats, import edges, and the
// selector references made through each import.
func scanFile(path, pkgPath string, pkgs map[string]*Package, edges map[string]*Edge, withSymbols bool, syms *[]Symbol) error {
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
	if withSymbols {
		*syms = append(*syms, symbolsOf(f, fset, pkgPath, filepath.Base(path))...)
	}

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

// symbolsOf lists one file's top-level declarations.
func symbolsOf(f *ast.File, fset *token.FileSet, pkgPath, file string) []Symbol {
	var out []Symbol
	span := func(from, to token.Pos) (int, int) {
		start := fset.Position(from).Line
		return start, fset.Position(to).Line - start + 1
	}
	for _, d := range f.Decls {
		switch v := d.(type) {
		case *ast.FuncDecl:
			line, n := span(v.Pos(), v.End())
			s := Symbol{
				Pkg: pkgPath, File: file, Line: line, Lines: n,
				Kind: "func", Name: v.Name.Name, Sig: signature(v.Type),
				Doc: firstDocLine(v.Doc), Exported: v.Name.IsExported(),
			}
			if v.Recv != nil && len(v.Recv.List) > 0 {
				s.Kind, s.Recv = "method", receiverName(v.Recv.List[0].Type)
			}
			out = append(out, s)
		case *ast.GenDecl:
			kind := map[token.Token]string{token.TYPE: "type", token.CONST: "const", token.VAR: "var"}[v.Tok]
			if kind == "" {
				continue
			}
			for _, sp := range v.Specs {
				switch t := sp.(type) {
				case *ast.TypeSpec:
					line, n := span(t.Pos(), t.End())
					doc := firstDocLine(t.Doc)
					if doc == "" {
						doc = firstDocLine(v.Doc)
					}
					out = append(out, Symbol{Pkg: pkgPath, File: file, Line: line, Lines: n,
						Kind: kind, Name: t.Name.Name, Doc: doc, Exported: t.Name.IsExported()})
				case *ast.ValueSpec:
					for _, id := range t.Names {
						line, _ := span(id.Pos(), id.End())
						doc := firstDocLine(t.Doc)
						if doc == "" {
							doc = firstDocLine(v.Doc)
						}
						out = append(out, Symbol{Pkg: pkgPath, File: file, Line: line, Lines: 1,
							Kind: kind, Name: id.Name, Doc: doc, Exported: id.IsExported()})
					}
				}
			}
		}
	}
	return out
}

// signature renders a function's parameters and results, e.g.
// "(ctx context.Context, spec machine.Spec) (*machine.Access, error)".
func signature(t *ast.FuncType) string {
	var b strings.Builder
	b.WriteString("(" + fieldList(t.Params) + ")")
	if t.Results != nil && len(t.Results.List) > 0 {
		r := fieldList(t.Results)
		if len(t.Results.List) > 1 || len(t.Results.List[0].Names) > 0 {
			r = "(" + r + ")"
		}
		b.WriteString(" " + r)
	}
	return b.String()
}

func fieldList(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	parts := make([]string, 0, len(fl.List))
	for _, f := range fl.List {
		typ := types.ExprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typ)
			continue
		}
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
		parts = append(parts, strings.Join(names, ", ")+" "+typ)
	}
	return strings.Join(parts, ", ")
}

// receiverName strips the pointer and any type parameters from a receiver.
func receiverName(e ast.Expr) string {
	s := types.ExprString(e)
	s = strings.TrimPrefix(s, "*")
	if i := strings.IndexByte(s, '['); i > 0 {
		s = s[:i]
	}
	return s
}

// firstDocLine is the doc comment's first sentence-line, which is where Go
// convention puts what the symbol is.
func firstDocLine(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	for _, c := range g.List {
		t := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), "/*"))
		if t != "" {
			return t
		}
	}
	return ""
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

// layerOf maps a package path to its architectural layer. The fine-grained
// placement is docs/dev/architecture/layers.md (L0-L6), which internal/arch
// enforces per package; these are the coarse buckets this graph groups by:
//
//	usecase       L5 — one use case, one function
//	orchestration L4 — assembles services into one run
//	domain        L2a/L2b/L3 — consensus family, chain adapter, policy
//	core          L0/L1 — kernel vocabulary and primitives
//
// A package named here that no longer exists is a stale entry: every module
// package must land in a named bucket, and "other" means this map is behind.
func layerOf(path string) string {
	switch {
	case strings.HasPrefix(path, "cmd/"):
		return "entry"
	case strings.HasPrefix(path, "internal/app"):
		return "usecase"
	case strings.HasPrefix(path, "internal/mcp"),
		strings.HasPrefix(path, "internal/dashboard"),
		strings.HasPrefix(path, "internal/testspec"),
		strings.HasPrefix(path, "internal/testengine"),
		strings.HasPrefix(path, "internal/chainsetup"):
		return "orchestration"
	case strings.HasPrefix(path, "internal/chains"),
		strings.HasPrefix(path, "internal/consensus"),
		strings.HasPrefix(path, "internal/validatorset"),
		strings.HasPrefix(path, "internal/accounts"):
		return "domain"
	// netmap is the L1 module surface over core/netmap and the server set
	// (layers.md §3): a primitive, not an orchestrator.
	case strings.HasPrefix(path, "internal/core"),
		strings.HasPrefix(path, "internal/netmap"):
		return "core"
	case strings.HasPrefix(path, "internal/testkit"):
		return "support"
	// arch holds no production code and imports nothing; it has no layer.
	case strings.HasPrefix(path, "internal/arch"),
		strings.HasPrefix(path, "tests/"):
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
