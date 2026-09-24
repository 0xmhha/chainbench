package statemachine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// A machine used to drop a message no state handled: it wrote one log line and
// moved on. That is how a rebuild stalled for a week without an error — the
// message that would have started the composition again was offered to the
// state that sent it and to its ancestors, none of which knew it, while the
// state that did know it sat on a sibling branch. The caller got nil back and
// the failure surfaced two stages later as something else entirely.
//
// So a state now says what it deals with, and the machine holds it to that.

// ErrUnhandled is a message that no state from the current one up to the root
// handled, and that none of them declared it ignores.
var ErrUnhandled = errors.New("statemachine: no state handles the message")

// ErrNotTerminal is a machine that came to rest somewhere its caller does not
// accept as an end: the walk stalled rather than finished or failed.
var ErrNotTerminal = errors.New("statemachine: the machine stopped outside its terminal states")

// Contract is what a state says about the messages it deals with.
//
// Accepts is what its Process handles, Emits what its Enter, Exit and Process
// leave the machine with SendSelf, and Ignores what it deliberately lets pass —
// the one way to have a message go unhandled without that being a failure.
// Refuses marks a state that answers every message outside Accepts with an
// error of its own, which is what a failed state does.
type Contract struct {
	Accepts []What
	Emits   []What
	Ignores []What
	Refuses bool
}

// Declared is a state that states its contract.
type Declared interface {
	Contract() Contract
}

// RequireContracts makes every state of this machine declare its contract and
// keep to it: Start refuses a tree with an undeclared state, and a state that
// handles or sends a message it did not declare fails the Send it did it in.
func (m *Machine) RequireContracts() { m.contracts = true }

// OnUnhandled names what a message nobody handled becomes.
//
// A machine that has a failure path of its own — a failed state reached by a
// failure event — turns the unhandled message into that event here, so the
// walk ends where every other failure ends and the caller reads the reason the
// usual way. Without it, an unhandled message fails the Send with ErrUnhandled.
// If the event f returns is itself unhandled, the Send fails: a failure path
// that nothing handles is not one.
func (m *Machine) OnUnhandled(f func(msg Message, at StateName) Message) { m.onUnhandled = f }

// RequireAt reports whether the machine is at rest in one of states.
//
// An entry point calls it after its Send: returning nil from any other state
// would report a stall as a success.
func (m *Machine) RequireAt(states ...State) error {
	for _, s := range states {
		if s != nil && s.Name() == m.current {
			return nil
		}
	}
	names := make([]string, 0, len(states))
	for _, s := range states {
		if s != nil {
			names = append(names, string(s.Name()))
		}
	}
	return fmt.Errorf("%w: in %s, want one of %s", ErrNotTerminal, m.Path(m.Current()), strings.Join(names, ", "))
}

// States returns the states in the order they were added.
func (m *Machine) States() []State {
	out := make([]State, 0, len(m.order))
	for _, n := range m.order {
		out = append(out, m.info[n].state)
	}
	return out
}

// Audit checks the declared contracts against each other: every state is
// declared, and every message a state says it emits is accepted or ignored by
// that state or one of its ancestors — which is where the machine offers it.
//
// It checks the declarations, not the code; the Send-time checks and a static
// test hold the code to the declarations.
func (m *Machine) Audit() error {
	var problems []string
	for _, name := range m.order {
		s := m.info[name].state
		d, ok := s.(Declared)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s declares no contract", name))
			continue
		}
		for _, w := range d.Contract().Emits {
			if !m.takenOnPath(name, w) {
				problems = append(problems, fmt.Sprintf("%s emits %d, and neither it nor an ancestor accepts or ignores it", name, w))
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("statemachine %s: %s", m.name, strings.Join(problems, "; "))
	}
	return nil
}

// takenOnPath reports whether some state from name up to the root accepts or
// ignores w.
func (m *Machine) takenOnPath(name StateName, w What) bool {
	for n := name; n != ""; n = m.info[n].parent {
		c, ok := m.contractOf(n)
		if !ok {
			continue
		}
		if c.Refuses || slices.Contains(c.Accepts, w) || slices.Contains(c.Ignores, w) {
			return true
		}
	}
	return false
}

// ignoredOnPath reports whether some state from name up to the root ignores w.
func (m *Machine) ignoredOnPath(name StateName, w What) bool {
	for n := name; n != ""; n = m.info[n].parent {
		if c, ok := m.contractOf(n); ok && slices.Contains(c.Ignores, w) {
			return true
		}
	}
	return false
}

// contractOf is the contract of the state called name, if it declares one.
func (m *Machine) contractOf(name StateName) (Contract, bool) {
	i, ok := m.info[name]
	if !ok {
		return Contract{}, false
	}
	d, ok := i.state.(Declared)
	if !ok {
		return Contract{}, false
	}
	return d.Contract(), true
}

// checkDeclared is Start's half of RequireContracts.
func (m *Machine) checkDeclared() error {
	if !m.contracts {
		return nil
	}
	var missing []string
	for _, name := range m.order {
		if _, ok := m.info[name].state.(Declared); !ok {
			missing = append(missing, string(name))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("statemachine %s: states without a contract: %s", m.name, strings.Join(missing, ", "))
	}
	return nil
}

// checkHandled is the Send-time check that a state handled only what it
// declared.
func (m *Machine) checkHandled(name StateName, msg Message) {
	if !m.contracts {
		return
	}
	c, ok := m.contractOf(name)
	if !ok || c.Refuses || slices.Contains(c.Accepts, msg.What()) {
		return
	}
	m.misuse(fmt.Sprintf("%s handled %d, which its contract does not accept", name, msg.What()))
}

// checkEmitted is the Send-time check that a state sent only what it declared.
func (m *Machine) checkEmitted(msg Message) {
	if !m.contracts || m.acting == "" {
		return
	}
	c, ok := m.contractOf(m.acting)
	if !ok || slices.Contains(c.Emits, msg.What()) {
		return
	}
	m.misuse(fmt.Sprintf("%s sent %d, which its contract does not emit", m.acting, msg.What()))
}

// ContractTable renders every state's contract as a markdown table, in the
// order the states were added, with name turning a message number into what it
// is called. A design document keeps this table between markers and a test
// compares the two, so the document cannot drift from the declarations.
func (m *Machine) ContractTable(name func(What) string) string {
	var b strings.Builder
	b.WriteString("| 상태 | Accepts | Emits | Ignores |\n|---|---|---|---|\n")
	list := func(ws []What) string {
		if len(ws) == 0 {
			return "—"
		}
		parts := make([]string, len(ws))
		for i, w := range ws {
			parts[i] = "`" + name(w) + "`"
		}
		return strings.Join(parts, ", ")
	}
	for _, n := range m.order {
		c, _ := m.contractOf(n)
		acc := list(c.Accepts)
		if c.Refuses {
			acc += " (그 밖은 거절)"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", n, acc, list(c.Emits), list(c.Ignores))
	}
	return b.String()
}
