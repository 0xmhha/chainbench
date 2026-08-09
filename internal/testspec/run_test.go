package testspec_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// --- fakes ---

type fakeAction struct {
	ran *bool
	err error
}

func (a fakeAction) Do(_ context.Context, _ *testspec.ActionCtx) error {
	if a.ran != nil {
		*a.ran = true
	}
	return a.err
}

type fakeAssertion struct{ pass bool }

func (a fakeAssertion) Check(_ context.Context, _ *testspec.AssertCtx) (session.AssertResult, error) {
	return session.AssertResult{Pass: a.pass}, nil
}

// fakeRecord captures TestRecord calls in memory.
type fakeRecord struct {
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
func (r *fakeRecord) PostAction(p session.PostResult) { r.posts = append(r.posts, p) }

func testEnv(t *testing.T) session.Environment {
	t.Helper()
	s, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC(), nil)
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
	reg := testspec.NewRegistry(false)
	stepRan, postRan := false, false
	reg.RegisterAction("tx", fakeAction{ran: &stepRan})
	reg.RegisterAction("cleanup", fakeAction{ran: &postRan})
	reg.RegisterAssertion("Len", fakeAssertion{pass: true})

	spec := testspec.Spec{
		Steps:       []map[string]any{{"tx": map[string]any{"on": "bp1"}}},
		Assertions:  []map[string]any{{"on": "bp1", "assert": "Len", "expected": 1}},
		PostActions: []map[string]any{{"cleanup": true}},
	}
	rec := &fakeRecord{}
	it := testspec.NewInterpreter(testspec.Deps{Actions: reg})

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
	reg := testspec.NewRegistry(false)
	stepRan := false
	reg.RegisterAction("ensureChain", fakeAction{err: errors.New("no chain")})
	reg.RegisterAction("tx", fakeAction{ran: &stepRan})

	spec := testspec.Spec{
		PreActions: []map[string]any{{"ensureChain": true}},
		Steps:      []map[string]any{{"tx": true}},
	}
	rec := &fakeRecord{}
	it := testspec.NewInterpreter(testspec.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusBlocked {
		t.Fatalf("status = %v, want blocked", status)
	}
	if stepRan {
		t.Fatal("steps must not run after pre-action failure")
	}
}

func TestRun_AssertFail(t *testing.T) {
	reg := testspec.NewRegistry(false)
	reg.RegisterAssertion("Len", fakeAssertion{pass: false})
	spec := testspec.Spec{Assertions: []map[string]any{{"assert": "Len", "expected": 7}}}
	rec := &fakeRecord{}
	it := testspec.NewInterpreter(testspec.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail", status)
	}
}

func TestRun_UnknownActionFails(t *testing.T) {
	reg := testspec.NewRegistry(false)
	spec := testspec.Spec{Steps: []map[string]any{{"nope": true}}}
	rec := &fakeRecord{}
	it := testspec.NewInterpreter(testspec.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("unknown action: status = %v, want fail", status)
	}
}

// provenanceAction surfaces a tx hash and receipt for step-provenance recording.
type provenanceAction struct{}

func (provenanceAction) Do(_ context.Context, ac *testspec.ActionCtx) error {
	ac.Hash = "0xdeadbeef"
	ac.Receipt = map[string]any{"status": "0x1"}
	return nil
}

func TestRun_StepRecordsProvenance(t *testing.T) {
	reg := testspec.NewRegistry(false)
	reg.RegisterAction("tx", provenanceAction{})

	spec := testspec.Spec{Steps: []map[string]any{{"tx": map[string]any{"on": "bp1"}}}}
	rec := &fakeRecord{}
	it := testspec.NewInterpreter(testspec.Deps{Actions: reg})

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
	reg := testspec.NewRegistry(false)
	spec := testspec.Spec{Steps: []map[string]any{{"nope": true}}}
	rec := &fakeRecord{}
	it := testspec.NewInterpreter(testspec.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail for unknown step action", status)
	}
	if len(rec.stepResults) != 1 {
		t.Fatalf("unknown step must still be recorded, got %d", len(rec.stepResults))
	}
}
