package feature_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
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
// Measured 2026-09-08, after S1 and S4 registered the compose and report stages.
// Lower it as more register; never raise it.
const unregistered = 63

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
	registered := map[string]bool{}
	for _, d := range feature.Registered() {
		// Probes registered by this package's own tests are not migration
		// progress, and counting them would make the ratchet depend on which
		// test ran first.
		if strings.HasPrefix(d.Name, "probe.") {
			continue
		}
		// The registry spells a feature "chain.genesis"; app spells it
		// NetGenesis. Only the count is compared here — the names converge as
		// features move, and pinning a mapping now would be a third spelling
		// of something being actively renamed.
		registered[d.Name] = true
	}
	got := len(cases) - len(registered)
	if got < 0 {
		got = 0
	}
	switch {
	case got > unregistered:
		t.Errorf("%d of app's %d use cases are unregistered, up from %d — a new use case should be registered rather than added beside the registry",
			got, len(cases), unregistered)
	case got < unregistered:
		t.Errorf("%d unregistered, down from %d — lower the constant so it keeps tracking reality", got, unregistered)
	}
	t.Logf("app exposes %d use cases; %d are registered, %d are not", len(cases), len(registered), got)
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
