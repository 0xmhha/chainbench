package feature_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/feature"
)

// unregistered is how many of app's use cases the registry does not yet hold.
//
// S0 builds the skeleton; S1, S2 and S4 move the features onto it. Until then
// the number is large and that is the plan, not a defect. What the ratchet
// stops is the number going UP: a use case added straight to app while the
// migration is in flight is one more thing to move later, and one more chance
// for the three surfaces to describe it three ways.
//
// All 90 app entry points are bound (2026-10-13). AST reachability prevents
// duplicate descriptors for one function from inflating that coverage.
// Lower it as more register; never raise it.
const unregistered = 0

// TestRegistry_CoversMoreOfAppEachTime counts the use cases in internal/app and
// holds the unregistered remainder to a ceiling that only comes down.
//
// A use case is an exported function shaped
// func(context.Context, Deps, In) (Out, error) or func(Deps, ...) — the shape
// every surface calls. Counting the shape rather than a hand-kept list means a
// use case added tomorrow is counted tomorrow.
func TestRegistry_CoversMoreOfAppEachTime(t *testing.T) {
	cases := appUseCases(t)
	if len(cases) == 0 {
		t.Fatal("no use case was found in internal/app, so the walk is wrong rather than app empty")
	}
	registered := registeredAppUseCases(t, cases)
	var missing []string
	for _, name := range cases {
		if !registered[name] {
			missing = append(missing, name)
		}
	}
	got := len(missing)
	switch {
	case got > unregistered:
		t.Errorf("%d of app's %d use cases are unregistered, up from %d — a new use case should be registered rather than added beside the registry",
			got, len(cases), unregistered)
		t.Logf("missing bindings: %v", missing)
	case got < unregistered:
		t.Errorf("%d unregistered, down from %d — lower the constant so it keeps tracking reality", got, unregistered)
	}
	t.Logf("app exposes %d use cases; %d are registered, %d are not", len(cases), len(registered), got)
}

// Count the app functions each real registration invokes, not descriptor names.
// Two names for one function cannot conceal an unregistered function elsewhere.
func registeredAppUseCases(t *testing.T, cases []string) map[string]bool {
	t.Helper()
	known := map[string]bool{}
	for _, name := range cases {
		known[name] = true
	}
	registered := map[string]bool{}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		appAlias := ""
		for _, spec := range file.Imports {
			if spec.Path.Value == `"github.com/0xmhha/chainbench/internal/app"` {
				appAlias = "app"
				if spec.Name != nil {
					appAlias = spec.Name.Name
				}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			fn, ok := call.Fun.(*ast.Ident)
			if !ok || fn.Name != "Register" {
				return true
			}
			descriptor, ok := call.Args[0].(*ast.CompositeLit)
			if !ok {
				t.Errorf("uninspectable descriptor in %s", path)
				return true
			}
			name := ""
			for _, entry := range descriptor.Elts {
				pair, ok := entry.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := pair.Key.(*ast.Ident)
				if !ok || key.Name != "Name" {
					continue
				}
				if literal, ok := pair.Value.(*ast.BasicLit); ok {
					name, _ = strconv.Unquote(literal.Value)
				}
			}
			if _, ok := feature.Lookup(name); !ok {
				t.Errorf("AST registration %s was not installed", name)
			}
			reached := map[string]bool{}
			mark := func(expr ast.Expr) {
				selector, ok := expr.(*ast.SelectorExpr)
				if !ok {
					return
				}
				alias, ok := selector.X.(*ast.Ident)
				if ok && alias.Name == appAlias && known[selector.Sel.Name] {
					reached[selector.Sel.Name] = true
				}
			}
			mark(call.Args[1])
			ast.Inspect(call.Args[1], func(node ast.Node) bool {
				if invoke, ok := node.(*ast.CallExpr); ok {
					mark(invoke.Fun)
				}
				return true
			})
			if len(reached) != 1 {
				t.Errorf("registration %s must reach exactly one app usecase, got %v", name, reached)
			}
			for usecase := range reached {
				registered[usecase] = true
			}
			return true
		})
	}
	return registered
}

// appUseCases returns the exported functions in internal/app that a surface
// calls, by name.
func appUseCases(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("../app/*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if takesDeps(fn) {
				out = append(out, fn.Name.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// takesDeps reports whether fn has a Deps parameter, which is what marks a use
// case apart from a helper: Deps is the boundary a surface fills.
func takesDeps(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, p := range fn.Type.Params.List {
		if id, ok := p.Type.(*ast.Ident); ok && id.Name == "Deps" {
			return true
		}
	}
	return false
}
