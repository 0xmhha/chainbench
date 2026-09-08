package interp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// --- fakes ---

type fakeAction struct {
	ran *bool
	err error
}

func (a fakeAction) Do(_ context.Context, _ *interp.ActionCtx) error {
	if a.ran != nil {
		*a.ran = true
	}
	return a.err
}

type fakeAssertion struct{ pass bool }

func (a fakeAssertion) Check(_ context.Context, _ *interp.AssertCtx) (session.AssertResult, error) {
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
func (r *fakeRecord) Assert(a session.AssertResult)     { r.asserts = append(r.asserts, a) }
func (r *fakeRecord) Status(s session.TestStatus)       { r.status = s }
func (r *fakeRecord) Reason(why string)                 { r.reason = why }
func (r *fakeRecord) PostAction(p session.PostResult)   { r.posts = append(r.posts, p) }
func (r *fakeRecord) Artifacts(_ session.TestArtifacts) {}
func (r *fakeRecord) Observation(_ string, _ []byte)    {}

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

// captureOnAction records the "on" selector each step is dispatched with, so a
// test can assert how a statement was routed.
type captureOnAction struct{ seen *[]string }

func (a captureOnAction) Do(_ context.Context, ac *interp.ActionCtx) error {
	on, _ := ac.Args["on"].(string)
	*a.seen = append(*a.seen, on)
	return nil
}

// captureDeadlineAction records whether the ctx it ran under carried a deadline.
type captureDeadlineAction struct{ hadDeadline *bool }

func (a captureDeadlineAction) Do(ctx context.Context, _ *interp.ActionCtx) error {
	_, ok := ctx.Deadline()
	*a.hadDeadline = ok
	return nil
}

// TestRun_CaseTimeoutBoundsTheRun pins WA15: a case-level timeout puts a
// deadline on the whole run's context, so a hanging step fails within the
// declared budget. Before the fix timeouts was parsed but never consulted.
func TestRun_CaseTimeoutBoundsTheRun(t *testing.T) {
	reg := interp.NewRegistry()
	var hadDeadline bool
	reg.RegisterAction("tx", captureDeadlineAction{hadDeadline: &hadDeadline})
	reg.RegisterAssertion("Len", fakeAssertion{pass: true})

	spec := dsl.Spec{
		Timeouts:   map[string]string{"case": "10m"},
		Steps:      []map[string]any{{"tx": map[string]any{}}},
		Assertions: []map[string]any{{"assert": "Len"}},
	}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})
	if _, err := it.Run(context.Background(), spec, testEnv(t), rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hadDeadline {
		t.Fatal("a case timeout must put a deadline on the run's context")
	}
}

// TestRun_DefaultOnRoutesStatements pins WA14: a case-level default target
// routes every statement that names none, while a statement with its own on
// still wins. Before the fix defaultOn was parsed but never consulted, so both
// steps silently went to the primary node.
func TestRun_DefaultOnRoutesStatements(t *testing.T) {
	reg := interp.NewRegistry()
	var seen []string
	reg.RegisterAction("tx", captureOnAction{seen: &seen})
	reg.RegisterAssertion("Len", fakeAssertion{pass: true})

	spec := dsl.Spec{
		DefaultOn: "bp3",
		Steps: []map[string]any{
			{"tx": map[string]any{}},            // no on -> the case default
			{"tx": map[string]any{"on": "bp1"}}, // its own on wins
		},
		Assertions: []map[string]any{{"assert": "Len"}},
	}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})
	if _, err := it.Run(context.Background(), spec, testEnv(t), rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"bp3", "bp1"}
	if len(seen) != 2 || seen[0] != want[0] || seen[1] != want[1] {
		t.Fatalf("routed on = %v, want %v (defaultOn routes the first, explicit on wins the second)", seen, want)
	}
}

func TestRun_PassFlow(t *testing.T) {
	reg := interp.NewRegistry()
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
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

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
	reg := interp.NewRegistry()
	stepRan := false
	reg.RegisterAction("ensureChain", fakeAction{err: errors.New("no chain")})
	reg.RegisterAction("tx", fakeAction{ran: &stepRan})

	spec := dsl.Spec{
		PreActions: []map[string]any{{"ensureChain": true}},
		Steps:      []map[string]any{{"tx": true}},
	}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusBlocked {
		t.Fatalf("status = %v, want blocked", status)
	}
	if stepRan {
		t.Fatal("steps must not run after pre-action failure")
	}
	// WA12: a reader of status.json alone must learn why it blocked.
	if rec.reason == "" {
		t.Fatal("a blocked pre-action must record a reason")
	}
}

func TestRun_AssertFail(t *testing.T) {
	reg := interp.NewRegistry()
	reg.RegisterAssertion("Len", fakeAssertion{pass: false})
	spec := dsl.Spec{Assertions: []map[string]any{{"assert": "Len", "expected": 7}}}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("status = %v, want fail", status)
	}
	// WA12: a failed case records why, not just that it failed.
	if rec.reason == "" {
		t.Fatal("a failed case must record a reason")
	}
}

func TestRun_UnknownActionFails(t *testing.T) {
	reg := interp.NewRegistry()
	spec := dsl.Spec{Steps: []map[string]any{{"nope": true}}}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

	status, _ := it.Run(context.Background(), spec, testEnv(t), rec)
	if status != session.StatusFail {
		t.Fatalf("unknown action: status = %v, want fail", status)
	}
}

// provenanceAction surfaces a tx hash and receipt for step-provenance recording.
type provenanceAction struct{}

func (provenanceAction) Do(_ context.Context, ac *interp.ActionCtx) error {
	ac.Hash = "0xdeadbeef"
	ac.Receipt = map[string]any{"status": "0x1"}
	return nil
}

func TestRun_StepRecordsProvenance(t *testing.T) {
	reg := interp.NewRegistry()
	reg.RegisterAction("tx", provenanceAction{})

	spec := dsl.Spec{Steps: []map[string]any{{"tx": map[string]any{"on": "bp1"}}}}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

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
	reg := interp.NewRegistry()
	spec := dsl.Spec{Steps: []map[string]any{{"nope": true}}}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

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
	reg := interp.NewRegistry()
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
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

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
	reg := interp.NewRegistry()
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
	it := interp.NewInterpreter(interp.Deps{Actions: reg})

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

// TestRun_OnEachStepFansOut pins WA18: a do step's onEach runs the action once
// per selected node, mirroring how an assertion checks each. Before the fix the
// action path read only "on", so onEach was silently ignored and the step ran
// once against the primary node.
func TestRun_OnEachStepFansOut(t *testing.T) {
	reg := interp.NewRegistry()
	var seen []string
	reg.RegisterAction("tx", captureOnAction{seen: &seen})
	reg.RegisterAssertion("Len", fakeAssertion{pass: true})

	spec := dsl.Spec{
		Steps:      []map[string]any{{"tx": map[string]any{"onEach": []any{"node1", "node2"}}}},
		Assertions: []map[string]any{{"assert": "Len"}},
	}
	rec := &fakeRecord{}
	it := interp.NewInterpreter(interp.Deps{Actions: reg})
	if _, err := it.Run(context.Background(), spec, testEnv(t), rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"node1", "node2"}
	if len(seen) != 2 || seen[0] != want[0] || seen[1] != want[1] {
		t.Fatalf("fan-out targets = %v, want %v (onEach runs the action per node)", seen, want)
	}
	if rec.steps != 2 {
		t.Fatalf("recorded %d steps, want 2 (one per fanned-out node)", rec.steps)
	}
}
