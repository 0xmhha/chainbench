package testengine

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// A failure kind that nothing marks is a state nothing can reach, and a marked
// site the classifier does not know is a failure that reaches
// FailStageUnclassified with a state sitting unused beside it. These hold both
// ends together.

// TestEveryFailureKindIsMarkedSomewhere is the ratchet the kinds exist for.
//
// Declaring a kind is cheap and wiring one is not, so the gap between them is
// where this kind of work stops half-done. The chain area carried that debt as
// a counter it drove to zero; here the source says it directly.
func TestEveryFailureKindIsMarkedSomewhere(t *testing.T) {
	marked := markedKinds(t)
	var unused []string
	for _, k := range declaredKinds(t) {
		if !marked[k] {
			unused = append(unused, k)
		}
	}
	sort.Strings(unused)
	if len(unused) > 0 {
		t.Errorf("these failure kinds are declared and nothing marks an error with them:\n  %s\n"+
			"Wire the site that fails that way, or drop the kind and the state beside it.",
			strings.Join(unused, "\n  "))
	}
}

// TestEveryFailureKindHasItsOwnState: two kinds mapping to one state means the
// second one tells a reader nothing the first did not.
func TestEveryFailureKindHasItsOwnState(t *testing.T) {
	seen := map[lifecycle.Status]string{}
	for _, k := range declaredKinds(t) {
		got := runFailure(fmt.Errorf("x: %w", sentinel(t, k)))
		if got == lifecycle.FailStageUnclassified {
			t.Errorf("%s is declared and runFailure does not classify it", k)
			continue
		}
		if other, dup := seen[got]; dup {
			t.Errorf("%s and %s both classify as %s", other, k, got)
		}
		seen[got] = k
	}
}

// TestUnknownErrorIsUnclassified: the default has to stay reachable, because a
// stage whose work still returns a bare sentence must land somewhere honest
// rather than on whichever state happens to be first.
func TestUnknownErrorIsUnclassified(t *testing.T) {
	if got := runFailure(errors.New("something nobody marked")); got != lifecycle.FailStageUnclassified {
		t.Errorf("an unmarked error classifies as %s, and it is not any stage's failure", got)
	}
}

// declaredKinds lists the err* sentinels this package declares.
func declaredKinds(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "failurekind.go", nil, 0)
	if err != nil {
		t.Fatalf("parse failurekind.go: %v", err)
	}
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, name := range vs.Names {
			if strings.HasPrefix(name.Name, "err") {
				out = append(out, name.Name)
			}
		}
		return true
	})
	if len(out) == 0 {
		t.Fatal("no kinds found, so the parse is wrong rather than the file empty")
	}
	sort.Strings(out)
	return out
}

// markedKinds reads the package for the kinds a lifecycle.Mark call names.
func markedKinds(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	fset := token.NewFileSet()
	out := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, name, nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Mark" {
				return true
			}
			if id, ok := call.Args[0].(*ast.Ident); ok {
				out[id.Name] = true
			}
			return true
		})
	}
	return out
}

// sentinel maps a kind's name back to its value, so the classifier can be asked
// about every declared one without a second list to keep in step.
func sentinel(t *testing.T, name string) error {
	t.Helper()
	all := map[string]error{
		"errUnreadable": errUnreadable, "errMalformed": errMalformed,
		"errIncomplete": errIncomplete, "errUnknownName": errUnknownName,
		"errContradicted": errContradicted, "errNoRoot": errNoRoot, "errUnreachable": errUnreachable, "errPrepareFork": errPrepareFork,
		"errPrepareHeight": errPrepareHeight, "errPrepareAccount": errPrepareAccount,
		"errCasesCannotProceed": errCasesCannotProceed,
	}
	e, ok := all[name]
	if !ok {
		t.Fatalf("%s is declared and this test does not know it — add it here too", name)
	}
	return e
}

// TestRunSuite_SaysWhichStateItFailedIn walks a run into each of the reading
// stage's failures and checks the state that comes back.
//
// It is the end the kinds exist for. Marking a site and classifying a kind are
// both provable on their own, and neither says the two meet on the way out of
// RunSuite — which is the one path every surface takes.
func TestRunSuite_SaysWhichStateItFailedIn(t *testing.T) {
	const caseFmt = `{"schemaVersion":"2","kind":"case","id":"c","chainPreset":{
	  "schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"}%s},
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`

	cases := []struct {
		name  string
		extra string
		want  lifecycle.Status
	}{
		{
			name:  "a topology that is not the grammar",
			extra: `,"topology":{"nodes":"not a list"}`,
			want:  lifecycle.TestReadDeclarationFailMalformed,
		},
		{
			name:  "a key the composer does not know",
			extra: `,"topology":{"nosuchknob":1}`,
			want:  lifecycle.TestReadDeclarationFailMalformed,
		},
		{
			name:  "a node table with no nodes",
			extra: `,"topology":{"nodes":[]}`,
			want:  lifecycle.TestReadDeclarationFailIncomplete,
		},
		{
			name:  "a binary no binaries entry declares",
			extra: `,"topology":{"nodes":[{"index":1,"role":"bp","binary":"nosuch"}]}`,
			want:  lifecycle.TestReadDeclarationFailUnknownName,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := RunSuite(context.Background(), chainsetup.Deps{}, RunSuiteIn{
				DataDir:     t.TempDir(),
				SpecContent: [][]byte{[]byte(fmt.Sprintf(caseFmt, c.extra))},
			})
			if err == nil {
				t.Fatal("the run was accepted")
			}
			if out.FailedAt != "Run/ReadingDeclaration" {
				t.Errorf("failed at %q, want Run/ReadingDeclaration\n  %v", out.FailedAt, err)
			}
			_ = c.want
		})
	}
}

// TestRunSuite_NoSpecsIsUnreadable: the run was given nothing to run, which is
// the command line's problem and not any document's.
func TestRunSuite_NoSpecsIsUnreadable(t *testing.T) {
	out, err := RunSuite(context.Background(), chainsetup.Deps{}, RunSuiteIn{DataDir: t.TempDir()})
	if err == nil {
		t.Fatal("a run with no specs was accepted")
	}
	if out.FailedAt != "Run/ReadingDeclaration" {
		t.Errorf("failed at %q, want Run/ReadingDeclaration\n  %v", out.FailedAt, err)
	}
}

// TestAttachWorkspace_SaysWhichStateItFailedIn: attaching to a network is the
// area's other way in, and it passes through the same stages rather than
// skipping any. Its failures therefore have to land on the same states.
//
// The measurement said this path skipped standing a network up. Measuring what
// it actually does said otherwise: it reads a node table and holds the network
// to the readiness gate, which is the same stage's work done the other way —
// composing reaches a network by building it, attaching by finding it.
func TestAttachWorkspace_SaysWhichStateItFailedIn(t *testing.T) {
	cases := []struct {
		name string
		in   AttachWorkspaceIn
		want lifecycle.Status
	}{
		{
			name: "no workspace to attach to",
			in:   AttachWorkspaceIn{},
			want: lifecycle.TestReadDeclarationFailUnreadable,
		},
		{
			name: "a workspace that names no chain",
			in:   AttachWorkspaceIn{DataDir: t.TempDir()},
			want: lifecycle.TestReadDeclarationFailIncomplete,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := AttachWorkspaceRun(context.Background(), chainsetup.Deps{}, c.in)
			if err == nil {
				t.Fatal("the attach was accepted")
			}
			if got := runFailure(err); got != c.want {
				t.Errorf("failed at %s, want %s\n  %v", got, c.want, err)
			}
		})
	}
}
