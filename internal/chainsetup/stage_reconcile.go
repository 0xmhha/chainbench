package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// nameChainReconcile is the state that compares a running network with what this
// run would compose.
const nameChainReconcile statemachine.StateName = "CHAIN_RECONCILE"

// reconciling is reuse-if-matching: what is already running, held against what
// this request would build.
//
// It stands between the keys and the genesis because that is the last moment
// the comparison is still true. The genesis step writes, and once it has, what
// was on the target is gone and there is nothing left to compare.
//
// It is not one of the composition's steps. It has no rung in the record and no
// `chain <step>` command; a run either reconciles or it does not, and the
// request is what says which.
type reconciling struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (reconciling) Name() statemachine.StateName { return nameChainReconcile }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (reconciling) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventReconciled, eventReconcileRefused, eventStageFailed}}
}

// Enter makes the comparison and says what it decided.
//
// A refusal is a failure of the run: reuse-if-matching asked to keep a network
// and the network cannot be kept, so composing over it anyway would destroy the
// thing the request was trying to preserve.
func (s *reconciling) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	in := s.mg.request
	gopts, err := GenesisOptsFor(ChainGenesisIn{
		DataDir: s.mg.ws.Dir(), ChainID: in.ChainID, Set: in.GenesisSet,
		OverlayPath: in.OverlayPath, GenesisExisting: in.GenesisExisting,
		PerBinary: in.GenesisPerBinary, Fork: in.GenesisFork,
	})
	if err != nil {
		s.mg.fail(m, stepGenesis, err)
		return nil
	}
	plan, err := reconcileUp(ctx, s.mg.d, s.mg.ws.Dir(), s.mg.reuse, gopts)
	if err != nil {
		s.mg.fail(m, stepGenesis, err)
		return nil
	}
	s.mg.note("reuse", plan.describe())
	if plan.Refuse != "" {
		s.mg.failure = fmt.Errorf("chainsetup: chain up: reuse-if-matching refused: %s", plan.Refuse)
		m.SendSelf(reconcileRefused{})
		return nil
	}
	m.SendSelf(reconciled{Kept: len(plan.redo()) == 0})
	return nil
}
