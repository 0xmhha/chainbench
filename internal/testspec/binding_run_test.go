package testspec_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// savingAction writes value into the ActionCtx output so the interpreter can
// bind it under the step's "save" name.
type savingAction struct{ value any }

func (a savingAction) Do(_ context.Context, ac *testspec.ActionCtx) error {
	ac.Value = a.value
	return nil
}

// capturingAction records the args it received, after reference substitution.
type capturingAction struct{ got *map[string]any }

func (a capturingAction) Do(_ context.Context, ac *testspec.ActionCtx) error {
	*a.got = ac.Args
	return nil
}

// capturingAssertion records the assertion entry it received, after substitution.
type capturingAssertion struct{ got *map[string]any }

func (a capturingAssertion) Check(_ context.Context, ac *testspec.AssertCtx) (session.AssertResult, error) {
	*a.got = ac.Spec
	return session.AssertResult{Pass: true}, nil
}

func TestRun_StepValueBindsIntoLaterStep(t *testing.T) {
	reg := testspec.NewRegistry(false)
	var gotArgs map[string]any
	reg.RegisterAction("produce", savingAction{value: "0xdeadbeef"})
	reg.RegisterAction("consume", capturingAction{got: &gotArgs})
	reg.RegisterAssertion("noop", fakeAssertion{pass: true})

	spec := testspec.Spec{
		Steps: []map[string]any{
			{"produce": map[string]any{"save": "hash"}},
			{"consume": map[string]any{"hash": "$hash"}},
		},
		Assertions: []map[string]any{{"assert": "noop"}},
	}
	rec := &fakeRecord{}
	st, err := testspec.NewInterpreter(testspec.Deps{Actions: reg}).Run(context.Background(), spec, testEnv(t), rec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st != session.StatusPass {
		t.Fatalf("status = %s, want pass", st)
	}
	if gotArgs["hash"] != "0xdeadbeef" {
		t.Fatalf("consume got hash %#v, want %q", gotArgs["hash"], "0xdeadbeef")
	}
}

func TestRun_StepValueBindsIntoAssertion(t *testing.T) {
	reg := testspec.NewRegistry(false)
	var gotSpec map[string]any
	reg.RegisterAction("produce", savingAction{value: "0xhash"})
	reg.RegisterAssertion("check", capturingAssertion{got: &gotSpec})

	spec := testspec.Spec{
		Steps:      []map[string]any{{"produce": map[string]any{"save": "h"}}},
		Assertions: []map[string]any{{"assert": "check", "hash": "$h", "expected": "0x1"}},
	}
	rec := &fakeRecord{}
	if _, err := testspec.NewInterpreter(testspec.Deps{Actions: reg}).Run(context.Background(), spec, testEnv(t), rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotSpec["hash"] != "0xhash" {
		t.Fatalf("assertion got hash %#v, want %q", gotSpec["hash"], "0xhash")
	}
}

func TestRun_SendTxHashIsBoundWithoutExplicitValue(t *testing.T) {
	// An action that only sets Hash (as sendTx does) still binds, so a spec can
	// reference the transaction it just submitted.
	reg := testspec.NewRegistry(false)
	var gotSpec map[string]any
	reg.RegisterAction("tx", hashOnlyAction{hash: "0xabc"})
	reg.RegisterAssertion("check", capturingAssertion{got: &gotSpec})

	spec := testspec.Spec{
		Steps:      []map[string]any{{"tx": map[string]any{"save": "sent"}}},
		Assertions: []map[string]any{{"assert": "check", "hash": "$sent"}},
	}
	if _, err := testspec.NewInterpreter(testspec.Deps{Actions: reg}).Run(context.Background(), spec, testEnv(t), &fakeRecord{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotSpec["hash"] != "0xabc" {
		t.Fatalf("got %#v, want %q", gotSpec["hash"], "0xabc")
	}
}

type hashOnlyAction struct{ hash string }

func (a hashOnlyAction) Do(_ context.Context, ac *testspec.ActionCtx) error {
	ac.Hash = a.hash
	return nil
}

func TestRun_UnboundReferenceFailsTheStep(t *testing.T) {
	reg := testspec.NewRegistry(false)
	ran := false
	reg.RegisterAction("consume", fakeAction{ran: &ran})
	reg.RegisterAssertion("noop", fakeAssertion{pass: true})

	spec := testspec.Spec{
		Steps:      []map[string]any{{"consume": map[string]any{"hash": "$never"}}},
		Assertions: []map[string]any{{"assert": "noop"}},
	}
	rec := &fakeRecord{}
	st, err := testspec.NewInterpreter(testspec.Deps{Actions: reg}).Run(context.Background(), spec, testEnv(t), rec)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}
	if st != session.StatusFail {
		t.Fatalf("status = %s, want fail", st)
	}
	if ran {
		t.Fatal("the action ran despite an unbound reference")
	}
}

func TestRun_PreActionCanSaveForSteps(t *testing.T) {
	reg := testspec.NewRegistry(false)
	var gotArgs map[string]any
	reg.RegisterAction("prepare", savingAction{value: float64(42)})
	reg.RegisterAction("consume", capturingAction{got: &gotArgs})
	reg.RegisterAssertion("noop", fakeAssertion{pass: true})

	spec := testspec.Spec{
		PreActions: []map[string]any{{"prepare": map[string]any{"save": "n"}}},
		Steps:      []map[string]any{{"consume": map[string]any{"n": "$n"}}},
		Assertions: []map[string]any{{"assert": "noop"}},
	}
	if _, err := testspec.NewInterpreter(testspec.Deps{Actions: reg}).Run(context.Background(), spec, testEnv(t), &fakeRecord{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotArgs["n"] != float64(42) {
		t.Fatalf("got %#v, want float64(42)", gotArgs["n"])
	}
}
