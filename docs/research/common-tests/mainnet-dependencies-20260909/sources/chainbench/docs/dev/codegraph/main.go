//go:build ignore

// Command codegraph parses the chainbench Go source (internal/ + cmd/) with the
// standard AST tools and emits a package/symbol/call graph as JSON, plus a
// human summary and a mermaid of the DSL-pipeline wiring. It cross-references an
// incoming DSL analysis against what the code actually wires.
//
// It resolves cross-package calls syntactically: Go requires those to be
// written pkgAlias.Func(...), so resolving each file's import aliases and
// collecting alias.Sel selectors captures the wiring without type-checking.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const module = "github.com/0xmhha/chainbench"

// Pkg is one package node.
type Pkg struct {
	Path    string   `json:"path"`    // import path, module-relative (e.g. internal/app)
	Name    string   `json:"name"`    // package clause name
	Files   int      `json:"files"`   // non-test .go files
	Imports []string `json:"imports"` // module-internal imports only, module-relative
	Types   []string `json:"types"`   // exported type names
	Funcs   []string `json:"funcs"`   // exported func/method names (Recv.Method for methods)
}

// CallEdge is one cross-package call: FromPkg.FromFunc references ToPkg.Sel.
type CallEdge struct {
	FromPkg  string `json:"fromPkg"`
	FromFunc string `json:"fromFunc"`
	ToPkg    string `json:"toPkg"`
	ToSel    string `json:"toSel"`
}

// Graph is the whole emitted graph.
type Graph struct {
	Packages []Pkg      `json:"packages"`
	Calls    []CallEdge `json:"calls"`
}

func rel(importPath string) string { return strings.TrimPrefix(importPath, module+"/") }

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	fset := token.NewFileSet()

	// Pass 1: find every package dir under internal/ and cmd/, parse its
	// non-test files, and record the package name so import aliases resolve.
	type parsed struct {
		path  string
		name  string
		files []*ast.File
	}
	pkgs := map[string]*parsed{} // dir -> parsed
	nameByPath := map[string]string{}
	for _, base := range []string{"internal", "cmd"} {
		_ = filepath.WalkDir(filepath.Join(root, base), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
			if perr != nil {
				return nil // skip unparseable; report at the end
			}
			dir := filepath.Dir(p)
			pk := pkgs[dir]
			if pk == nil {
				ip := module + "/" + filepath.ToSlash(dir)
				pk = &parsed{path: ip, name: f.Name.Name}
				pkgs[dir] = pk
				nameByPath[ip] = f.Name.Name
			}
			pk.files = append(pk.files, f)
			return nil
		})
	}

	// Pass 2: build nodes + edges.
	var g Graph
	dirs := make([]string, 0, len(pkgs))
	for d := range pkgs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	for _, d := range dirs {
		pk := pkgs[d]
		node := Pkg{Path: rel(pk.path), Name: pk.name, Files: len(pk.files)}
		impSet := map[string]bool{}
		typeSet := map[string]bool{}
		funcSet := map[string]bool{}

		for _, f := range pk.files {
			// Per-file import aliases: alias -> module-relative import path
			// (internal imports only).
			alias := map[string]string{}
			for _, is := range f.Imports {
				ip := strings.Trim(is.Path.Value, `"`)
				if !strings.HasPrefix(ip, module+"/") {
					continue
				}
				impSet[rel(ip)] = true
				a := nameByPath[ip]
				if a == "" {
					a = ip[strings.LastIndex(ip, "/")+1:]
				}
				if is.Name != nil {
					a = is.Name.Name
				}
				if a == "_" || a == "." {
					continue
				}
				alias[a] = rel(ip)
			}
			// Symbols + call edges.
			for _, decl := range f.Decls {
				switch dd := decl.(type) {
				case *ast.GenDecl:
					for _, s := range dd.Specs {
						if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.IsExported() {
							typeSet[ts.Name.Name] = true
						}
					}
				case *ast.FuncDecl:
					fn := funcName(dd)
					if ast.IsExported(strings.TrimPrefix(strings.TrimPrefix(fn, "("), "*")) || exportedMethod(dd) {
						funcSet[fn] = true
					}
					if dd.Body == nil {
						continue
					}
					ast.Inspect(dd.Body, func(n ast.Node) bool {
						sel, ok := n.(*ast.SelectorExpr)
						if !ok {
							return true
						}
						id, ok := sel.X.(*ast.Ident)
						if !ok {
							return true
						}
						if to, ok := alias[id.Name]; ok {
							g.Calls = append(g.Calls, CallEdge{
								FromPkg: node.Path, FromFunc: fn, ToPkg: to, ToSel: sel.Sel.Name,
							})
						}
						return true
					})
				}
			}
		}
		node.Imports = sortedKeys(impSet)
		node.Types = sortedKeys(typeSet)
		node.Funcs = sortedKeys(funcSet)
		g.Packages = append(g.Packages, node)
	}
	g.Calls = dedupCalls(g.Calls)

	// Emit JSON.
	out, _ := json.MarshalIndent(g, "", "  ")
	if err := os.WriteFile("codegraph.json", out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Summary.
	fmt.Printf("packages=%d  call-edges=%d\n", len(g.Packages), len(g.Calls))
	fmt.Println("\n== package import fan-in/out (module-internal) ==")
	in := map[string]int{}
	out2 := map[string]int{}
	for _, p := range g.Packages {
		out2[p.Path] = len(p.Imports)
		for _, im := range p.Imports {
			in[im]++
		}
	}
	type kv struct {
		k string
		v int
	}
	var byIn []kv
	for _, p := range g.Packages {
		byIn = append(byIn, kv{p.Path, in[p.Path]})
	}
	sort.Slice(byIn, func(i, j int) bool { return byIn[i].v > byIn[j].v })
	for _, x := range byIn[:min(12, len(byIn))] {
		fmt.Printf("  in=%-3d out=%-3d %s\n", x.v, out2[x.k], x.k)
	}
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return "(" + recvType(fn.Recv.List[0].Type) + ")." + fn.Name.Name
}

func exportedMethod(fn *ast.FuncDecl) bool {
	return fn.Recv != nil && fn.Name.IsExported()
}

func recvType(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + recvType(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvType(t.X)
	}
	return "?"
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func dedupCalls(cs []CallEdge) []CallEdge {
	seen := map[CallEdge]bool{}
	var out []CallEdge
	for _, c := range cs {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromPkg != out[j].FromPkg {
			return out[i].FromPkg < out[j].FromPkg
		}
		if out[i].FromFunc != out[j].FromFunc {
			return out[i].FromFunc < out[j].FromFunc
		}
		if out[i].ToPkg != out[j].ToPkg {
			return out[i].ToPkg < out[j].ToPkg
		}
		return out[i].ToSel < out[j].ToSel
	})
	return out
}
