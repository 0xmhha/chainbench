package statemachine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// These are the checks the reference's own test makes (StateMachineTest.java),
// written against this machine: what order enter and exit run in, that a
// message nobody handles goes up to the parent, that a self message is handled
// after the move rather than during it, and that the two things a state may
// not do are refused.

// note is a message that is nothing but its number.
type note int

func (n note) What() What { return What(n) }

const (
	noteGo note = iota + 1
	noteDone
	noteOther
)

// probe is a state that writes down every call it gets.
//
// The hooks are how a case says what this state does — move somewhere, leave
// itself a message, fail — without a new type per case.
type probe struct {
	name      StateName
	trace     *[]string
	onEnter   func(m *Machine)
	onExit    func(m *Machine)
	onProcess func(m *Machine, msg Message) (bool, error)
	enterErr  error
	exitErr   error
}

func (p *probe) Name() StateName { return p.name }

func (p *probe) Enter(_ context.Context, m *Machine) error {
	*p.trace = append(*p.trace, string(p.name)+".enter")
	if p.onEnter != nil {
		p.onEnter(m)
	}
	return p.enterErr
}

func (p *probe) Exit(_ context.Context, m *Machine) error {
	*p.trace = append(*p.trace, string(p.name)+".exit")
	if p.onExit != nil {
		p.onExit(m)
	}
	return p.exitErr
}

func (p *probe) Process(_ context.Context, m *Machine, msg Message) (bool, error) {
	*p.trace = append(*p.trace, string(p.name)+".process")
	if p.onProcess != nil {
		return p.onProcess(m, msg)
	}
	return false, nil
}

// tree is the shape both ordering cases use:
//
//	P
//	  S1
//	    S3
//	  S2
//	    S4
type tree struct {
	m                 *Machine
	trace             []string
	p, s1, s2, s3, s4 *probe
	clock             time.Time
}

func newTree(t *testing.T) *tree {
	t.Helper()
	tr := &tree{}
	mk := func(name StateName) *probe { return &probe{name: name, trace: &tr.trace} }
	tr.p, tr.s1, tr.s2, tr.s3, tr.s4 = mk("P"), mk("S1"), mk("S2"), mk("S3"), mk("S4")

	tr.m = New("test", nil)
	tr.m.Add(tr.p, nil)
	tr.m.Add(tr.s1, tr.p)
	tr.m.Add(tr.s3, tr.s1)
	tr.m.Add(tr.s2, tr.p)
	tr.m.Add(tr.s4, tr.s2)

	// A fixed clock: the log's times are compared, and time.Now would make the
	// comparison about when the test ran.
	tr.clock = time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	tr.m.now = func() time.Time { tr.clock = tr.clock.Add(time.Second); return tr.clock }
	return tr
}

// reset drops the trace, so a case can talk about what happened after Start.
func (tr *tree) reset() { tr.trace = nil }

func (tr *tree) got() string { return strings.Join(tr.trace, " ") }

// TestStart_EntersFromTheOutermostAncestorDown: a parent is entered before its
// child, so a child's Enter can rely on what its parent armed.
func TestStart_EntersFromTheOutermostAncestorDown(t *testing.T) {
	tr := newTree(t)
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	if want := "P.enter S1.enter S3.enter"; tr.got() != want {
		t.Errorf("start entered %q, want %q", tr.got(), want)
	}
	if tr.m.Current() != tr.s3 {
		t.Errorf("current is %v, want S3", tr.m.Current())
	}
}

// TestTransition_ExitsDeepestFirstAndLeavesTheCommonAncestorAlone is the
// reference's own ordering case: S3 -> S4 exits S3 then S1, enters S2 then S4,
// and never touches P.
//
// P being left alone is the point of a hierarchy. If a move within it re-entered
// the parent, every state below would have to be written as though its parent's
// setup could vanish under it.
func TestTransition_ExitsDeepestFirstAndLeavesTheCommonAncestorAlone(t *testing.T) {
	tr := newTree(t)
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s4); return true, nil }
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	tr.reset()

	if err := tr.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	if want := "S3.process S3.exit S1.exit S2.enter S4.enter"; tr.got() != want {
		t.Errorf("moved as %q, want %q", tr.got(), want)
	}
	if strings.Contains(tr.got(), "P.") {
		t.Error("P was entered or exited, and it is the common ancestor of both states")
	}
}

// TestTransition_ToItselfExitsThenEnters.
//
// This is the one place the synchronous reference and the asynchronous one
// disagree, and the asynchronous one is right for us: SyncStateMachine returns
// early when the destination equals the current state, which makes
// TransitionTo(self) a silent no-op. StateMachine.java enters the destination
// "even if it is active ... if we are exiting/entering the current state"
// (StateMachine.java:1105-1112). Doing the stage over is a real move here — a
// launch phase retried, a resumed state re-run — and Exit undoes what Enter
// armed, so skipping the pair would leave the second Enter arming on top of the
// first.
func TestTransition_ToItselfExitsThenEnters(t *testing.T) {
	tr := newTree(t)
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s3); return true, nil }
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	tr.reset()

	if err := tr.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	if want := "S3.process S3.exit S3.enter"; tr.got() != want {
		t.Errorf("self transition did %q, want %q", tr.got(), want)
	}
	if strings.Contains(tr.got(), "S1.") {
		t.Error("S1 was touched, and a state moving to itself does not leave its parent")
	}
}

// TestProcess_GoesUpToTheParentUntilSomebodyHandlesIt.
func TestProcess_GoesUpToTheParentUntilSomebodyHandlesIt(t *testing.T) {
	tr := newTree(t)
	tr.p.onProcess = func(*Machine, Message) (bool, error) { return true, nil }
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	tr.reset()

	if err := tr.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	if want := "S3.process S1.process P.process"; tr.got() != want {
		t.Errorf("offered as %q, want %q", tr.got(), want)
	}
	last := lastRec(t, tr.m)
	if last.Processed != "P" || last.Original != "S3" {
		t.Errorf("log says %s handled what reached %s, want P and S3", last.Processed, last.Original)
	}
}

// TestProcess_AMessageNobodyHandlesIsRecordedAndNotAnError.
//
// Refusing it would make every machine answer for messages meant for a state it
// is not in, which is normal: an observer's event arriving mid-composition has
// no handler and is not a fault. The log is where it shows up.
func TestProcess_AMessageNobodyHandlesIsRecordedAndNotAnError(t *testing.T) {
	tr := newTree(t)
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	if err := tr.m.Send(context.Background(), noteOther); err != nil {
		t.Fatalf("an unhandled message failed the send: %v", err)
	}
	last := lastRec(t, tr.m)
	if last.Processed != "" {
		t.Errorf("log names %s as the handler of a message nobody handled", last.Processed)
	}
	if last.What != What(noteOther) || last.Original != "S3" {
		t.Errorf("log recorded what=%d at %s, want %d at S3", last.What, last.Original, noteOther)
	}
}

// TestSendSelf_IsHandledAfterTheMoveByTheNewState.
//
// This is what lets Enter do the work. Enter cannot move, so it says what
// happened; by the time anyone reads that, the machine is in the state the move
// led to, and that state decides what it means.
func TestSendSelf_IsHandledAfterTheMoveByTheNewState(t *testing.T) {
	tr := newTree(t)
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s4); return true, nil }
	tr.s4.onEnter = func(m *Machine) { m.SendSelf(noteDone) }
	var handledIn StateName
	tr.s2.onProcess = func(m *Machine, _ Message) (bool, error) {
		handledIn = m.current
		return true, nil
	}
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	tr.reset()

	if err := tr.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	want := "S3.process S3.exit S1.exit S2.enter S4.enter S4.process S2.process"
	if tr.got() != want {
		t.Errorf("got %q, want %q", tr.got(), want)
	}
	if handledIn != "S4" {
		t.Errorf("the self message was handled while current was %s, want S4", handledIn)
	}
	if len(tr.m.self) != 0 || len(tr.m.inbox) != 0 {
		t.Errorf("Send returned with %d self and %d inbox messages left", len(tr.m.self), len(tr.m.inbox))
	}
}

// TestSendSelf_Chains: a self message whose handling moves again, whose Enter
// leaves another one. One Send carries the whole chain.
func TestSendSelf_Chains(t *testing.T) {
	tr := newTree(t)
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s4); return true, nil }
	tr.s4.onEnter = func(m *Machine) { m.SendSelf(noteDone) }
	tr.s4.onProcess = func(m *Machine, msg Message) (bool, error) {
		if msg.What() == What(noteDone) {
			m.TransitionTo(tr.s3)
			return true, nil
		}
		return false, nil
	}
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	tr.reset()

	if err := tr.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	want := "S3.process S3.exit S1.exit S2.enter S4.enter S4.process S4.exit S2.exit S1.enter S3.enter"
	if tr.got() != want {
		t.Errorf("chain ran %q, want %q", tr.got(), want)
	}
	if tr.m.Current() != tr.s3 {
		t.Errorf("chain ended in %v, want S3", tr.m.Current())
	}
}

// TestMisuseIsRefused covers the two things a state may not do and the two a
// caller may not.
func TestMisuseIsRefused(t *testing.T) {
	cases := []struct {
		name string
		arm  func(tr *tree)
		send func(tr *tree) error
		want string
	}{
		{
			name: "TransitionTo inside Enter",
			arm:  func(tr *tree) { tr.s4.onEnter = func(m *Machine) { m.TransitionTo(tr.s3) } },
			want: "inside Enter or Exit",
		},
		{
			name: "TransitionTo inside Exit",
			arm:  func(tr *tree) { tr.s3.onExit = func(m *Machine) { m.TransitionTo(tr.s3) } },
			want: "inside Enter or Exit",
		},
		{
			name: "TransitionTo twice in one message",
			arm: func(tr *tree) {
				tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) {
					m.TransitionTo(tr.s4)
					m.TransitionTo(tr.s4)
					return true, nil
				}
			},
			want: "twice in one message",
		},
		{
			name: "SendSelf from outside any state",
			arm:  func(tr *tree) {},
			send: func(tr *tree) error {
				tr.m.SendSelf(noteDone)
				return tr.m.Send(context.Background(), noteGo)
			},
			want: "SendSelf outside",
		},
		{
			name: "Send from inside a state",
			arm: func(tr *tree) {
				tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) {
					return true, m.Send(context.Background(), noteOther)
				}
			},
			want: "re-entrant",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := newTree(t)
			// Every case but the last needs a move to S4 to reach its misuse.
			tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s4); return true, nil }
			c.arm(tr)
			if err := tr.m.Start(context.Background(), tr.s3); err != nil {
				t.Fatal(err)
			}
			send := c.send
			if send == nil {
				send = func(tr *tree) error { return tr.m.Send(context.Background(), noteGo) }
			}
			err := send(tr)
			if err == nil {
				t.Fatalf("the misuse was accepted (trace: %s)", tr.got())
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("refused with %q, want it to say %q", err, c.want)
			}
		})
	}
}

// TestMisuse_DoesNotStickToTheNextSend: a refusal is the fault of the message
// it happened in, and reporting it again on the next one would send a reader
// looking at the wrong message.
func TestMisuse_DoesNotStickToTheNextSend(t *testing.T) {
	tr := newTree(t)
	tr.s3.onProcess = func(m *Machine, msg Message) (bool, error) {
		if msg.What() == What(noteGo) {
			m.TransitionTo(tr.s4)
			m.TransitionTo(tr.s4)
		}
		return true, nil
	}
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	if err := tr.m.Send(context.Background(), noteGo); err == nil {
		t.Fatal("the double TransitionTo was accepted")
	}
	if err := tr.m.Send(context.Background(), noteOther); err != nil {
		t.Errorf("the next message inherited the refusal: %v", err)
	}
}

// TestPost_OnlyEnqueuesAndIsDrainedAtTheEndOfTheNextSend.
//
// This is the shape a child machine reports through, so what matters is that it
// does not handle anything at the moment it is called.
func TestPost_OnlyEnqueuesAndIsDrainedAtTheEndOfTheNextSend(t *testing.T) {
	tr := newTree(t)
	tr.p.onProcess = func(*Machine, Message) (bool, error) { return true, nil }
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	tr.reset()

	tr.m.Post(noteOther)
	if len(tr.trace) != 0 {
		t.Fatalf("Post handled something on the spot: %s", tr.got())
	}
	if err := tr.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	// noteGo first, then the posted one — the inbox waits for the self queue.
	if want := "S3.process S1.process P.process S3.process S1.process P.process"; tr.got() != want {
		t.Errorf("drained as %q, want %q", tr.got(), want)
	}
	if len(tr.m.inbox) != 0 {
		t.Errorf("Send returned with %d in the inbox", len(tr.m.inbox))
	}
}

// TestPost_FromAChildReachesTheController is the direction that keeps two
// synchronous machines from being re-entrant: the child only enqueues.
func TestPost_FromAChildReachesTheController(t *testing.T) {
	parent := newTree(t)
	parent.p.onProcess = func(*Machine, Message) (bool, error) { return true, nil }
	if err := parent.m.Start(context.Background(), parent.s3); err != nil {
		t.Fatal(err)
	}

	child := New("child", parent.m)
	if child.Controller() != parent.m {
		t.Fatal("the child does not know its controller")
	}
	child.Controller().Post(noteDone)
	if len(parent.m.inbox) != 1 {
		t.Fatalf("the controller's inbox holds %d, want 1", len(parent.m.inbox))
	}
	parent.reset()
	if err := parent.m.Send(context.Background(), noteGo); err != nil {
		t.Fatal(err)
	}
	if len(parent.m.inbox) != 0 {
		t.Error("the posted message was not drained")
	}
}

// TestErrorFromAState_AbortsTheSend.
func TestErrorFromAState_AbortsTheSend(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name string
		arm  func(tr *tree)
		want string
	}{
		{
			name: "Process",
			arm: func(tr *tree) {
				tr.s3.onProcess = func(*Machine, Message) (bool, error) { return false, boom }
			},
			want: "S3 processing",
		},
		{
			name: "Exit",
			arm:  func(tr *tree) { tr.s3.exitErr = boom },
			want: "leaving S3",
		},
		{
			name: "Enter",
			arm:  func(tr *tree) { tr.s4.enterErr = boom },
			want: "entering S4",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := newTree(t)
			tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s4); return true, nil }
			c.arm(tr)
			if err := tr.m.Start(context.Background(), tr.s3); err != nil {
				t.Fatal(err)
			}
			err := tr.m.Send(context.Background(), noteGo)
			if err == nil {
				t.Fatal("the failing state did not fail the send")
			}
			if !errors.Is(err, boom) {
				t.Errorf("%v does not wrap the state's own error", err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("refused with %q, want it to say %q", err, c.want)
			}
		})
	}
}

// TestDrain_StopsAStateMachineThatLoops.
//
// Two states answering each other is a loop with no I/O in it, so nothing else
// would ever end it. The old machine had a failure state for this.
func TestDrain_StopsAStateMachineThatLoops(t *testing.T) {
	tr := newTree(t)
	tr.s3.onEnter = func(m *Machine) { m.SendSelf(noteDone) }
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s3); return true, nil }
	err := tr.m.Start(context.Background(), tr.s3)
	if err == nil {
		t.Fatal("a loop ran to completion")
	}
	if !strings.Contains(err.Error(), "looping") {
		t.Errorf("stopped with %q, want it to say the states are looping", err)
	}
}

// TestDrain_HonoursCancellation: a cancelled context stops the chain between
// messages, so a caller that gave up is not kept waiting for the rest.
func TestDrain_HonoursCancellation(t *testing.T) {
	tr := newTree(t)
	ctx, cancel := context.WithCancel(context.Background())
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) {
		m.SendSelf(noteDone)
		cancel()
		return true, nil
	}
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	err := tr.m.Send(ctx, noteGo)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("drain returned %v, want a cancellation", err)
	}
}

// TestAddIsCheckedAtStart: the tree is built without error checks so that it
// reads as the tree, so Start is where a bad one has to be caught.
func TestAddIsCheckedAtStart(t *testing.T) {
	cases := []struct {
		name  string
		build func(m *Machine, tr *[]string)
		want  string
	}{
		{
			name: "two states with one name",
			build: func(m *Machine, tr *[]string) {
				m.Add(&probe{name: "A", trace: tr}, nil)
				m.Add(&probe{name: "A", trace: tr}, nil)
			},
			want: "two states named A",
		},
		{
			name: "a parent that was never added",
			build: func(m *Machine, tr *[]string) {
				m.Add(&probe{name: "A", trace: tr}, &probe{name: "ghost", trace: tr})
			},
			want: "has not been added",
		},
		{
			name:  "a state with no name",
			build: func(m *Machine, tr *[]string) { m.Add(&probe{trace: tr}, nil) },
			want:  "empty name",
		},
		{
			name:  "Add(nil)",
			build: func(m *Machine, _ *[]string) { m.Add(nil, nil) },
			want:  "Add(nil)",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var trace []string
			m := New("test", nil)
			first := &probe{name: "first", trace: &trace}
			m.Add(first, nil)
			c.build(m, &trace)
			err := m.Start(context.Background(), first)
			if err == nil {
				t.Fatal("the tree was accepted")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("refused with %q, want it to say %q", err, c.want)
			}
			if len(trace) != 0 {
				t.Errorf("a refused tree still entered something: %v", trace)
			}
		})
	}
}

// TestStart_Guards covers what a machine refuses before Start and after it.
func TestStart_Guards(t *testing.T) {
	tr := newTree(t)
	if err := tr.m.Send(context.Background(), noteGo); err == nil {
		t.Error("Send before Start was accepted")
	}
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	if err := tr.m.Start(context.Background(), tr.s4); err == nil {
		t.Error("Start was accepted twice")
	}
	if err := tr.m.Start(context.Background(), nil); err == nil {
		t.Error("Start(nil) was accepted")
	}
	tr.m.Add(&probe{name: "late", trace: &tr.trace}, nil)
	if tr.m.setupErr == nil || !strings.Contains(tr.m.setupErr.Error(), "after the machine started") {
		t.Errorf("Add after Start was not recorded: %v", tr.m.setupErr)
	}
}

// TestStart_DrainsWhatEnteringProduced: Start is a complete unit too, so a
// machine whose first state reports at once is already past it when Start
// returns.
func TestStart_DrainsWhatEnteringProduced(t *testing.T) {
	tr := newTree(t)
	tr.s3.onEnter = func(m *Machine) { m.SendSelf(noteDone) }
	tr.s3.onProcess = func(m *Machine, _ Message) (bool, error) { m.TransitionTo(tr.s4); return true, nil }
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	if want := "P.enter S1.enter S3.enter S3.process S3.exit S1.exit S2.enter S4.enter"; tr.got() != want {
		t.Errorf("start ran %q, want %q", tr.got(), want)
	}
}

// TestTreeAndPath: the tree prints as the picture the states were added in, and
// a path names a state by its ancestry, which is the form a record keeps.
func TestTreeAndPath(t *testing.T) {
	tr := newTree(t)
	want := "P\n  S1\n    S3\n  S2\n    S4\n"
	if got := tr.m.Tree(); got != want {
		t.Errorf("tree is\n%s\nwant\n%s", got, want)
	}
	for _, c := range []struct {
		state State
		want  string
	}{
		{tr.p, "P"},
		{tr.s4, "P/S2/S4"},
		{tr.s3, "P/S1/S3"},
		{&probe{name: "stranger", trace: &tr.trace}, "stranger"},
		{nil, ""},
	} {
		if got := tr.m.Path(c.state); got != c.want {
			t.Errorf("path is %q, want %q", got, c.want)
		}
	}
}

// TestLog_KeepsTheLastMessagesAndOverwritesTheOldest.
func TestLog_KeepsTheLastMessagesAndOverwritesTheOldest(t *testing.T) {
	tr := newTree(t)
	tr.p.onProcess = func(*Machine, Message) (bool, error) { return true, nil }
	if err := tr.m.Start(context.Background(), tr.s3); err != nil {
		t.Fatal(err)
	}
	const sent = defaultLogSize + 5
	for i := 0; i < sent; i++ {
		if err := tr.m.Send(context.Background(), note(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	got := tr.m.Dump()
	if len(got) != defaultLogSize {
		t.Fatalf("the log holds %d records, want %d", len(got), defaultLogSize)
	}
	// Oldest first, and the first five are gone.
	if want := What(sent - defaultLogSize + 1); got[0].What != want {
		t.Errorf("the oldest record is what=%d, want %d", got[0].What, want)
	}
	if want := What(sent); got[len(got)-1].What != want {
		t.Errorf("the newest record is what=%d, want %d", got[len(got)-1].What, want)
	}
	for i := 1; i < len(got); i++ {
		if !got[i].Time.After(got[i-1].Time) {
			t.Fatalf("record %d is not after the one before it", i)
		}
	}
}

// lastRec returns the most recent log record.
func lastRec(t *testing.T, m *Machine) LogRec {
	t.Helper()
	recs := m.Dump()
	if len(recs) == 0 {
		t.Fatal("the log is empty")
	}
	return recs[len(recs)-1]
}
