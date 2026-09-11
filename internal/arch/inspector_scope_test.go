package arch_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// The worklist carried "rewire health as a composition over inspector" as open
// work for months. Deciding it needed only reading the two packages: they do not
// share a substrate to compose on.
//
// inspector answers three questions about a target's ENVIRONMENT, before
// anything runs -- is this port taken, is this path missing, is this host
// reachable -- over TCP and a file store. health answers whether a CHAIN is
// producing, over RPC, and judges it. Putting health on top of inspector would
// mean giving inspector a fourth question, "sample these nodes over RPC", that
// nothing else asks and that its own doc excludes.
//
// So the decision is recorded here as a constraint rather than as prose that
// gets rediscovered: inspector stays on the environment side of the line. A file
// in it that reaches for RPC is the first step of the rewiring, and it fails
// here with the reason.
//
// This is not a claim that nothing overlaps. Three places sample chain liveness
// -- health (the verdict for `verify`), collector (the dashboard's stream), and
// the engine's launch gate -- and health's own doc argues the gate must stay
// separate because one reports on every node while the other blocks a bring-up
// on one. Whether health and collector should share a sampler is a real open
// question; inspector is not where it gets answered.
func TestInspectorStaysOnTheEnvironmentSide(t *testing.T) {
	const modPrefix = "github.com/0xmhha/chainbench/"
	// Packages that make inspector a chain-fact reader rather than an
	// environment one. rpc is the whole of it today; the list is here so a
	// second one is a deliberate edit.
	forbidden := map[string]string{
		"internal/core/rpc": "inspector reports what is on a target, not what a chain says; " +
			"an RPC read belongs to health (a verdict) or collector (a stream)",
	}

	files, err := filepath.Glob("../core/inspector/*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no inspector files found — this ratchet is pointing at nothing")
	}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, imp := range af.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(path, modPrefix) {
				continue
			}
			rel := strings.TrimPrefix(path, modPrefix)
			if why, bad := forbidden[rel]; bad {
				t.Errorf("%s imports %s: %s", filepath.Base(f), rel, why)
			}
		}
	}
}
