package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The two ways the launch inputs end up where a node will look for them.
const (
	nameInputsVerifiedLocal statemachine.StateName = "InputsVerifiedLocal"
	nameInputsShippedRemote statemachine.StateName = "InputsShippedRemote"
)

// deployingInputs puts every file a node needs where that node will look.
//
// The branch here stands after the work, not before it. Which way a deploy went
// is whether anything had to be sent to another machine, and that is counted by
// doing it — a local target ships nothing because its key set already is the
// place the config points at. So Enter does the work, the result says how many
// files went, and Process turns that into the state that names it.
type deployingInputs struct {
	statemachine.Base
	mg *Manager

	verifiedLocal *deployOutcome
	shippedRemote *deployOutcome
}

// newDeployingInputs builds the stage and the two results it can have.
func newDeployingInputs(mg *Manager) *deployingInputs {
	s := &deployingInputs{mg: mg}
	s.verifiedLocal = &deployOutcome{parent: s, name: nameInputsVerifiedLocal}
	s.shippedRemote = &deployOutcome{parent: s, name: nameInputsShippedRemote}
	return s
}

// Name says what this state is called.
func (deployingInputs) Name() statemachine.StateName { return nameDeployingInputs }

// step is which of the composition's steps this state runs.
func (deployingInputs) step() string { return stepDeploy }

// leafStates is the results this stage can have, in the order the tree shows.
func (s *deployingInputs) leafStates() []statemachine.State {
	return []statemachine.State{s.verifiedLocal, s.shippedRemote}
}

// Enter verifies what is present and ships what is not.
func (s *deployingInputs) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	out, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (StepOut, error) {
		return ws.Provision(ctx)
	})
	if err != nil {
		s.mg.fail(m, stepDeploy, err)
		return nil
	}
	m.SendSelf(inputsPresent{Detail: out.Detail, Shipped: out.Shipped})
	return nil
}

// Process turns what the deploy counted into the state that names it.
func (s *deployingInputs) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	e, ok := msg.(inputsPresent)
	if !ok {
		return false, nil
	}
	outcome := s.verifiedLocal
	if e.Shipped > 0 {
		outcome = s.shippedRemote
	}
	outcome.detail = e.Detail
	m.TransitionTo(outcome)
	return true, nil
}

// deployOutcome is which way the deploy went.
//
// It does no work. Being entered is the whole of it: the record keeps the path,
// and the path is the answer to "did this composition ship anything, or was
// everything already where it needed to be".
type deployOutcome struct {
	statemachine.Base
	parent *deployingInputs
	name   statemachine.StateName
	detail string
}

// Name says what this state is called.
func (o *deployOutcome) Name() statemachine.StateName { return o.name }

// Enter records where the composition is and reports the stage finished.
func (o *deployOutcome) Enter(_ context.Context, m *statemachine.Machine) error {
	o.parent.mg.recordPath(o)
	m.SendSelf(inputsDeployed{Detail: o.detail})
	return nil
}
