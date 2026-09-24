package statemachine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// maxDispatchPerSend bounds how many messages one Send may handle.
//
// A state that answers its own message with a move back to where it came from
// is a loop with no I/O in it, so nothing else would ever stop it — the old
// machine had a failure state for exactly this (FailLoop). A thousand is far
// above any real chain of stages and low enough that the loop is reported in
// the same second it starts.
const maxDispatchPerSend = 1000

// stateInfo is one state and where it sits.
type stateInfo struct {
	state  State
	parent StateName // "" when the state has no parent
	active bool      // entered and not yet exited
}

// Machine holds a tree of states and moves through it on messages.
//
// One machine is one goroutine's, and nothing here is safe to use from two.
// That is not a limitation being worked around: the whole model is that Send
// is a complete unit of work, and two callers inside it at once is the thing it
// refuses.
type Machine struct {
	name       string
	controller *Machine

	info  map[StateName]*stateInfo
	order []StateName // the order Add was called in, which is what Tree prints

	current  StateName
	dest     StateName
	haveDest bool

	self  []Message // SendSelf: drained after the transition, in the new state
	inbox []Message // Post: what a child or an observer left

	sending      bool // inside Send or Start
	inState      bool // inside Enter, Exit or Process, where SendSelf is allowed
	inTransition bool // inside Enter or Exit, where TransitionTo is refused

	setupErr error // the first Add that was wrong, reported by Start
	opErr    error // a misuse inside a state, reported by the Send it happened in

	log *ring
	now func() time.Time
}

// New returns an empty machine called name.
//
// controller is the machine this one reports to, or nil when it answers to no
// one. A child posts to its controller and a controller sends into its child;
// see the package doc for why those two directions are not the same call.
func New(name string, controller *Machine) *Machine {
	return &Machine{
		name:       name,
		controller: controller,
		info:       map[StateName]*stateInfo{},
		log:        newRing(defaultLogSize),
		now:        time.Now,
	}
}

// Controller returns the machine this one reports to, or nil.
func (m *Machine) Controller() *Machine { return m.controller }

// Add puts s in the tree under parent, or at the top when parent is nil.
//
// It returns nothing on purpose. The calls that build a tree are meant to be
// read as the tree — indented, one line per state, the way the reference writes
// its addState block — and an error check on every line would bury the picture.
// A bad Add is kept and reported by [Machine.Start], which is before the
// machine has done anything.
func (m *Machine) Add(s, parent State) {
	switch {
	case m.current != "":
		m.setup("Add after the machine started")
		return
	case s == nil:
		m.setup("Add(nil)")
		return
	}
	name := s.Name()
	if name == "" {
		m.setup("a state with an empty name")
		return
	}
	if _, dup := m.info[name]; dup {
		m.setup(fmt.Sprintf("two states named %s", name))
		return
	}
	var pname StateName
	if parent != nil {
		pname = parent.Name()
		if _, ok := m.info[pname]; !ok {
			m.setup(fmt.Sprintf("%s is under %s, which has not been added", name, pname))
			return
		}
	}
	m.info[name] = &stateInfo{state: s, parent: pname}
	m.order = append(m.order, name)
}

// Start enters the machine at initial and drains what that entering produced.
//
// It enters every state from the outermost ancestor down to initial, so a
// parent's Enter runs before its child's, and then handles whatever those
// Enters left with SendSelf. When Start returns, the machine is at rest.
func (m *Machine) Start(ctx context.Context, initial State) error {
	if m.setupErr != nil {
		return m.setupErr
	}
	if m.current != "" {
		return fmt.Errorf("statemachine %s: already started, and now in %s", m.name, m.current)
	}
	if initial == nil {
		return fmt.Errorf("statemachine %s: Start(nil)", m.name)
	}
	if _, ok := m.info[initial.Name()]; !ok {
		return fmt.Errorf("statemachine %s: %s was never added", m.name, initial.Name())
	}

	m.sending = true
	defer func() { m.sending = false }()

	m.dest, m.haveDest = initial.Name(), true
	if err := m.performTransition(ctx); err != nil {
		return err
	}
	if m.opErr != nil {
		return m.takeOpErr()
	}
	return m.drain(ctx)
}

// Send handles one message and everything that follows from it.
//
// The message is offered to the current state and then to each of its parents
// until one handles it; a message nobody handles is recorded and is not an
// error. At most one transition follows, and then the self queue and the inbox
// are drained until both are empty. When Send returns, the machine is at rest.
//
// Calling Send from inside a state is refused. A state that wants to say
// something to its own machine uses [Machine.SendSelf]; a child that wants to
// say something to its controller uses [Machine.Post] on that controller.
func (m *Machine) Send(ctx context.Context, msg Message) error {
	switch {
	case m.sending:
		return fmt.Errorf("statemachine %s: Send is re-entrant, and a message is already being handled", m.name)
	case m.current == "":
		return fmt.Errorf("statemachine %s: Send before Start", m.name)
	case msg == nil:
		return fmt.Errorf("statemachine %s: Send(nil)", m.name)
	}

	m.sending = true
	defer func() { m.sending = false }()

	if err := m.dispatch(ctx, msg); err != nil {
		return err
	}
	return m.drain(ctx)
}

// Post leaves a message for this machine to handle when it next comes to rest.
//
// It is what a child machine calls on its controller, and what an observer
// outside any state calls. It only enqueues: nothing is handled here, so it is
// safe from inside the child's own Send.
func (m *Machine) Post(msg Message) {
	if msg == nil {
		return
	}
	m.inbox = append(m.inbox, msg)
}

// TransitionTo names the state to move to once the current message has been
// handled.
//
// It belongs inside Process and nowhere else, at most once per message. Called
// from Enter or Exit it is refused, because a move that began in the middle of
// another move would leave the machine describing neither state; a state that
// wants to move as a consequence of being entered leaves itself a message
// instead. A refusal fails the [Machine.Send] it happened in.
func (m *Machine) TransitionTo(s State) {
	switch {
	case s == nil:
		m.misuse("TransitionTo(nil)")
	case m.inTransition:
		m.misuse("TransitionTo inside Enter or Exit — leave a self message instead")
	case !m.inState:
		m.misuse("TransitionTo outside Process")
	case m.haveDest:
		m.misuse(fmt.Sprintf("TransitionTo twice in one message, already going to %s", m.dest))
	default:
		name := s.Name()
		if _, ok := m.info[name]; !ok {
			m.misuse(fmt.Sprintf("TransitionTo(%s), which was never added", name))
			return
		}
		m.dest, m.haveDest = name, true
	}
}

// SendSelf leaves a message for this machine, to be handled after the current
// transition finishes — by whichever state is current then, not by this one.
//
// This is how Enter reports what it did: it cannot move, so it says what
// happened and the state that comes next decides what that means.
func (m *Machine) SendSelf(msg Message) {
	switch {
	case msg == nil:
		m.misuse("SendSelf(nil)")
	case !m.inState:
		m.misuse("SendSelf outside Enter, Exit or Process")
	default:
		m.self = append(m.self, msg)
	}
}

// Current returns the state the machine is in, or nil before Start.
//
// Inside Enter and Exit it is still the state being left: current changes once
// the whole transition is done, so that a half-finished move is never visible.
func (m *Machine) Current() State {
	i, ok := m.info[m.current]
	if !ok {
		return nil
	}
	return i.state
}

// Path is s and its ancestors, outermost first, joined with "/" — the form a
// run's record keeps, e.g. "Composition/Composing/BuildingGenesis".
//
// A state this machine does not hold has no ancestors here, so it comes back as
// its own name alone.
func (m *Machine) Path(s State) string {
	if s == nil {
		return ""
	}
	var parts []string
	for name := s.Name(); name != ""; {
		parts = append(parts, string(name))
		i, ok := m.info[name]
		if !ok {
			break
		}
		name = i.parent
	}
	for l, r := 0, len(parts)-1; l < r; l, r = l+1, r-1 {
		parts[l], parts[r] = parts[r], parts[l]
	}
	return strings.Join(parts, "/")
}

// Tree is the state tree as indented text, in the order the states were added.
//
// It exists to be compared against a golden file. The tree is the design's
// picture, and a picture nobody checks is one that drifts from the code under
// it.
func (m *Machine) Tree() string {
	var b strings.Builder
	for _, name := range m.order {
		depth := 0
		for p := m.info[name].parent; p != ""; p = m.info[p].parent {
			depth++
		}
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteString(string(name))
		b.WriteByte('\n')
	}
	return b.String()
}

// Dump returns the messages the machine still remembers, oldest first.
func (m *Machine) Dump() []LogRec { return m.log.all() }

// drain handles the self queue and then the inbox, until both are empty.
//
// Self messages go first at every step: they are what the transition just made
// true, and an inbox message handled before them would be answered by a state
// that has not yet learnt where it is.
func (m *Machine) drain(ctx context.Context) error {
	for n := 0; ; n++ {
		if n >= maxDispatchPerSend {
			return fmt.Errorf("statemachine %s: %d messages in one Send, still in %s — the states are looping",
				m.name, n, m.current)
		}
		var msg Message
		switch {
		case len(m.self) > 0:
			msg, m.self = m.self[0], m.self[1:]
		case len(m.inbox) > 0:
			msg, m.inbox = m.inbox[0], m.inbox[1:]
		default:
			return nil
		}
		if err := m.dispatch(ctx, msg); err != nil {
			return err
		}
	}
}

// dispatch offers one message up the tree, then performs the move it asked for.
func (m *Machine) dispatch(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("statemachine %s: %w", m.name, err)
	}
	original := m.current

	var handled StateName
	m.inState = true
	for name := original; name != ""; name = m.info[name].parent {
		ok, err := m.info[name].state.Process(ctx, m, msg)
		if err != nil {
			m.inState = false
			m.record(msg, name, original, "")
			return fmt.Errorf("statemachine %s: %s processing %v: %w", m.name, name, msg.What(), err)
		}
		if ok {
			handled = name
			break
		}
	}
	m.inState = false
	if m.opErr != nil {
		m.record(msg, handled, original, "")
		return m.takeOpErr()
	}

	dest := m.dest
	if !m.haveDest {
		dest = ""
	}
	if err := m.performTransition(ctx); err != nil {
		m.record(msg, handled, original, dest)
		return err
	}
	m.record(msg, handled, original, dest)
	if m.opErr != nil {
		return m.takeOpErr()
	}
	return nil
}

// performTransition exits up to the nearest active ancestor of the destination
// and enters back down to it.
//
// The destination is entered even when it is already the current state: a
// state that moves to itself exits and enters again, which is how "do this
// stage over" is said. A move that enters nothing is not silently a no-op.
func (m *Machine) performTransition(ctx context.Context) error {
	if !m.haveDest {
		return nil
	}
	dest := m.dest
	m.dest, m.haveDest = "", false

	// entering holds the destination and every inactive ancestor above it,
	// innermost first. The walk stops at the first ancestor already entered;
	// that one is where the exits stop too.
	var entering []StateName
	var ancestor StateName
	for name := dest; ; {
		entering = append(entering, name)
		p := m.info[name].parent
		if p == "" || m.info[p].active {
			ancestor = p
			break
		}
		name = p
	}

	m.inState, m.inTransition = true, true
	defer func() { m.inState, m.inTransition = false, false }()

	for name := m.current; name != "" && name != ancestor; name = m.info[name].parent {
		i := m.info[name]
		if err := i.state.Exit(ctx, m); err != nil {
			return fmt.Errorf("statemachine %s: leaving %s: %w", m.name, name, err)
		}
		i.active = false
	}
	for k := len(entering) - 1; k >= 0; k-- {
		i := m.info[entering[k]]
		if err := i.state.Enter(ctx, m); err != nil {
			return fmt.Errorf("statemachine %s: entering %s: %w", m.name, entering[k], err)
		}
		i.active = true
	}
	m.current = dest
	return nil
}

// record writes one line of the log.
func (m *Machine) record(msg Message, processed, original, dest StateName) {
	m.log.add(LogRec{
		Time:      m.now(),
		What:      msg.What(),
		Processed: processed,
		Original:  original,
		Dest:      dest,
	})
}

// setup keeps the first thing wrong with the tree, for Start to report.
func (m *Machine) setup(what string) {
	if m.setupErr == nil {
		m.setupErr = fmt.Errorf("statemachine %s: %s", m.name, what)
	}
}

// misuse keeps the first thing a state did wrong, for this Send to report.
func (m *Machine) misuse(what string) {
	if m.opErr == nil {
		m.opErr = fmt.Errorf("statemachine %s: %s", m.name, what)
	}
}

// takeOpErr returns the pending misuse and clears it, so the next Send starts
// clean rather than reporting a failure that has already been reported.
func (m *Machine) takeOpErr() error {
	err := m.opErr
	m.opErr = nil
	return err
}
