package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// targetBranches lists every place a composition step asks whether its target
// is remote, and says why that question is a FACT about the target rather than
// a second code path.
//
// The distinction is the whole of N4's gate ("no local/remote branch"). A step
// that runs different logic for the two targets is two implementations of one
// job, and they drift: A3 found remote provisioning writing genesis and config
// to the operator's own machine while shipping only the identities, which is
// exactly that drift, and it had been shipping.
//
// Asking WHERE something is, on the other hand, is unavoidable and correct. A
// remote node's keys live under its data root; a local node's are the key set
// itself. Baking the operator-side path into a remote config is how a node came
// to look for its nodekey on a machine it cannot see, so keysBase has to ask.
//
// The measurement is that the restructuring is essentially done: of the entries
// below, one is a missing capability and the rest are facts. The list may only
// shrink. A new entry means a step learned to do two things, and it has to be
// justified here in front of a reviewer rather than in a diff nobody re-reads.
var targetBranches = map[string]string{
	"new.go:New":                         "a local target defaults its data root to the workspace; a remote one was given one",
	"steps_compose.go:Allocate":          "only a remote pool has host names distinct from addresses, so only then is there a name to record",
	"steps_compose.go:genesisArtifacts":  "a genesis its own binary writes runs where the binary is; the inputs are staged over the same access init and start use",
	"steps_compose.go:shipIdentities":    "a local target ships nothing because keysBase already IS the key set — the same operation, with no work to do",
	"steps_lifecycle.go:scanPorts":       "a port is probed by whoever can bind it; remotely that means running the probe on that machine, which inspector owns",
	"steps_lifecycle.go:runPhaseActions": "the bootstrap's keystore, socket and config are on the target, so the paths point there",
	"verbs_network.go:NetRunner":         "there is a command runner only when there is a machine to run commands on",
	"workspace.go:keysBase":              "where the keys are: under the target's data root when remote, the key set itself when local",
	"workspace.go:RPCHost":               "which host answers RPC",

	// The list is exhausted: every entry left is a fact about the target, not a
	// branch around a missing capability.
	//
	// "steps_lifecycle.go:Rm" was the one exception — removing a remote data
	// plane needed a delete on the filestore boundary and there was none, so the
	// step refused out loud rather than deleting the wrong thing. filestore.Store
	// gained Remove (verified live on the docker fleet, 2026-09-11) and Rm now
	// goes through the target's store like every other step, so it no longer asks
	// where the target is.
}

// TestStepsDoNotBranchOnTheTarget holds N4's gate: a composition step asks
// where its target is, and never runs a different job because of the answer.
func TestStepsDoNotBranchOnTheTarget(t *testing.T) {
	found := targetQuestions(t, "../chainsetup")

	seen := map[string]bool{}
	for _, where := range found {
		if _, ok := targetBranches[where]; !ok {
			t.Errorf("%s asks whether the target is remote and is not on the list — say why it is a fact about the target, or make the step do one job for both", where)
			continue
		}
		seen[where] = true
	}
	// Both directions. An entry whose branch is gone is a claim about the code
	// that has stopped being true, and a list that has drifted teaches the next
	// reader to skip it.
	for where := range targetBranches {
		if !seen[where] {
			t.Errorf("targetBranches[%q] matches no branch — the step no longer asks, so remove the entry", where)
		}
	}
	t.Logf("%d places ask where the target is; %d of them are facts about it", len(found), len(found)-1)
}

// targetQuestions returns "file.go:function" for every IsRemote call in dir.
//
// The function, not the line: a line number drifts with every edit above it,
// and a ratchet that has to be renumbered is a ratchet people start ignoring.
func targetQuestions(t *testing.T, dir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
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
		base := filepath.Base(path)
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			asks := false
			ast.Inspect(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "IsRemote" {
					asks = true
				}
				return true
			})
			if asks {
				out = append(out, base+":"+fn.Name.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}
