package chainsetup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestCases_AreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Cases() {
		if c.ID == "" || c.Title == "" || c.Doc == "" {
			t.Errorf("case %+v is missing an id, title, or doc", c)
		}
		if seen[c.ID] {
			t.Errorf("duplicate case id %q", c.ID)
		}
		seen[c.ID] = true
		if len(c.Steps) == 0 {
			t.Errorf("case %q has no steps", c.ID)
		}
		if len(c.Knobs) == 0 {
			t.Errorf("case %q lists no customization points", c.ID)
		}
		stepIDs := map[string]bool{}
		for _, s := range c.Steps {
			if s.ID == "" || s.Detail == "" {
				t.Errorf("case %q has a step missing an id or detail: %+v", c.ID, s)
			}
			if stepIDs[s.ID] {
				t.Errorf("case %q has duplicate step id %q", c.ID, s.ID)
			}
			stepIDs[s.ID] = true
		}
		// A case that claims support must have every step built; claiming
		// support for a procedure with a hole in it is the drift this model
		// exists to prevent.
		if c.Support == Supported {
			for _, s := range c.Steps {
				if !s.Implemented {
					t.Errorf("case %q claims %s but step %q is not implemented", c.ID, c.Support, s.ID)
				}
			}
		}
		// Anything short of Supported has to say why.
		if c.Support != Supported && c.Note == "" {
			t.Errorf("case %q is %s but gives no reason", c.ID, c.Support)
		}
	}
}

func TestFind(t *testing.T) {
	if _, ok := Find("stablenet"); !ok {
		t.Fatal("stablenet case not found")
	}
	if _, ok := Find("nope"); ok {
		t.Fatal("Find returned a case for an unknown id")
	}
}

func TestTracker_StopsAtTheFirstFailure(t *testing.T) {
	steps := []CaseStep{
		{ID: "a", Detail: "a", Implemented: true},
		{ID: "b", Detail: "b", Implemented: true},
		{ID: "c", Detail: "c", Implemented: true},
	}
	tr := newTracker(nil, "")
	tr.do(steps[0], func() (string, error) { return "ok", nil })
	tr.do(steps[1], func() (string, error) { return "", errors.New("boom") })
	tr.do(steps[2], func() (string, error) { t.Fatal("step c ran after a failure"); return "", nil })

	if got := []Outcome{tr.results[0].Outcome, tr.results[1].Outcome, tr.results[2].Outcome}; got[0] != OK || got[1] != Failed || got[2] != Skipped {
		t.Fatalf("outcomes = %v, want [OK FAIL SKIP]", got)
	}
}

func TestTracker_UnimplementedStepIsDistinctFromFailure(t *testing.T) {
	tr := newTracker(nil, "")
	tr.do(CaseStep{ID: "todo", Detail: "d"}, nil)
	if tr.results[0].Outcome != NotImplemented {
		t.Fatalf("outcome = %s, want %s", tr.results[0].Outcome, NotImplemented)
	}
	if !tr.halted() {
		t.Fatal("an unimplemented step should stop the run")
	}
}

func TestTracker_StopAfterEndsTheRun(t *testing.T) {
	tr := newTracker(nil, "a")
	tr.do(CaseStep{ID: "a", Detail: "a", Implemented: true}, func() (string, error) { return "", nil })
	tr.do(CaseStep{ID: "b", Detail: "b", Implemented: true}, func() (string, error) { t.Fatal("ran past --stop-after"); return "", nil })
	if tr.results[1].Outcome != Skipped {
		t.Fatalf("outcome = %s, want SKIP", tr.results[1].Outcome)
	}
}

func TestValidateStopAfter(t *testing.T) {
	c, _ := Find("stablenet")
	if err := validateStopAfter(c, "provision"); err != nil {
		t.Fatalf("provision is a real step: %v", err)
	}
	if err := validateStopAfter(c, "nonsense"); err == nil {
		t.Fatal("expected an error for a step the case does not have")
	}
	if err := validateStopAfter(c, ""); err != nil {
		t.Fatalf("an empty --stop-after is valid: %v", err)
	}
}

func TestRun_FirstProblem(t *testing.T) {
	r := Run{Results: []Result{
		{Step: CaseStep{ID: "a"}, Outcome: OK},
		{Step: CaseStep{ID: "b"}, Outcome: Failed, Detail: "why"},
		{Step: CaseStep{ID: "c"}, Outcome: Skipped},
	}}
	if !r.Failed() {
		t.Fatal("Failed() = false")
	}
	p, ok := r.FirstProblem()
	if !ok || p.Step.ID != "b" {
		t.Fatalf("FirstProblem = %+v", p)
	}
}

func fakeRunner(out string, err error) poa.Runner {
	return func(context.Context, string, ...string) ([]byte, error) { return []byte(out), err }
}

// --- handoff step sequencing ---

// fakeHandoff records the order its methods are called in.
type fakeHandoff struct {
	calls  []string
	failAt string
}

func (f *fakeHandoff) mark(name string) error {
	f.calls = append(f.calls, name)
	if f.failAt == name {
		return errors.New("injected failure at " + name)
	}
	return nil
}

func (f *fakeHandoff) Prepare(context.Context, HandoffOptions) (string, error) {
	return "prepared", f.mark("prepare")
}
func (f *fakeHandoff) Config(context.Context, HandoffOptions) (string, error) {
	return "/cfg", f.mark("config")
}
func (f *fakeHandoff) BaseGenesis(context.Context, HandoffOptions, string) (string, error) {
	return "/base", f.mark("base")
}
func (f *fakeHandoff) Plan(context.Context, HandoffOptions, string) (string, error) {
	return "planned", f.mark("plan")
}
func (f *fakeHandoff) Overlay(context.Context, HandoffOptions) (string, error) {
	return "none", f.mark("overlay")
}
func (f *fakeHandoff) Launch(context.Context, HandoffOptions) (node.NodeSet, error) {
	err := f.mark("launch")
	return node.NodeSet{Nodes: []node.Node{
		{Index: 0, RPCURL: "http://p"}, {Index: 1, RPCURL: "http://v"},
	}}, err
}
func (f *fakeHandoff) WireMesh(context.Context, node.NodeSet) (string, error) {
	return "meshed", f.mark("mesh")
}
func (f *fakeHandoff) DeployGovernance(context.Context, HandoffOptions, node.Node) (string, error) {
	return "deployed", f.mark("governance")
}
func (f *fakeHandoff) EtcdInit(context.Context, HandoffOptions, node.Node) (string, error) {
	return "called", f.mark("etcd-init")
}
func (f *fakeHandoff) ProducerIPC(HandoffOptions, node.Node) string { return "/ipc" }
func (f *fakeHandoff) AwaitFork(context.Context, node.NodeSet, HandoffOptions) (string, error) {
	return "forked", f.mark("await-fork")
}

func TestRunHandoff_VerifiesTheClusterAfterEtcdInit(t *testing.T) {
	c, _ := Find("wemix-wbft")
	f := &fakeHandoff{}
	// etcd-init "succeeds" while the cluster stays empty — the exact shape of
	// the real failure.
	empty := `"{\"governance\":\"0xabc\",\"etcd\":{\"cluster\":\"\"}}"`
	run, err := RunHandoff(context.Background(), c, HandoffOptions{
		DataDir: t.TempDir(), EtcdTimeout: 20 * time.Millisecond,
		Exec: fakeRunner(empty, nil),
	}, f, nil)
	if err != nil {
		t.Fatalf("RunHandoff: %v", err)
	}
	problem, bad := run.FirstProblem()
	if !bad {
		t.Fatal("an empty etcd cluster must fail the run")
	}
	if problem.Step.ID != "verify-etcd" {
		t.Fatalf("failed at %q, want verify-etcd (etcd-init itself returns success)", problem.Step.ID)
	}
	// etcd-init must have been reported OK: the point is that its success is
	// not evidence, not that it errors.
	for _, r := range run.Results {
		if r.Step.ID == "etcd-init" && r.Outcome != OK {
			t.Fatalf("etcd-init outcome = %s, want OK", r.Outcome)
		}
	}
}

func TestRunHandoff_StopsAtTheFailedStep(t *testing.T) {
	c, _ := Find("wemix-wbft")
	f := &fakeHandoff{failAt: "base"}
	run, err := RunHandoff(context.Background(), c, HandoffOptions{
		DataDir: t.TempDir(), Exec: fakeRunner("", nil),
	}, f, nil)
	if err != nil {
		t.Fatalf("RunHandoff: %v", err)
	}
	problem, _ := run.FirstProblem()
	if problem.Step.ID != "base-genesis" {
		t.Fatalf("failed at %q, want base-genesis", problem.Step.ID)
	}
	for _, name := range []string{"plan", "launch", "governance"} {
		for _, c := range f.calls {
			if c == name {
				t.Fatalf("%s ran after an earlier step failed", name)
			}
		}
	}
}

func TestRunHandoff_StopAfterHalts(t *testing.T) {
	c, _ := Find("wemix-wbft")
	f := &fakeHandoff{}
	run, err := RunHandoff(context.Background(), c, HandoffOptions{
		DataDir: t.TempDir(), StopAfter: "build-plan", Exec: fakeRunner("", nil),
	}, f, nil)
	if err != nil {
		t.Fatalf("RunHandoff: %v", err)
	}
	if run.Failed() {
		p, _ := run.FirstProblem()
		t.Fatalf("stopping early is not a failure, got %+v", p)
	}
	for _, call := range f.calls {
		if call == "launch" {
			t.Fatal("launch ran past --stop-after build-plan")
		}
	}
}

func TestRunHandoff_RejectsAnUnknownStopAfter(t *testing.T) {
	c, _ := Find("wemix-wbft")
	if _, err := RunHandoff(context.Background(), c, HandoffOptions{
		DataDir: t.TempDir(), StopAfter: "nope",
	}, &fakeHandoff{}, nil); err == nil {
		t.Fatal("expected an error for an unknown --stop-after step")
	}
}
