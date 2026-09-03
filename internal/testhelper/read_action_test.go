package testhelper

import (
	"context"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestReadAction_SavesRPCValue(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_call": "0x00000000000000000000000000000000000000000000000000000000000003e8",
	})
	d := deps()
	env := envWithNode(t, srv.URL)

	act, ok := d.Actions.Action(interp.ActionRead)
	if !ok {
		t.Fatal("read action not registered")
	}
	ac := &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"source": "call",
		"to":     "0x1000000000000000000000000000000000000000",
		"data":   "0x18160ddd",
		"save":   "supply",
	}}
	if err := act.Do(context.Background(), ac); err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "0x00000000000000000000000000000000000000000000000000000000000003e8"
	if ac.Value != want {
		t.Fatalf("Value = %#v, want %q", ac.Value, want)
	}
}

func TestReadAction_RejectsUnknownSource(t *testing.T) {
	d := deps()
	act, _ := d.Actions.Action(interp.ActionRead)
	err := act.Do(context.Background(), &interp.ActionCtx{Env: envWithNode(t, "http://unused"), Deps: &d,
		Args: map[string]any{"source": "nosuchreader"}})
	if err == nil {
		t.Fatal("expected an error for an unknown source")
	}
}

func TestReadAction_RequiresSource(t *testing.T) {
	d := deps()
	act, _ := d.Actions.Action(interp.ActionRead)
	if err := act.Do(context.Background(), &interp.ActionCtx{Env: envWithNode(t, "http://unused"), Deps: &d,
		Args: map[string]any{}}); err == nil {
		t.Fatal("expected an error when source is missing")
	}
}

// End-to-end: read a value, then assert another read against it. This is the
// cross-call comparison the legacy anzeon suite needs (totalSupply >= balance).
func TestRun_CrossCallComparison(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_call": "0x0000000000000000000000000000000000000000000000000000000000000064",
	})
	d := deps()
	env := envWithNode(t, srv.URL)

	spec := dsl.Spec{
		Steps: []map[string]any{
			{interp.ActionRead: map[string]any{"source": "call", "to": "0xaaa", "data": "0x18160ddd", "save": "supply"}},
		},
		Assertions: []map[string]any{
			{"assert": assertCall, "to": "0xaaa", "data": "0x70a08231", "expected": "$supply"},
		},
	}
	st, err := interp.NewInterpreter(d).Run(context.Background(), spec, env, &recordStub{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st != session.StatusPass {
		t.Fatalf("status = %s, want pass", st)
	}
}

// recordStub is a no-op TestRecord for interpreter-level tests.
type recordStub struct{ asserts []session.AssertResult }

func (r *recordStub) Dir() string                     { return "" }
func (r *recordStub) SetEnvRef(string)                {}
func (r *recordStub) Spec([]byte)                     {}
func (r *recordStub) Step(int, session.StepResult)    {}
func (r *recordStub) Assert(a session.AssertResult)   { r.asserts = append(r.asserts, a) }
func (r *recordStub) Status(session.TestStatus)       {}
func (r *recordStub) Reason(string)                   {}
func (r *recordStub) PostAction(session.PostResult)   {}
func (r *recordStub) Artifacts(session.TestArtifacts) {}

func TestReaderNames_CoverEveryRPCReadingAssertion(t *testing.T) {
	// Every assertion that reads one value must be usable as a read source:
	// they are the same vocabulary, and a name that works in "assert" but not in
	// "read" is a trap the spec author only hits at run time.
	d := deps()
	for _, name := range readerNames() {
		if _, ok := d.Actions.Assertion(name); !ok {
			t.Errorf("read source %q is not registered as an assertion", name)
		}
	}
	for _, want := range []string{"rpcCall", "gasPrice", "logs", "call", "balanceAt"} {
		if _, ok := d.Actions.Reader(want); !ok {
			t.Errorf("assertion %q should also be a read source", want)
		}
	}
}
