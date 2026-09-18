package arch

import (
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

// Use says who reaches an exported symbol, which is the difference between a
// number and a list of things to do.
//
// A8 has been reported three times as one figure — 230, then 896, then 928 —
// and each time it was read as "this much is dead". It never was. The count is
// of symbols no OTHER package names with a selector, and four different things
// produce that: a symbol its own package uses (needlessly exported), one only
// tests reach (a test surface), one dispatched by name through a registry
// (alive and invisible here), and one nothing reaches at all. Only the last is
// a deletion, and only the first is the "mostly unexport" the task describes.
type Use string

const (
	// UseOutside is reached by another package's production code.
	UseOutside Use = "outside"
	// UseTest is reached only from a test, in this package or another. The
	// symbol is alive, but it exists for tests, which is worth knowing.
	UseTest Use = "test"
	// UseSignature appears in an exported function's own signature: a caller
	// receives the value and never names the type, so no selector can see it.
	//
	// This is not over-exporting, it is the API. Missing it is what made every
	// previous count wrong in the same direction — internal/app returns forty
	// named result types that nobody has reason to spell.
	UseSignature Use = "signature"
	// UseSet is a member of a const group whose siblings are named.
	//
	// A group declared together is one vocabulary. govbind's seven proposal
	// states mirror what a contract's getter returns; dropping the ones no
	// caller has needed yet would leave a partial mapping and send the next
	// reader back to the contract to work out the rest.
	UseSet Use = "set"
	// UseInside is reached only within its own package: exported for no
	// caller. This is the unexport list.
	UseInside Use = "inside"
	// UseNowhere is reached by nothing the source names.
	//
	// It is still not a delete list on its own. A registry dispatches by
	// value, an interface dispatches by method, and a plugin is wired in an
	// init — none of which names the symbol where this can see it. It is where
	// to look, not what to remove.
	UseNowhere Use = "nowhere"
)

// Reachable is one exported symbol and who reaches it.
type Reachable struct {
	Pkg  string `json:"pkg"`
	File string `json:"file"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Use  Use    `json:"use"`
}

// Reach classifies every exported non-method declaration under root by who
// names it. One implementation so the tool that reports it and the test that
// guards it count the same thing — the lesson of the surface graph, where a
// script and a ratchet drifted into measuring two different numbers.
func Reach(root string) ([]Reachable, error) {
	crossProd := map[string]map[string]bool{} // pkg -> name, from another package's production code
	crossTest := map[string]map[string]bool{} // pkg -> name, from any test file
	selfUse := map[string]map[string]int{}    // pkg -> how many times a bare identifier appears here
	inSig := map[string]map[string]bool{}     // pkg -> type named in an exported signature here
	group := map[string]string{}              // pkg.name -> the const group it belongs to
	groupOf := map[string][]string{}          // group -> its members, as pkg.name
	var decls []Reachable

	add := func(m map[string]map[string]bool, pkg, name string) {
		if m[pkg] == nil {
			m[pkg] = map[string]bool{}
		}
		m[pkg][name] = true
	}
	// Counted, not merely seen. A declaration names itself once, so presence
	// alone marks every symbol as used by its own package and nothing can ever
	// be unreachable. The first version did exactly that, and a mutation —
	// adding an exported function nobody calls — passed, which is how it was
	// found.
	bump := func(m map[string]map[string]int, pkg, name string) {
		if m[pkg] == nil {
			m[pkg] = map[string]int{}
		}
		m[pkg][name]++
	}

	// Methods are absent by design. A method call's receiver is a value rather
	// than a package name, so no selector in this walk can see one; counting
	// them put 424 unmeasurable entries into the 928 and buried the rest.
	for _, tree := range []string{"cmd", "internal", "scripts", "tests"} {
		dir := filepath.Join(root, tree)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			fset := token.NewFileSet()
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return nil
			}
			pkgPath := filepath.ToSlash(strings.TrimPrefix(filepath.Dir(path), root+string(filepath.Separator)))
			isTest := strings.HasSuffix(path, "_test.go")

			// Cross-package: pkg.Name.
			aliases := map[string]string{}
			for _, imp := range f.Imports {
				ip, uerr := strconv.Unquote(imp.Path.Value)
				if uerr != nil || !strings.HasPrefix(ip, modulePath+"/") {
					continue
				}
				rel := strings.TrimPrefix(ip, modulePath+"/")
				name := rel[strings.LastIndex(rel, "/")+1:]
				if imp.Name != nil {
					name = imp.Name.Name
				}
				aliases[name] = rel
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && id.Obj == nil {
						if to, ok := aliases[id.Name]; ok {
							if isTest {
								add(crossTest, to, sel.Sel.Name)
							} else {
								add(crossProd, to, sel.Sel.Name)
							}
						}
					}
					return true
				}
				// Same package: a bare identifier. Deliberately generous — a
				// local variable of the same name marks the symbol used. Over-
				// counting keeps a live symbol off the list, and the cost of
				// the opposite mistake is deleting something that works.
				if id, ok := n.(*ast.Ident); ok {
					bump(selfUse, pkgPath, id.Name)
				}
				return true
			})

			if !isTest {
				for _, d := range f.Decls {
					fn, ok := d.(*ast.FuncDecl)
					if !ok || !fn.Name.IsExported() {
						continue
					}
					ast.Inspect(fn.Type, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok {
							add(inSig, pkgPath, id.Name)
						}
						return true
					})
				}
				for _, s := range fileDecls(path, pkgPath) {
					decls = append(decls, Reachable{Pkg: s.Pkg, File: s.File, Kind: s.Kind, Name: s.Name})
				}
				for i, d := range f.Decls {
					gd, ok := d.(*ast.GenDecl)
					if !ok || gd.Tok != token.CONST || len(gd.Specs) < 2 {
						continue
					}
					id := fmt.Sprintf("%s:%s:%d", pkgPath, filepath.Base(path), i)
					for _, sp := range gd.Specs {
						vs, ok := sp.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, n := range vs.Names {
							if !n.IsExported() {
								continue
							}
							key := pkgPath + "." + n.Name
							group[key] = id
							groupOf[id] = append(groupOf[id], key)
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// How many times each name is DECLARED here, so those occurrences can be
	// subtracted from the count of times it appears.
	declared := map[string]int{}
	for _, d := range decls {
		declared[d.Pkg+"."+d.Name]++
	}
	// A group is live when any member of it is named.
	liveGroup := map[string]bool{}
	for _, d := range decls {
		key := d.Pkg + "." + d.Name
		if g := group[key]; g != "" {
			if crossProd[d.Pkg][d.Name] || crossTest[d.Pkg][d.Name] ||
				selfUse[d.Pkg][d.Name] > declared[key] {
				liveGroup[g] = true
			}
		}
	}
	for i := range decls {
		d := &decls[i]
		switch {
		case crossProd[d.Pkg][d.Name]:
			d.Use = UseOutside
		case crossTest[d.Pkg][d.Name]:
			d.Use = UseTest
		case d.Kind == "type" && inSig[d.Pkg][d.Name]:
			d.Use = UseSignature
		case selfUse[d.Pkg][d.Name] > declared[d.Pkg+"."+d.Name]:
			d.Use = UseInside
		case liveGroup[group[d.Pkg+"."+d.Name]]:
			d.Use = UseSet
		default:
			d.Use = UseNowhere
		}
	}
	sort.Slice(decls, func(i, j int) bool {
		if decls[i].Pkg != decls[j].Pkg {
			return decls[i].Pkg < decls[j].Pkg
		}
		return decls[i].Name < decls[j].Name
	})
	return decls, nil
}
