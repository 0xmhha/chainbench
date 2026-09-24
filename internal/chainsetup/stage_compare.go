package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// Composing against a target that may already hold what the request wants.
const (
	nameChainCompare                statemachine.StateName = "CHAIN_COMPARE"
	nameChainCompareNodesDiffer     statemachine.StateName = "CHAIN_COMPARE_NODES_DIFFER"
	nameChainCompareNetworkDiffers  statemachine.StateName = "CHAIN_COMPARE_NETWORK_DIFFERS"
	nameChainCompareSame            statemachine.StateName = "CHAIN_COMPARE_SAME"
	nameChainCompareNothingComposed statemachine.StateName = "CHAIN_COMPARE_NOTHING_COMPOSED"
	nameChainCompareNetworkStopped  statemachine.StateName = "CHAIN_COMPARE_NETWORK_STOPPED"
)

// comparing is the check a suite makes before it builds anything: what is on
// the target, against what this run declares.
//
// It was a switch over four verdicts. What the switch could not say is where a
// run WAS: one that recomposed and then failed in the genesis reported a
// genesis failure with no trace of the comparison that sent it there.
//
// The five answers are five moves, and each is a state of its own so the record
// says which one a run took. The network is the one wanted, so there is nothing
// to do. Some nodes differ, so those come back. A network-wide fact differs, so
// the network stops and is composed again. Nothing is composed, so compose. The
// network is the one wanted and none of it runs, so its nodes are launched.
//
// Every one of them reports back here, to the parent that chose it. The two
// that used to report to the composition instead — a sibling of this state —
// were dropped, and a rebuild stalled with no error.
type comparing struct {
	statemachine.Base
	mg *Manager

	decision preflight.Decision

	same       *compareOutcome
	restarting *restartingNodes
	stopping   *stoppingToRebuild
	nothing    *compareOutcome
	relaunch   *compareOutcome
}

// newComparing builds the comparison and the one move out of it that works.
func newComparing(mg *Manager) *comparing {
	s := &comparing{mg: mg}
	s.same = &compareOutcome{parent: s, name: nameChainCompareSame, report: networkKept{}}
	s.restarting = &restartingNodes{parent: s}
	s.stopping = &stoppingToRebuild{parent: s}
	s.nothing = &compareOutcome{parent: s, name: nameChainCompareNothingComposed, report: nothingComposed{}}
	s.relaunch = &compareOutcome{parent: s, name: nameChainCompareNetworkStopped, report: networkStopped{}}
	return s
}

// Name says what this state is called.
func (comparing) Name() statemachine.StateName { return nameChainCompare }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (comparing) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: []statemachine.What{eventComparisonMade, eventNetworkKept, eventNothingComposed, eventNodesRestarted, eventStoppedToRebuild, eventNetworkStopped}, Emits: []statemachine.What{eventComparisonMade}}
}

// leafStates is the moves out of the comparison, one per verdict.
func (s *comparing) leafStates() []statemachine.State {
	return []statemachine.State{s.same, s.restarting, s.stopping, s.nothing, s.relaunch}
}

// Enter asks the workspace what it has against what is wanted.
func (s *comparing) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	s.decision = s.mg.compare(ctx, s.mg.d, s.mg.request)
	s.mg.note("preflight", s.decision.String())
	m.SendSelf(comparisonMade{Verdict: s.decision.Verdict})
	return nil
}

// Process turns the verdict into the move it is, and each move's report into
// where the run goes next.
func (s *comparing) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	switch c := msg.(type) {
	case comparisonMade:
		next, err := s.nextFor(c.Verdict)
		if err != nil {
			s.mg.failure = err
			m.TransitionTo(s.mg.failed)
			return true, nil
		}
		m.TransitionTo(next)
	case networkKept, nodesRestarted:
		m.TransitionTo(s.mg.ready)
	case stoppedToRebuild, nothingComposed:
		m.TransitionTo(s.mg.stages[0])
	case networkStopped:
		launch, err := s.mg.stageFor(stepStart)
		if err != nil {
			return true, err
		}
		m.TransitionTo(launch)
	default:
		return false, nil
	}
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
		return s.same, nil
	case preflight.RebuildNodes:
		return s.restarting, nil
	case preflight.RebuildAll:
		return s.stopping, nil
	case preflight.Compose:
		// Nothing composed has nothing to stop.
		return s.nothing, nil
	case preflight.Relaunch:
		return s.relaunch, nil
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
func (stoppingToRebuild) Name() statemachine.StateName { return nameChainCompareNetworkDiffers }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (stoppingToRebuild) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventStoppedToRebuild, eventStageFailed}}
}

// Enter stops the network.
func (l *stoppingToRebuild) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := l.parent.mg
	mg.recordPath(l)
	detail, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
		return ws.Stop(ctx)
	})
	if err != nil {
		m.SendSelf(stageFailed{Step: "stop", Err: fmt.Errorf("chainsetup: preflight stop before rebuild: %w", err)})
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
func (restartingNodes) Name() statemachine.StateName { return nameChainCompareNodesDiffer }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (restartingNodes) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventNodesRestarted, eventStageFailed}}
}

// Enter restarts each named node.
func (l *restartingNodes) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := l.parent.mg
	mg.recordPath(l)
	for _, idx := range l.parent.decision.Nodes {
		detail, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
			return ws.Restart(ctx, idx)
		})
		if err != nil {
			m.SendSelf(stageFailed{Step: "restart", Err: fmt.Errorf("chainsetup: preflight restart node%d: %w", idx, err)})
			return nil
		}
		mg.note("restart", detail)
	}
	m.SendSelf(nodesRestarted{})
	return nil
}

// compareOutcome is a verdict with no work of its own: the record keeps the
// path, and the report tells the comparison where the run goes.
//
// The network being the one wanted used to end in a state called Verifying,
// which verified nothing — whether the network produces is the readiness
// gate's question, and the gate is not in this package. A verdict that needs no
// work now says so and hands the run on.
type compareOutcome struct {
	statemachine.Base
	parent *comparing
	name   statemachine.StateName
	report statemachine.Message
}

// Name says what this state is called.
func (l *compareOutcome) Name() statemachine.StateName { return l.name }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (l *compareOutcome) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{l.report.What()}}
}

// Enter records the verdict and reports it.
func (l *compareOutcome) Enter(_ context.Context, m *statemachine.Machine) error {
	l.parent.mg.recordPath(l)
	m.SendSelf(l.report)
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
