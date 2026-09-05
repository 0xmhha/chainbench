// Surface inventory: what each surface registers, and which of those
// registrations reach past the app layer.
//
// This lives beside the rules it serves rather than inside the tool that prints
// it, because the U track's ratchet test and that tool must count the same
// thing. Two implementations of "how many registrations bypass app" would
// drift, and the number is the point.
//
// Each surface is found by its own registration shape. A cobra command is a
// composite literal with a Use field; an MCP tool is one with a
// chainbench_-prefixed Name; a DSL action or assertion arrives as
// RegisterAction or RegisterAssertion, and the handler type it is given holds
// the work. Table-driven assertions name themselves in the table rather than at
// the call, so the table is read too — missing that once cost 17 of them.
//
// Calls are followed within the registering package, because a command that
// delegates to a helper still owns what the helper touches. Parsing is
// go/parser only, so this needs no build and stays usable mid-refactor.

package arch

import (
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

// Entry is one registered feature on one surface.
type Entry struct {
	// Surface is CLI, MCP, DSL for an action, or DSLa for an assertion.
	Surface string
	// Name identifies the feature within its surface. A CLI name carries the
	// file it is declared in, because "new" is a command under both chain and
	// keyring, and "run" is both the suite runner and a subcommand of upgrade.
	Name string
	// Where is the file, for a reader who wants to go look.
	Where string
	// Pkgs are the internal packages the entry reaches, "app" among them.
	Pkgs []string
}

// Via says how an entry reaches its work.
func (e Entry) Via() string {
	hasApp := false
	for _, p := range e.Pkgs {
		if p == "app" {
			hasApp = true
		}
	}
	switch {
	case hasApp && len(e.Pkgs) == 1:
		return "app"
	case hasApp:
		return "app+core"
	case len(e.Pkgs) == 0:
		return "-"
	default:
		return "core"
	}
}

// ReachesPastApp reports whether the entry touches anything below app. This is
// the count the U track drives to zero.
func (e Entry) ReachesPastApp() bool {
	v := e.Via()
	return v == "core" || v == "app+core"
}

// Entries walks the three surfaces rooted at a module directory and reports
// every registered feature with the internal packages it reaches.
func Entries(root string) []Entry {
	fset := token.NewFileSet()
	var entries []Entry

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
				entries = append(entries, Entry{"CLI", group + ":" + strings.Fields(use)[0],
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
			entries = append(entries, Entry{"MCP", strings.TrimPrefix(name, "chainbench_"),
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
			entries = append(entries, Entry{kind, name, "internal/testhelper/" + fname, keys(out)})
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
			entries = append(entries, Entry{"DSLa", name, "internal/testhelper/" + fname, keys(out)})
			return true
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Surface < entries[j].Surface
	})
	return entries
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
