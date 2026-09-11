package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// AttachRun is the entry point for "run these specs against a network that is
// already up". It had no coverage, and it is where WA10 was fixed: an attach
// that names a WORKSPACE must go through the engine's workspace path, which
// wires the readiness gate, failure-evidence collection, fault control and the
// composition manifest, while the bare-URL path wires none of those because it
// owns no workspace and no processes.
//
// That distinction is a routing decision with no visible output of its own. If
// somebody folds the two branches together, nothing fails -- the evidence simply
// stops being collected, which is the failure WA10 described. So the routing is
// pinned by where the failure comes FROM.

func specJSON(t *testing.T, id string, steps ...map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "case", "id": id,
		"requires": []string{"rpc"},
		// A v2 case needs an env to parse at all, and a spec that does not parse
		// is dropped from the precheck by design — so a test about the precheck
		// has to hand it something parseable.
		"env": map[string]any{
			"schemaVersion": "2", "kind": "env", "id": "e", "chain": "wbft",
			"binaries": map[string]any{"default": "gwbft"},
			"topology": map[string]any{"bp": 1},
		},
		"steps": steps,
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestAttachRun_AWorkspaceGoesThroughTheWiredPath: with a DataDir and nothing
// else, the run must not complain that a chain is missing -- that refusal
// belongs to the bare-URL branch, so seeing it would mean the workspace went
// down the unwired road.
func TestAttachRun_AWorkspaceGoesThroughTheWiredPath(t *testing.T) {
	_, err := AttachRun(context.Background(), Deps{}, AttachRunIn{
		DataDir: t.TempDir(), // a real directory, but no composition in it
		Specs:   [][]byte{specJSON(t, "x", map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"})},
	})
	if err == nil {
		t.Fatal("an empty workspace should not produce a successful run")
	}
	// The workspace path prefixes its failures; the bare-URL branch's refusals
	// come straight from AttachRun. Which one spoke is the routing.
	if !strings.Contains(err.Error(), "attach workspace") {
		t.Fatalf("a workspace attach took the bare-URL branch (%q), which wires no readiness gate or failure evidence", err)
	}
}

// TestAttachRun_BareURLsNeedTheirInputs covers the two refusals of the branch
// that has no workspace to read them from.
func TestAttachRun_BareURLsNeedTheirInputs(t *testing.T) {
	spec := [][]byte{specJSON(t, "x", map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"})}

	_, err := AttachRun(context.Background(), Deps{}, AttachRunIn{Specs: spec, RPCURLs: []string{"http://x"}})
	if err == nil || !strings.Contains(err.Error(), "a chain is required") {
		t.Errorf("attaching with no chain should say so: %v", err)
	}

	_, err = AttachRun(context.Background(), Deps{}, AttachRunIn{Specs: spec, Chain: "wbft"})
	if err == nil || !strings.Contains(err.Error(), "no endpoint to attach to") {
		t.Errorf("attaching with no endpoint should say so: %v", err)
	}
}

// TestAttachRun_PrecheckRefusesBeforeAnythingRuns is the other half of WA10. A
// spec naming an assertion that does not exist used to reach the interpreter,
// where the mistake vanished into a result with no field to carry it. It fails
// here instead, naming the spec and the reference.
func TestAttachRun_PrecheckRefusesBeforeAnythingRuns(t *testing.T) {
	_, err := AttachRun(context.Background(), Deps{}, AttachRunIn{
		Chain: "wbft", RPCURLs: []string{"http://127.0.0.1:1"},
		Specs: [][]byte{specJSON(t, "typo-case", map[string]any{
			"expect": "blockNumbr", "compare": "GreaterOrEqual", "is": "1",
		})},
	})
	if err == nil {
		t.Fatal("a spec naming an assertion that does not exist was accepted")
	}
	if !strings.Contains(err.Error(), "typo-case") || !strings.Contains(err.Error(), "blockNumbr") {
		t.Errorf("the refusal should name the spec and the reference: %v", err)
	}
}

// TestAttachRun_AnUnparseableSpecIsLeftToTheEngine pins the deliberate gap in
// the precheck: a blob that is not a spec at all is not refused here, because
// the engine records a parse failure against that spec rather than failing the
// whole run. A precheck that refused it would turn one bad file into no run.
func TestAttachRun_AnUnparseableSpecIsLeftToTheEngine(t *testing.T) {
	_, err := AttachRun(context.Background(), Deps{}, AttachRunIn{
		Specs: [][]byte{[]byte("{not json")},
	})
	if err == nil {
		t.Fatal("the run should still fail for want of a chain")
	}
	if strings.Contains(err.Error(), "unresolved") || strings.Contains(err.Error(), "malformed") {
		t.Errorf("an unparseable spec was refused by the precheck rather than left to the engine: %v", err)
	}
}
