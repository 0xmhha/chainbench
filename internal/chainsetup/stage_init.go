package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// initializingDatadirs runs each node's init against the genesis.
//
// One state. Every node initialises the same way from the same document, and a
// node whose datadir already holds a chain is the same case as one that does
// not — the step's own skip-if-initialised rule handles it without the
// composition having to take a different route.
type initializingDatadirs struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (initializingDatadirs) Name() statemachine.StateName { return nameChainInitNodes }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (initializingDatadirs) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventDatadirsInitialized, eventStageFailed}}
}

// step is which of the composition's steps this state runs.
func (initializingDatadirs) step() string { return stepInit }

// Enter initialises every node's datadir and says how many it did.
func (s *initializingDatadirs) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	binary := s.mg.request.Binary
	detail, err := WithWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (string, error) {
		return ws.Init(ctx, binary)
	})
	if err != nil {
		s.mg.fail(m, stepInit, err)
		return nil
	}
	m.SendSelf(datadirsInitialized{Detail: detail})
	return nil
}
