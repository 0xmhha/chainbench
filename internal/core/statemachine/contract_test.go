package statemachine

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// declared is a probe with a contract.
type declared struct {
	*probe
	c Contract
}

func (d declared) Contract() Contract { return d.c }

// contractTree is P over A and B, every state declared, contracts required.
func contractTree(t *testing.T, p, a, b Contract) (*Machine, declared, declared, declared) {
	t.Helper()
	var trace []string
	mk := func(name StateName, c Contract) declared {
		return declared{probe: &probe{name: name, trace: &trace}, c: c}
	}
	dp, da, db := mk("P", p), mk("A", a), mk("B", b)
	m := New("contract", nil)
	m.RequireContracts()
	m.Add(dp, nil)
	m.Add(da, dp)
	m.Add(db, dp)
	return m, dp, da, db
}

// TestContract_IgnoredMessagePasses: a message a state declared it ignores passes without a failure.

func TestContract_IgnoredMessagePasses(t *testing.T) {
	m, _, a, _ := contractTree(t, Contract{}, Contract{Ignores: []What{What(noteOther)}}, Contract{})
	if err := m.Start(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if err := m.Send(context.Background(), noteOther); err != nil {
		t.Fatalf("an ignored message failed the send: %v", err)
	}
}

// TestContract_UnhandledBecomesTheFailureEvent: OnUnhandled turns a message nobody handled into the machine's own failure
// event, and the walk ends where every other failure ends.

func TestContract_UnhandledBecomesTheFailureEvent(t *testing.T) {
	m, p, a, b := contractTree(t,
		Contract{Accepts: []What{What(noteDone)}},
		Contract{},
		Contract{})
	p.onProcess = func(m *Machine, msg Message) (bool, error) {
		if msg == noteDone {
			m.TransitionTo(b)
			return true, nil
		}
		return false, nil
	}
	var sawAt StateName
	m.OnUnhandled(func(_ Message, at StateName) Message { sawAt = at; return noteDone })
	if err := m.Start(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if err := m.Send(context.Background(), noteOther); err != nil {
		t.Fatalf("the failure event did not end the walk cleanly: %v", err)
	}
	if sawAt != "A" || m.Current().Name() != "B" {
		t.Errorf("unhandled at %q ended in %s, want A and B", sawAt, m.Current().Name())
	}
}

// TestContract_UnhandledFailureEventStops: a failure event that is itself unhandled stops the machine with ErrUnhandled
// instead of making another.

func TestContract_UnhandledFailureEventStops(t *testing.T) {
	m, _, a, _ := contractTree(t, Contract{}, Contract{}, Contract{})
	m.OnUnhandled(func(Message, StateName) Message { return noteDone })
	if err := m.Start(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if err := m.Send(context.Background(), noteOther); !errors.Is(err, ErrUnhandled) {
		t.Fatalf("got %v, want ErrUnhandled", err)
	}
}

// TestContract_UndeclaredHandlingAndSendingAreRefused: a state that handles or sends what it did not declare fails the Send.

func TestContract_UndeclaredHandlingAndSendingAreRefused(t *testing.T) {
	m, _, a, _ := contractTree(t, Contract{}, Contract{}, Contract{})
	a.onProcess = func(*Machine, Message) (bool, error) { return true, nil }
	if err := m.Start(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if err := m.Send(context.Background(), noteGo); err == nil || !strings.Contains(err.Error(), "does not accept") {
		t.Errorf("an undeclared handling returned %v", err)
	}

	m2, _, a2, _ := contractTree(t, Contract{}, Contract{}, Contract{})
	a2.onEnter = func(m *Machine) { m.SendSelf(noteDone) }
	if err := m2.Start(context.Background(), a2); err == nil || !strings.Contains(err.Error(), "does not emit") {
		t.Errorf("an undeclared send returned %v", err)
	}
}

// TestContract_UndeclaredStateIsRefusedAtStart: Start refuses a tree with an undeclared state when contracts are required.

func TestContract_UndeclaredStateIsRefusedAtStart(t *testing.T) {
	var trace []string
	m := New("contract", nil)
	m.RequireContracts()
	bare := &probe{name: "Bare", trace: &trace}
	m.Add(bare, nil)
	if err := m.Start(context.Background(), bare); err == nil || !strings.Contains(err.Error(), "Bare") {
		t.Errorf("an undeclared state was accepted: %v", err)
	}
}

// TestContract_AuditFindsAnEmissionNobodyTakes: Audit finds a declared emission that nothing on its path takes.

func TestContract_AuditFindsAnEmissionNobodyTakes(t *testing.T) {
	m, _, _, _ := contractTree(t,
		Contract{},
		Contract{Emits: []What{What(noteDone)}},
		Contract{Accepts: []What{What(noteDone)}}) // a sibling: not on A's path
	if err := m.Audit(); err == nil || !strings.Contains(err.Error(), "A emits") {
		t.Errorf("audit missed a message only a sibling accepts: %v", err)
	}
	m2, _, _, _ := contractTree(t,
		Contract{Accepts: []What{What(noteDone)}},
		Contract{Emits: []What{What(noteDone)}},
		Contract{})
	if err := m2.Audit(); err != nil {
		t.Errorf("audit refused a message the parent accepts: %v", err)
	}
}

// TestRequireAt: RequireAt accepts only the named states.

func TestRequireAt(t *testing.T) {
	m, _, a, b := contractTree(t, Contract{}, Contract{}, Contract{})
	if err := m.Start(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if err := m.RequireAt(a); err != nil {
		t.Errorf("at A, RequireAt(A) = %v", err)
	}
	if err := m.RequireAt(b); !errors.Is(err, ErrNotTerminal) || !strings.Contains(err.Error(), "P/A") {
		t.Errorf("at A, RequireAt(B) = %v", err)
	}
}
