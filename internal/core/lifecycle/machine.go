package lifecycle

import (
	"context"
	"fmt"
)

// Handler does one stage's work.
//
// It is given the state the machine is in, because a stage's progress states
// reach the same handler: the handler switches on the state and holds the whole
// stage in one place. It ends by asking for the next state through
// [Machine.Request], and the error it returns is the one that asking produced.
//
// A handler that returns nil without asking for anything leaves the machine
// where it was, which the loop treats as a stall rather than as progress. That
// is deliberate: a stage that does nothing and says nothing is the shape a
// silently skipped step takes, and it should stop the run rather than spin.
type Handler func(ctx context.Context, m *Machine, at Status) error

// Machine walks a lifecycle from one state to the next.
//
// It owns three things and nothing else: where the run is, how many times each
// stage has been entered, and which handler owns which stage. What each stage
// DOES belongs to the handler, so this file has no knowledge of chains,
// workspaces or tests — which is why one machine can drive a composition and a
// test run without either learning about the other.
type Machine struct {
	at       Status
	target   Status
	handlers map[Status]Handler
	entries  map[Status]int
	moves    int
	failedAt error
}

// moveLimit is the most moves one walk may take, whatever the table says.
//
// The block limit bounds moves BETWEEN stages and nothing bounds moves inside
// one, because a stage that walks its own states is doing work — the launch
// stage goes round once per phase, and how many phases there are is the
// family's answer, not this package's. That leaves a handler free to ask for
// the same state forever, and the first walk written against this table did
// exactly that.
//
// So there is a ceiling. It is far above any real walk (the composition is ten
// stages and the largest family declares five phases) and it turns a handler
// that never finishes into a failure with a state attached rather than a
// process that has to be killed.
const moveLimit = 1000

// New returns a machine that will walk from start to target.
//
// It refuses at construction if any stage it could reach has no handler. That
// is the whole reason the check exists here rather than in the loop: a handler
// looked up per step has to be nil-checked per step, and every nil check is a
// place where a missing stage turns into a quiet skip. Checking once means the
// loop can call what it finds.
func New(start, target Status, handlers map[Status]Handler) (*Machine, error) {
	if len(handlers) == 0 {
		return nil, fmt.Errorf("lifecycle: a machine needs handlers")
	}
	missing := unhandled(start, target, handlers)
	if len(missing) > 0 {
		return nil, fmt.Errorf("lifecycle: no handler for %s, which walking %s to %s can reach",
			list(missing), start, target)
	}
	return &Machine{
		at: start, target: target,
		handlers: handlers,
		entries:  map[Status]int{start.Block(): 1},
	}, nil
}

// At is the state the machine is in.
func (m *Machine) At() Status { return m.at }

// Err is why the machine stopped, or nil when it reached its target.
func (m *Machine) Err() error { return m.failedAt }

// Request moves to next, or refuses.
//
// A refusal is an error rather than a silent no-op because the caller is a
// handler that has just done work: if the move it asked for is not one the
// table allows, then either the table or the handler is wrong, and both are
// worth stopping for.
func (m *Machine) Request(next Status) error {
	if _, known := names[next]; !known {
		return fmt.Errorf("lifecycle: %s asked for %s, which is not a declared state", m.at, next)
	}
	// A common failure is reachable from anywhere: what it reports is the
	// workspace failing, not the stage.
	if next.Block() == areaCommon {
		m.at = next
		return nil
	}
	if !permits(m.at, next) {
		return fmt.Errorf("lifecycle: %s may not move to %s", m.at, next)
	}
	// Only a move into a DIFFERENT block is an entry. A stage walking its own
	// progress states has not re-entered anything, and counting those was the
	// first thing the table checks caught: every stage with detail states
	// tripped the limit on its own second state.
	if block := next.Block(); block != m.at.Block() {
		m.entries[block]++
		if m.entries[block] > entryLimit {
			m.at = FailLoop
			return fmt.Errorf("lifecycle: %s was entered %d times, and the limit is %d",
				block, m.entries[block], entryLimit)
		}
	}
	m.moves++
	if m.moves > moveLimit {
		m.at = FailLoop
		return fmt.Errorf("lifecycle: %d moves without reaching the target, and the ceiling is %d",
			m.moves, moveLimit)
	}
	m.at = next
	return nil
}

// Run walks until the target is reached, a failure state is entered, or a
// handler stops making progress.
//
// The loop is the whole control flow: read the state, find who owns it, let
// them move it. What used to decide how far to go (a comparison on a stage name
// inside the loop) and whether to reuse what is there (a comparison on another
// stage name a few lines later) are both states now, so neither is an exception
// wedged into the walk.
func (m *Machine) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			m.failedAt = err
			return err
		}
		if m.at == m.target {
			return nil
		}
		if m.at.IsFailure() {
			if m.failedAt == nil {
				m.failedAt = fmt.Errorf("lifecycle: stopped at %s", m.at)
			}
			return m.failedAt
		}
		h, ok := m.handlers[m.at.Block()]
		if !ok {
			// New refused this at construction, so reaching it means the table
			// allows a move out of the walked range. Say so rather than skip.
			m.at = FailNoHandler
			m.failedAt = fmt.Errorf("lifecycle: nothing owns %s", m.at)
			return m.failedAt
		}
		was := m.at
		if err := h(ctx, m, m.at); err != nil {
			m.failedAt = err
			return err
		}
		if m.at == was {
			m.failedAt = fmt.Errorf("lifecycle: the handler for %s asked for nothing", was)
			return m.failedAt
		}
	}
}

// permits reports whether the table allows this move.
func permits(from, to Status) bool {
	for _, t := range allowed[from] {
		if t == to {
			return true
		}
	}
	return false
}

// unhandled is every block reachable from start on the way to target that no
// handler owns. Failure blocks are not included: a failure ends the walk, so
// nothing has to own it.
func unhandled(start, target Status, handlers map[Status]Handler) []Status {
	seen := map[Status]bool{}
	var missing []Status
	var walk func(Status)
	walk = func(s Status) {
		if seen[s] || s == target || s.IsFailure() {
			return
		}
		seen[s] = true
		if _, ok := handlers[s.Block()]; !ok {
			if !contains(missing, s.Block()) {
				missing = append(missing, s.Block())
			}
		}
		for _, t := range allowed[s] {
			walk(t)
		}
	}
	walk(start)
	return missing
}

func contains(xs []Status, x Status) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func list(xs []Status) string {
	s := ""
	for i, x := range xs {
		if i > 0 {
			s += ", "
		}
		s += x.String()
	}
	return s
}
