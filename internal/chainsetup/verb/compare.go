package verb

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/preflight"
)

// Composing against a target that may already hold what the request wants.
//
// This is the comparison a suite makes before it builds anything: what is on
// the target against what this run declares. It was a switch over four verdicts
// written outside this package, and the four things it did — leave the network
// alone, restart the nodes that differ, stop and recompose, compose — are four
// states with four moves out of them.
//
// What the switch could not say is where a run WAS. A run that recomposed and
// then failed in the genesis step reported a genesis failure with no trace of
// the comparison that sent it there; now the walk passes through
// CompareChainNetworkDiffers and the state says so.

// stateOfVerdict is which state a preflight verdict is.
//
// The two vocabularies are kept apart on purpose. preflight answers "how much
// has to be rebuilt", which is a fact about two chains and nothing to do with a
// walk; the state is where that answer puts the run. Mapping them here means
// preflight does not have to know a machine exists.
func stateOfVerdict(v preflight.Verdict) (lifecycle.Status, error) {
	switch v {
	case preflight.Reuse:
		return lifecycle.CompareChainSame, nil
	case preflight.RebuildNodes:
		return lifecycle.CompareChainNodesDiffer, nil
	case preflight.RebuildAll:
		return lifecycle.CompareChainNetworkDiffers, nil
	case preflight.Compose:
		return lifecycle.CompareChainNothingComposed, nil
	}
	return 0, fmt.Errorf("chainsetup: preflight returned %s, which is not a verdict this knows", v)
}

// CompareIn asks for a composition that reuses what the target already has when
// the comparison says it can.
type CompareIn struct {
	// Up is the composition to reach, and what the comparison compares against.
	Up chainsetup.NetUpIn
}

// CompareOut is what the comparison and the composition did.
type CompareOut struct {
	NetUpOut
	// Decision is the comparison as a report prints it, verdict and reasons.
	Decision string
	// At is the state the walk ended in. A caller that stopped at the check has
	// ChainVerify; one that failed has the failure.
	At lifecycle.Status
}

// NetUpComparing composes the network the request declares, reusing what is on
// the target when the comparison says it can.
//
// Four ways through, and they are the four states rather than four arms of a
// switch: the network is the one wanted (nothing to do), some nodes differ
// (bring those back), a network-wide fact differs (stop, then compose), or
// nothing is composed (compose).
//
// It stops at ChainVerify. Whether the network that resulted is producing is a
// question this package cannot answer — the readiness gate belongs to whoever
// owns the monitor — so the walk hands it over rather than guessing.
func NetUpComparing(ctx context.Context, d chainsetup.Deps, in CompareIn) (CompareOut, error) {
	var out CompareOut
	m, err := lifecycle.New(lifecycle.CompareChain, lifecycle.ChainVerify,
		compareHandlers(ctx, d, in, &out))
	if err != nil {
		return out, err
	}
	rerr := m.Run(ctx)
	out.At = m.At()
	return out, rerr
}

// compareHandlers is the comparison's four states plus the composition's own,
// so a verdict that says "compose" walks straight into the stages.
func compareHandlers(ctx context.Context, d chainsetup.Deps, in CompareIn, out *CompareOut) map[lifecycle.Status]lifecycle.Handler {
	up := in.Up
	steps := upSteps(ctx, d, up)
	// The composition's own record, written by the same closure the list walk
	// used, so a network composed through the comparison and one composed
	// directly leave the same record.
	run := func(name string) ([]lifecycle.Status, error) {
		r, err := steps[name]()
		if err != nil {
			werr := fmt.Errorf("chainsetup: chain up: %s: %w", name, err)
			markStepFailed(d, up.DataDir, name, werr)
			return r.Passed, werr
		}
		out.Steps = append(out.Steps, name+": "+r.Detail)
		return r.Passed, nil
	}

	// decision is taken once, by the compare state, and read by the state it
	// moves to. Holding it here rather than taking it twice is what keeps the
	// stop from being decided against a different answer than the move was.
	var decision preflight.Decision

	handlers := upHandlers(run)
	handlers[lifecycle.CompareChain] = func(ctx context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		switch at {
		case lifecycle.CompareChain:
			decision = compareWorkspace(ctx, d, up)
			out.Decision = decision.String()
			next, err := stateOfVerdict(decision.Verdict)
			if err != nil {
				if rerr := m.Request(lifecycle.CompareChainFailUnreadable); rerr != nil {
					return rerr
				}
				return err
			}
			return m.Request(next)

		case lifecycle.CompareChainSame:
			out.Steps = append(out.Steps, "preflight: reuse — "+decision.String())
			return m.Request(lifecycle.ChainVerify)

		case lifecycle.CompareChainNodesDiffer:
			// Only the nodes the comparison named. Composing again would
			// rewrite inputs every other node is already running on.
			for _, idx := range decision.Nodes {
				st, err := NetRestart(ctx, d, NetRestartIn{DataDir: up.DataDir, Node: idx})
				if err != nil {
					if rerr := m.Request(lifecycle.CompareChainFailUnreadable); rerr != nil {
						return rerr
					}
					return fmt.Errorf("chainsetup: preflight restart node%d: %w", idx, err)
				}
				out.Steps = append(out.Steps, "restart: "+st.Detail)
			}
			return m.Request(lifecycle.ChainVerify)

		case lifecycle.CompareChainNetworkDiffers:
			// The network is stopped before it is composed again, and that is
			// what makes "rebuild all" true. The compose steps alone do not
			// deliver it: init and start SKIP a node that still carries a
			// recorded pid, so a second up over a workspace whose nodes are
			// still running rewrites the genesis on disk and leaves every node
			// serving the old one.
			st, err := NetStop(ctx, d, NetStopIn{DataDir: up.DataDir})
			if err != nil {
				if rerr := m.Request(lifecycle.CompareChainFailUnreadable); rerr != nil {
					return rerr
				}
				return fmt.Errorf("chainsetup: preflight stop before rebuild: %w", err)
			}
			out.Steps = append(out.Steps, "stop (rebuild-all): "+st.Detail)
			return m.Request(lifecycle.ChainOpenWorkspace)

		case lifecycle.CompareChainNothingComposed:
			// Nothing to stop.
			return m.Request(lifecycle.ChainOpenWorkspace)
		}
		return fmt.Errorf("chainsetup: the comparison was asked for %s, which nothing sets", at)
	}
	return handlers
}

// compareWorkspace asks the workspace what it has against what is wanted.
//
// A workspace that will not open, or has no node table, is not a failure: it is
// a target with nothing composed on it, which is one of the four answers.
func compareWorkspace(ctx context.Context, d chainsetup.Deps, up chainsetup.NetUpIn) preflight.Decision {
	ws, err := chainsetup.Open(up.DataDir, d.Clock)
	if err != nil || len(ws.State().Nodes) == 0 {
		return preflight.Decision{Verdict: preflight.Compose, Reasons: []string{"nothing is composed on the target"}}
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)
	return ws.Compare(ctx, chainsetup.WantOf(up))
}
