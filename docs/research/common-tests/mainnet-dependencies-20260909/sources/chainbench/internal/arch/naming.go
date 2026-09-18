package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Decl is one exported package-level declaration.
//
// Methods are deliberately absent. A method name is namespaced by its receiver,
// so `collector.Store.Stop` and `process.Node.Stop` are not ambiguous at any
// call site. Counting them put 577 declarations into the tally and buried the
// names that actually collide.
type Decl struct {
	Pkg  string // import path below the module root, e.g. "internal/app"
	Name string
	Kind string // const, var, type, func
	File string

	// Refs names the module-internal packages this declaration's own source
	// references. It is what tells a re-export from a coincidence: app.Node is
	// `= node.Node` and refers to core/node, while app.Network refers to
	// nothing in core/keyring despite sharing that package's spelling.
	Refs []string
}

// Collision is one exported name declared at package level in more than one
// package.
type Collision struct {
	Name  string
	Pkgs  []string // sorted
	Kinds []string // sorted
	decls []Decl
}

// Decls walks root and returns every exported package-level declaration under
// cmd/, internal/ and scripts/. Test files are skipped: a helper that shares a
// name with a production type is not a vocabulary problem.
func Decls(root string) []Decl {
	var out []Decl
	for _, top := range []string{"cmd", "internal", "scripts"} {
		base := filepath.Join(root, top)
		_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			pkg := filepath.ToSlash(strings.TrimPrefix(filepath.Dir(p), root+string(filepath.Separator)))
			out = append(out, fileDecls(p, pkg)...)
			return nil
		})
	}
	return out
}

// fileDecls parses one file and returns its exported package-level
// declarations.
func fileDecls(path, pkg string) []Decl {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil
	}
	imports := internalImports(f)
	name := filepath.Base(path)

	var out []Decl
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			// A method belongs to its receiver's namespace, not the package's.
			if d.Recv != nil || !d.Name.IsExported() {
				continue
			}
			out = append(out, Decl{pkg, d.Name.Name, "func", name, refsIn(d, imports)})
		case *ast.GenDecl:
			kind := map[token.Token]string{token.CONST: "const", token.VAR: "var", token.TYPE: "type"}[d.Tok]
			if kind == "" {
				continue // import
			}
			for _, s := range d.Specs {
				for _, id := range specNames(s) {
					if !id.IsExported() {
						continue
					}
					out = append(out, Decl{pkg, id.Name, kind, name, refsIn(s, imports)})
				}
			}
		}
	}
	return out
}

// specNames returns the identifiers a value or type spec declares.
func specNames(s ast.Spec) []*ast.Ident {
	switch s := s.(type) {
	case *ast.ValueSpec:
		return s.Names
	case *ast.TypeSpec:
		return []*ast.Ident{s.Name}
	}
	return nil
}

// internalImports maps a file's import names to the module-internal package
// paths they stand for. Third-party imports are left out; they never explain a
// collision inside this module.
func internalImports(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		rel, ok := strings.CutPrefix(p, modulePath+"/")
		if !ok {
			continue
		}
		local := importName(rel)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		out[local] = rel
	}
	return out
}

// importName returns the identifier an import is referred to by when it carries
// no explicit name.
func importName(rel string) string {
	i := strings.LastIndex(rel, "/")
	return rel[i+1:]
}

// refsIn returns the module-internal packages a declaration's source mentions.
func refsIn(n ast.Node, imports map[string]string) []string {
	seen := map[string]bool{}
	ast.Inspect(n, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			if p, ok := imports[id.Name]; ok {
				seen[p] = true
			}
		}
		return true
	})
	return keys(seen)
}

// Collisions groups Decls by name and returns the names declared in more than
// one package, sorted by name.
func Collisions(root string) []Collision {
	byName := map[string][]Decl{}
	for _, d := range Decls(root) {
		byName[d.Name] = append(byName[d.Name], d)
	}
	var out []Collision
	for name, ds := range byName {
		pkgs, kinds := map[string]bool{}, map[string]bool{}
		for _, d := range ds {
			pkgs[d.Pkg] = true
			kinds[d.Kind] = true
		}
		if len(pkgs) < 2 {
			continue
		}
		c := Collision{Name: name, Pkgs: keys(pkgs), Kinds: keys(kinds), decls: ds}
		sort.Strings(c.Pkgs)
		sort.Strings(c.Kinds)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Explained says why a shared name is not a naming defect, or returns "" when
// nothing accounts for it.
//
// Each rule below is a case where two packages MUST spell something the same
// way, so flagging it would teach the reader to ignore the test. Anything else
// is one concept wearing two names or two concepts wearing one, which is what
// [[layers]] §5b forbids.
func (c Collision) Explained() string {
	switch {
	case c.Name == "New":
		return "Go 의 생성자 관용이다. 호출할 때 늘 패키지 이름이 앞에 붙으므로 헷갈릴 자리가 없다"
	case c.within("internal/chains/", "internal/core/registry"):
		return "체인 플러그인 계약이다. 패밀리마다 같은 이름을 내놓아야 레지스트리가 하나의 경계로 디스패치한다"
	case c.within("internal/consensus/"):
		return "합의 패밀리 계약이다. 위와 같은 이유로 패밀리끼리 이름이 같아야 한다"
	case c.within("scripts/"):
		return "따로 도는 도구들이라 어휘를 공유하지 않는다"
	case c.within("cmd/"):
		return "명령별 배선이다. 각 명령이 자기 New 를 갖는 것과 같다"
	case c.appForwards():
		return "app 이 같은 개념을 같은 이름으로 다시 내보낸다. 한 개념에 한 이름이라는 규칙이 지켜진 모습이다"
	}
	return ""
}

// within reports whether every package holding the name sits under one of the
// given prefixes (an exact path counts too).
func (c Collision) within(prefixes ...string) bool {
	for _, p := range c.Pkgs {
		hit := false
		for _, pre := range prefixes {
			if p == pre || strings.HasPrefix(p, pre) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// appForwards reports whether the name is shared by app and exactly one other
// package, and app's own declaration actually reaches for that package.
//
// The reference check is the point. Without it the rule would also bless
// app.Network against core/keyring's Network, which are a lookup of an attached
// network and a preset's chain parameters: two concepts, one spelling, exactly
// what this test is for.
func (c Collision) appForwards() bool {
	if len(c.Pkgs) != 2 {
		return false
	}
	other := ""
	for _, p := range c.Pkgs {
		if p != "internal/app" {
			other = p
		}
	}
	if other == "" {
		return false
	}
	for _, d := range c.decls {
		if d.Pkg != "internal/app" {
			continue
		}
		for _, r := range d.Refs {
			if r == other {
				return true
			}
		}
	}
	return false
}
