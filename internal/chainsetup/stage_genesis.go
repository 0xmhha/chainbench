package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The two ways a composition arrives at its genesis.
const (
	nameChainBuildGenesisFromTemplate statemachine.StateName = "CHAIN_BUILD_GENESIS_FROM_TEMPLATE"
	nameChainBuildGenesisFromExisting statemachine.StateName = "CHAIN_BUILD_GENESIS_FROM_EXISTING"
)

// buildingGenesis writes the document every node initialises from.
//
// Two ways, and the difference matters to a reader of a record: a genesis built
// from the family's template is one this run decided, and a genesis taken from
// an existing file is one somebody else decided and this run is bound to. A
// chain that will not start looks the same either way until you know which.
//
// Unlike the keys stage, choosing here needs nothing but the request. It is
// still done in this stage's Enter rather than in the stage before, so that
// every stage with more than one way reads the same: prepare and choose on the
// way in, say which way, and let Process turn that into the leaf.
type buildingGenesis struct {
	statemachine.Base
	mg *Manager

	// opts is the request read once, for the leaf to carry out.
	opts GenesisOpts

	fromTemplate *genesisLeaf
	fromExisting *genesisLeaf
}

// newBuildingGenesis builds the stage and its two ways.
func newBuildingGenesis(mg *Manager) *buildingGenesis {
	s := &buildingGenesis{mg: mg}
	s.fromTemplate = &genesisLeaf{parent: s, name: nameChainBuildGenesisFromTemplate}
	s.fromExisting = &genesisLeaf{parent: s, name: nameChainBuildGenesisFromExisting}
	return s
}

// Name says what this state is called.
func (buildingGenesis) Name() statemachine.StateName { return nameChainBuildGenesis }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (buildingGenesis) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: []statemachine.What{eventGenesisWayChosen}, Emits: []statemachine.What{eventGenesisWayChosen, eventStageFailed}}
}

// step is which of the composition's steps this state runs.
func (buildingGenesis) step() string { return stepGenesis }

// leafStates is the ways this stage can go, in the order the tree shows them.
func (s *buildingGenesis) leafStates() []statemachine.State {
	return []statemachine.State{s.fromTemplate, s.fromExisting}
}

// Enter reads the request and chooses.
//
// The request is read before any workspace is opened: one that contradicts
// itself needs no workspace, and refusing here keeps the lock and the state out
// of a request that was never going to be carried out.
func (s *buildingGenesis) Enter(_ context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	in := s.mg.request
	opts, err := GenesisOptsFor(ChainGenesisIn{
		DataDir: s.mg.ws.Dir(), ChainID: in.ChainID, Set: in.GenesisSet,
		OverlayPath: in.OverlayPath, GenesisExisting: in.GenesisExisting,
		PerBinary: in.GenesisPerBinary, Fork: in.GenesisFork,
	})
	if err != nil {
		s.mg.fail(m, stepGenesis, err)
		return nil
	}
	s.opts = opts
	m.SendSelf(genesisWayChosen{FromExisting: opts.Existing != ""})
	return nil
}

// Process turns the choice into the state that carries it out.
func (s *buildingGenesis) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	c, ok := msg.(genesisWayChosen)
	if !ok {
		return false, nil
	}
	if c.FromExisting {
		m.TransitionTo(s.fromExisting)
		return true, nil
	}
	m.TransitionTo(s.fromTemplate)
	return true, nil
}

// genesisLeaf builds the genesis the chosen way.
//
// One type for both, as with the key sources: what they do is the same call on
// options that already say where the document comes from. The state's name is
// the difference, and the name is the thing a record needed.
type genesisLeaf struct {
	statemachine.Base
	parent *buildingGenesis
	name   statemachine.StateName
}

// Name says what this state is called.
func (l *genesisLeaf) Name() statemachine.StateName { return l.name }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (l *genesisLeaf) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventGenesisBuilt, eventStageFailed}}
}

// Enter writes the genesis and says what it wrote.
func (l *genesisLeaf) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := l.parent.mg
	mg.recordPath(l)
	out, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (StepOut, error) {
		return ws.Genesis(ctx, l.parent.opts)
	})
	if err != nil {
		mg.fail(m, stepGenesis, err)
		return nil
	}
	m.SendSelf(genesisBuilt{Detail: out.Detail})
	return nil
}
