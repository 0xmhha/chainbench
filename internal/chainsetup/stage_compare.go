package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// Composing against a target that may already hold what the request wants.
const (
	nameComparing         statemachine.StateName = "Comparing"
	nameRestartingNodes   statemachine.StateName = "RestartingNodes"
	nameStoppingToRebuild statemachine.StateName = "StoppingToRebuild"
	nameVerifying         statemachine.StateName = "Verifying"
)

// comparing is the check a suite makes before it builds anything: what is on
// the target, against what this run declares.
//
// It was a switch over four verdicts. What the switch could not say is where a
// run WAS: one that recomposed and then failed in the genesis reported a
// genesis failure with no trace of the comparison that sent it there.
//
// The four answers are four moves. The network is the one wanted, so there is
// nothing to do. Some nodes differ, so those come back. A network-wide fact
// differs, so the network stops and is composed again. Nothing is composed, so
// compose.
type comparing struct {
	statemachine.Base
	mg *Manager

	decision preflight.Decision

	restarting *restartingNodes
	stopping   *stoppingToRebuild
}

// newComparing builds the comparison and the one move out of it that works.
func newComparing(mg *Manager) *comparing {
	s := &comparing{mg: mg}
	s.restarting = &restartingNodes{parent: s}
	s.stopping = &stoppingToRebuild{parent: s}
	return s
}

// Name says what this state is called.
func (comparing) Name() statemachine.StateName { return nameComparing }

// leafStates is the moves out of the comparison that have work of their own.
func (s *comparing) leafStates() []statemachine.State {
	return []statemachine.State{s.restarting, s.stopping}
}

// Enter asks the workspace what it has against what is wanted.
func (s *comparing) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	s.decision = compareWorkspace(ctx, s.mg.d, s.mg.request)
	s.mg.note("preflight", s.decision.String())
	m.SendSelf(comparisonMade{Verdict: s.decision.Verdict})
	return nil
}

// Process turns the verdict into the move it is.
//
// Deciding only. Two of the four moves have work in them — bringing nodes back,
// and stopping before a rebuild — and that work is in the states they name.
func (s *comparing) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	c, ok := msg.(comparisonMade)
	if !ok {
		return false, nil
	}
	next, err := s.nextFor(c.Verdict)
	if err != nil {
		s.mg.failure = err
		m.TransitionTo(s.mg.failed)
		return true, nil
	}
	m.TransitionTo(next)
	return true, nil
}

// nextFor is where a verdict puts the run.
//
// The two vocabularies are kept apart on purpose. preflight answers "how much
// has to be rebuilt", which is a fact about two chains and nothing to do with a
// walk; this says where that answer puts the run. A verdict added there without
// a move here is refused by name rather than falling into a default.
func (s *comparing) nextFor(v preflight.Verdict) (statemachine.State, error) {
	switch v {
	case preflight.Reuse:
		return s.mg.verifying, nil
	case preflight.RebuildNodes:
		return s.restarting, nil
	case preflight.RebuildAll:
		return s.stopping, nil
	case preflight.Compose:
		// Nothing composed has nothing to stop.
		return s.mg.stages[0], nil
	}
	return nil, fmt.Errorf("chainsetup: preflight returned %s, which is not a verdict this knows", v)
}

// stoppingToRebuild takes the running network down before it is composed again.
//
// This is what makes "rebuild all" true. The compose steps alone do not deliver
// it: init and start SKIP a node that still carries a recorded pid, so a second
// composition over a workspace whose nodes are still running rewrites the
// genesis on disk and leaves every node serving the old one.
type stoppingToRebuild struct {
	statemachine.Base
	parent *comparing
}

// Name says what this state is called.
func (stoppingToRebuild) Name() statemachine.StateName { return nameStoppingToRebuild }

// Enter stops the network.
func (l *stoppingToRebuild) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := l.parent.mg
	mg.recordPath(l)
	detail, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
		return ws.Stop(ctx)
	})
	if err != nil {
		mg.failure = fmt.Errorf("chainsetup: preflight stop before rebuild: %w", err)
		m.TransitionTo(mg.failed)
		return nil
	}
	mg.note("stop (rebuild-all)", detail)
	m.SendSelf(stoppedToRebuild{})
	return nil
}

// restartingNodes brings back only the nodes the comparison named.
//
// Only those: composing again would rewrite inputs every other node is already
// running on.
type restartingNodes struct {
	statemachine.Base
	parent *comparing
}

// Name says what this state is called.
func (restartingNodes) Name() statemachine.StateName { return nameRestartingNodes }

// Enter restarts each named node.
func (l *restartingNodes) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := l.parent.mg
	mg.recordPath(l)
	for _, idx := range l.parent.decision.Nodes {
		detail, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
			return ws.Restart(ctx, idx)
		})
		if err != nil {
			mg.failure = fmt.Errorf("chainsetup: preflight restart node%d: %w", idx, err)
			m.TransitionTo(mg.failed)
			return nil
		}
		mg.note("restart", detail)
	}
	m.SendSelf(nodesRestarted{})
	return nil
}

// verifying is where a comparison-led composition hands over.
//
// Whether the network that resulted is producing is a question this package
// cannot answer — the readiness gate belongs to whoever owns the monitor — so
// the walk stops here rather than guessing.
type verifying struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (verifying) Name() statemachine.StateName { return nameVerifying }

// Enter records that the composition reached the hand-over.
func (s *verifying) Enter(context.Context, *statemachine.Machine) error {
	s.mg.recordPath(s)
	return nil
}

// compareWorkspace asks the workspace what it has against what is wanted.
//
// A workspace that will not open, or has no node table, is not a failure: it is
// a target with nothing composed on it, which is one of the four answers.
func compareWorkspace(ctx context.Context, d Deps, up ChainUpIn) preflight.Decision {
	ws, err := Open(up.DataDir, d.Clock)
	if err != nil || len(ws.State().Nodes) == 0 {
		return preflight.Decision{Verdict: preflight.Compose, Reasons: []string{"nothing is composed on the target"}}
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)
	return ws.Compare(ctx, WantOf(up))
}
