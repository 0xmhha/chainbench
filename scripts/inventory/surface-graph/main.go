// Command surface-graph inventories every feature on every surface (CLI, MCP,
// DSL) and reports which of them reach past the app layer, so the U track's
// progress is measured rather than estimated.
//
// It starts at each surface's registration site and follows calls within the
// same package, because a command that delegates to a helper still owns what
// the helper touches. Surfaces register differently, so each is found by its
// own shape: a cobra command is a composite literal with a Use field, an MCP
// tool is one with a chainbench_-prefixed Name, and a DSL action arrives as
// RegisterAction(name, handler{}) whose handler's methods hold the work.
//
// It parses with go/parser only, so it needs no build and stays usable while
// the tree is mid-refactor.
//
// Run it as:
//
//	go run ./scripts/inventory/surface-graph .
//
// The baseline it produced on 2026-09-05, which the U track counts down from:
// 157 entries (CLI 58, MCP 54, DSL 45 = 18 actions and 27 assertions), of which
// 109 reach past app.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// pkgFiles is one Go package: its files, the import names each file uses, and
// every function and method in it, so a surface entry can be followed into the
// helpers it calls without leaving the package.
type pkgFiles struct {
	dir     string
	imports map[string]map[string]string // file -> local name -> internal path
	funcs   map[string]*ast.FuncDecl     // "name" and "Type.method"
	fileOf  map[*ast.FuncDecl]string
	files   map[string]*ast.File
}

func loadPkg(fset *token.FileSet, dir string) *pkgFiles {
	p := &pkgFiles{dir: dir,
		imports: map[string]map[string]string{},
		funcs:   map[string]*ast.FuncDecl{},
		fileOf:  map[*ast.FuncDecl]string{},
		files:   map[string]*ast.File{},
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return p
	}
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		p.files[e.Name()] = f
		im := map[string]string{}
		for _, i := range f.Imports {
			ip := strings.Trim(i.Path.Value, `"`)
			if !strings.Contains(ip, "0xmhha/chainbench/internal/") {
				continue
			}
			name := ip[strings.LastIndex(ip, "/")+1:]
			if i.Name != nil {
				name = i.Name.Name
			}
			im[name] = strings.SplitN(ip, "/internal/", 2)[1]
		}
		p.imports[e.Name()] = im
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			key := fd.Name.Name
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				key = recvType(fd.Recv.List[0].Type) + "." + fd.Name.Name
			}
			p.funcs[key] = fd
			p.fileOf[fd] = e.Name()
		}
	}
	return p
}

func recvType(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.StarExpr:
		return recvType(v.X)
	case *ast.Ident:
		return v.Name
	}
	return ""
}

// collect gathers the internal packages reached from a node, following calls to
// this package's own functions and methods so a surface entry that delegates
// one or two hops is still attributed to what it ultimately touches.
func (p *pkgFiles) collect(n ast.Node, file string, depth int, seen map[string]bool, out map[string]bool) {
	if depth > 3 {
		return
	}
	im := p.imports[file]
	ast.Inspect(n, func(nd ast.Node) bool {
		switch v := nd.(type) {
		case *ast.SelectorExpr:
			if id, ok := v.X.(*ast.Ident); ok {
				if pkg, ok := im[id.Name]; ok && ast.IsExported(v.Sel.Name) {
					out[pkg] = true
				}
			}
		case *ast.CallExpr:
			switch fn := v.Fun.(type) {
			case *ast.Ident:
				p.follow(fn.Name, depth, seen, out)
			case *ast.SelectorExpr:
				// A method on a value of this package's own type.
				if id, ok := fn.X.(*ast.Ident); ok {
					if _, isPkg := im[id.Name]; !isPkg {
						p.follow(id.Name+"."+fn.Sel.Name, depth, seen, out)
					}
				}
			}
		case *ast.CompositeLit:
			// A registered handler type: follow every method it has.
			if id, ok := v.Type.(*ast.Ident); ok {
				for k := range p.funcs {
					if strings.HasPrefix(k, id.Name+".") {
						p.follow(k, depth, seen, out)
					}
				}
			}
		}
		return true
	})
}

func (p *pkgFiles) follow(key string, depth int, seen map[string]bool, out map[string]bool) {
	fd, ok := p.funcs[key]
	if !ok || seen[key] {
		return
	}
	seen[key] = true
	p.collect(fd, p.fileOf[fd], depth+1, seen, out)
}

// litString reads a string value out of a composite literal's field.
func litString(cl *ast.CompositeLit, key string) string {
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		k, ok := kv.Key.(*ast.Ident)
		if !ok || k.Name != key {
			continue
		}
		if b, ok := kv.Value.(*ast.BasicLit); ok {
			return strings.Trim(b.Value, `"`)
		}
	}
	return ""
}

type entry struct {
	surface string
	name    string
	where   string
	pkgs    []string
}

func main() {
	root := os.Args[1]
	fset := token.NewFileSet()
	var entries []entry

	// CLI: every cobra command literal, wherever the surface package lives.
	var cliDirs []string
	_ = filepath.Walk(filepath.Join(root, "cmd"), func(p string, fi os.FileInfo, _ error) error {
		if fi != nil && fi.IsDir() {
			cliDirs = append(cliDirs, p)
		}
		return nil
	})
	for _, d := range cliDirs {
		p := loadPkg(fset, d)
		for fname, f := range p.files {
			ast.Inspect(f, func(n ast.Node) bool {
				cl, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				sel, ok := cl.Type.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Command" {
					return true
				}
				use := litString(cl, "Use")
				if use == "" {
					return true
				}
				out := map[string]bool{}
				p.collect(cl, fname, 0, map[string]bool{}, out)
				// The leaf verb alone is ambiguous: "new" is a command under
				// both chain and keyring. The surface package disambiguates
				// them, and conflating the two put keyring's packages under
				// chain new in an earlier count.
				// The file, not the package, because "run" is both the suite
				// runner and a subcommand of upgrade inside package main, and
				// merging them attributed the handoff's packages to the suite.
				group := strings.TrimSuffix(fname, ".go")
				if d := filepath.Base(d); d != "chainbench" {
					group = d + "/" + group
				}
				entries = append(entries, entry{"CLI", group + ":" + strings.Fields(use)[0],
					strings.TrimPrefix(d, root+"/") + "/" + fname, keys(out)})
				return true
			})
		}
	}

	// MCP: every tool literal.
	mp := loadPkg(fset, filepath.Join(root, "internal", "mcp"))
	for fname, f := range mp.files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			name := litString(cl, "Name")
			if !strings.HasPrefix(name, "chainbench_") {
				return true
			}
			out := map[string]bool{}
			mp.collect(cl, fname, 0, map[string]bool{}, out)
			entries = append(entries, entry{"MCP", strings.TrimPrefix(name, "chainbench_"),
				"internal/mcp/" + fname, keys(out)})
			return true
		})
	}

	// DSL: every registered action and assertion.
	tp := loadPkg(fset, filepath.Join(root, "internal", "testhelper"))
	consts := map[string]string{}
	for _, f := range tp.files {
		ast.Inspect(f, func(n ast.Node) bool {
			vs, ok := n.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				return true
			}
			if b, ok := vs.Values[0].(*ast.BasicLit); ok {
				consts[vs.Names[0].Name] = strings.Trim(b.Value, `"`)
			}
			return true
		})
	}
	for fname, f := range tp.files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "RegisterAction" && sel.Sel.Name != "RegisterAssertion") {
				return true
			}
			name := ""
			switch a := call.Args[0].(type) {
			case *ast.Ident:
				name = consts[a.Name]
			case *ast.BasicLit:
				name = strings.Trim(a.Value, `"`)
			case *ast.SelectorExpr:
				// A field selector such as a.name means this call sits in a
				// loop over a table, and the names are in the table rather
				// than here. Those are collected separately below; missing
				// that is how an earlier count lost 17 assertions.
				if a.Sel.Name == "name" {
					return true
				}
				name = a.Sel.Name
			}
			if name == "" {
				return true
			}
			out := map[string]bool{}
			tp.collect(call.Args[1], fname, 0, map[string]bool{}, out)
			kind := "DSL"
			if sel.Sel.Name == "RegisterAssertion" {
				kind = "DSLa"
			}
			entries = append(entries, entry{kind, name, "internal/testhelper/" + fname, keys(out)})
			return true
		})
	}

	// Assertions registered from a table carry their names in the table, not at
	// the registration call, so the table is read directly. Each entry names an
	// assertion and the reader that performs it, and the reader is where the
	// work is, so it is followed like any other handler.
	for fname, f := range tp.files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			var name, reader string
			for _, el := range cl.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				k, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch k.Name {
				case "name":
					if id, ok := kv.Value.(*ast.Ident); ok {
						name = consts[id.Name]
					}
				case "read":
					if id, ok := kv.Value.(*ast.Ident); ok {
						reader = id.Name
					}
				}
			}
			if name == "" || reader == "" {
				return true
			}
			out := map[string]bool{}
			tp.follow(reader, 0, map[string]bool{}, out)
			entries = append(entries, entry{"DSLa", name, "internal/testhelper/" + fname, keys(out)})
			return true
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].name != entries[j].name {
			return entries[i].name < entries[j].name
		}
		return entries[i].surface < entries[j].surface
	})

	fmt.Printf("%-5s %-26s %-9s %s\n", "SURF", "FEATURE", "VIA", "PACKAGES")
	var viaApp, direct int
	for _, e := range entries {
		via := "core"
		hasApp := false
		for _, p := range e.pkgs {
			if p == "app" {
				hasApp = true
			}
		}
		switch {
		case hasApp && len(e.pkgs) == 1:
			via = "app"
			viaApp++
		case hasApp:
			via = "app+core"
			direct++
		case len(e.pkgs) == 0:
			via = "-"
		default:
			direct++
		}
		fmt.Printf("%-5s %-26s %-9s %s\n", e.surface, e.name, via, strings.Join(e.pkgs, " "))
	}
	fmt.Printf("\nentries: %d — app-only %d, reaching past app %d\n", len(entries), viaApp, direct)
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
