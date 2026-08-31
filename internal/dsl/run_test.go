package dsl_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// --- fakes ---

type fakeAction struct {
	ran *bool
	err error
}

func (a fakeAction) Do(_ context.Context, _ *dsl.ActionCtx) error {
	if a.ran != nil {
		*a.ran = true
	}
	return a.err
}

type fakeAssertion struct{ pass bool }

func (a fakeAssertion) Check(_ context.Context, _ *dsl.AssertCtx) (session.AssertResult, error) {
	return session.AssertResult{Pass: a.pass}, nil
}

// fakeRecord captures TestRecord calls in memory.
type fakeRecord struct {
	reason      string
	steps       int
	stepResults []session.StepResult
	asserts     []session.AssertResult
	status      session.TestStatus
	posts       []session.PostResult
}

func (r *fakeRecord) Dir() string      { return "" }
func (r *fakeRecord) SetEnvRef(string) {}
func (r *fakeRecord) Spec([]byte)      {}
func (r *fakeRecord) Step(_ int, res session.StepResult) {
	r.steps++
	r.stepResults = append(r.stepResults, res)
}
func (r *fakeRecord) Assert(a session.AssertResult)   { r.asserts = append(r.asserts, a) }
func (r *fakeRecord) Status(s session.TestStatus)     { r.status = s }
func (r *fakeRecord) Reason(why string)               { r.reason = why }
func (r *fakeRecord) PostAction(p session.PostResult) { r.posts = append(r.posts, p) }

func testEnv(t *testing.T) session.Environment {
	t.Helper()
	s, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	env, err := s.NewEnvironment("aaaaaaaaaaaa0000")
	if err != nil {
		t.Fatal(err)
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"}}})
	return env
}

func TestRun_PassFlow(t *testing.T) {
	reg := dsl.NewRegistry()
	stepRan, postRan := false, false
	reg.RegisterAction("tx", fakeAction{ran: &stepRan})
	reg.RegisterAction("cleanup", fakeAction{ran: &postRan})
	reg.RegisterAssertion("Len", fakeAssertion{pass: true})

	spec := dsl.Spec{
		Steps:       []map[string]any{{"tx": map[string]any{"on": "bp1"}}},
		Assertions:  []map[string]any{{"on": "bp1", "assert": "Len", "expected": 1}},
		PostActions: []map[string]any{{"cleanup": true}},
	}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, err := it.Run(context.Background(), spec, testEnv(t), rec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if status != session.StatusPass {
		t.Fatalf("status = %v, want pass", status)
	}
	if !stepRan || !postRan {
		t.Fatalf("step ran=%v post ran=%v", stepRan, postRan)
	}
	if rec.steps != 1 || len(rec.asserts) != 1 || !rec.asserts[0].Pass || len(rec.posts) != 1 {
		t.Fatalf("record = %+v", rec)
	}
}

func TestRun_PreFailBlocked(t *testing.T) {
	reg := dsl.NewRegistry()
	stepRan := false
	reg.RegisterAction("ensureChain", fakeAction{err: errors.New("no chain")})
	reg.RegisterAction("tx", fakeAction{ran: &stepRan})

	spec := dsl.Spec{
		PreActions: []map[string]any{{"ensureChain": true}},
		Steps:      []map[string]any{{"tx": true}},
	}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusBlocked {
		t.Fatalf("status = %v, want blocked", status)
	}
	if stepRan {
		t.Fatal("steps must not run after pre-action failure")
	}
}

func TestRun_AssertFail(t *testing.T) {
	reg := dsl.NewRegistry()
	reg.RegisterAssertion("Len", fakeAssertion{pass: false})
	spec := dsl.Spec{Assertions: []map[string]any{{"assert": "Len", "expected": 7}}}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail", status)
	}
}

func TestRun_UnknownActionFails(t *testing.T) {
	reg := dsl.NewRegistry()
	spec := dsl.Spec{Steps: []map[string]any{{"nope": true}}}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("unknown action: status = %v, want fail", status)
	}
}

// provenanceAction surfaces a tx hash and receipt for step-provenance recording.
type provenanceAction struct{}

func (provenanceAction) Do(_ context.Context, ac *dsl.ActionCtx) error {
	ac.Hash = "0xdeadbeef"
	ac.Receipt = map[string]any{"status": "0x1"}
	return nil
}

func TestRun_StepRecordsProvenance(t *testing.T) {
	reg := dsl.NewRegistry()
	reg.RegisterAction("tx", provenanceAction{})

	spec := dsl.Spec{Steps: []map[string]any{{"tx": map[string]any{"on": "bp1"}}}}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	if _, err := it.Run(context.Background(), spec, testEnv(t), rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.stepResults) != 1 {
		t.Fatalf("want 1 step result, got %d", len(rec.stepResults))
	}
	sr := rec.stepResults[0]
	if sr.Hash != "0xdeadbeef" || sr.On != "bp1" || sr.Receipt["status"] != "0x1" {
		t.Fatalf("step provenance = %+v", sr)
	}
}

func TestRun_UnknownStepActionFails(t *testing.T) {
	reg := dsl.NewRegistry()
	spec := dsl.Spec{Steps: []map[string]any{{"nope": true}}}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail for unknown step action", status)
	}
	if len(rec.stepResults) != 1 {
		t.Fatalf("unknown step must still be recorded, got %d", len(rec.stepResults))
	}
}

// TestRun_InterleavedSequence pins the v2 unified-sequence semantics: an
// expect failure records and continues (later statements still run), a do
// failure stops the sequence, and onFail hooks run on a failed case.
func TestRun_InterleavedSequence(t *testing.T) {
	reg := dsl.NewRegistry()
	laterRan, onFailRan := false, false
	reg.RegisterAction("tx", fakeAction{})
	reg.RegisterAction("later", fakeAction{ran: &laterRan})
	reg.RegisterAction("diag", fakeAction{ran: &onFailRan})
	reg.RegisterAssertion("failing", fakeAssertion{pass: false})
	reg.RegisterAssertion("passing", fakeAssertion{pass: true})

	spec := dsl.Spec{
		Sequence: []dsl.Statement{
			{Do: "tx", Args: map[string]any{}},
			{Expect: "failing", Args: map[string]any{}}, // records, continues
			{Do: "later", Args: map[string]any{}},       // still runs
			{Expect: "passing", Args: map[string]any{}},
		},
		OnFailActions: []map[string]any{{"diag": true}},
	}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, err := it.Run(context.Background(), spec, testEnv(t), rec)
	if err != nil {
		t.Fatal(err)
	}
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail", status)
	}
	if !laterRan {
		t.Fatal("an expect failure must not stop later statements")
	}
	if !onFailRan {
		t.Fatal("onFail hooks must run on a failed case")
	}
	if len(rec.asserts) != 2 {
		t.Fatalf("asserts recorded = %d, want 2", len(rec.asserts))
	}
}

// TestRun_DoFailureStopsSequence pins fail-fast for do statements: later
// statements are skipped and post actions do not run (the v1 contract).
func TestRun_DoFailureStopsSequence(t *testing.T) {
	reg := dsl.NewRegistry()
	laterRan, postRan, onFailRan := false, false, false
	reg.RegisterAction("boom", fakeAction{err: errors.New("broken")})
	reg.RegisterAction("later", fakeAction{ran: &laterRan})
	reg.RegisterAction("cleanup", fakeAction{ran: &postRan})
	reg.RegisterAction("diag", fakeAction{ran: &onFailRan})

	spec := dsl.Spec{
		Sequence: []dsl.Statement{
			{Do: "boom", Args: map[string]any{}},
			{Do: "later", Args: map[string]any{}},
		},
		PostActions:   []map[string]any{{"cleanup": true}},
		OnFailActions: []map[string]any{{"diag": true}},
	}
	rec := &fakeRecord{}
	it := dsl.NewInterpreter(dsl.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail", status)
	}
	if laterRan {
		t.Fatal("a do failure must stop the sequence")
	}
	if postRan {
		t.Fatal("post actions must not run after a do failure (v1 contract)")
	}
	if !onFailRan {
		t.Fatal("onFail diagnostics must run after a do failure")
	}
}
