package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// buildingNodeTable decides which node runs where, on which ports.
//
// One state, no choice inside it: the layout may come from counts, a topology
// file, an inline topology or a blueprint, but those are four ways of saying
// the same thing and the allocator resolves them before anything is placed.
// The stages after this one do have choices, and those get a state each.
type buildingNodeTable struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (buildingNodeTable) Name() statemachine.StateName { return nameChainBuildNodeTable }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (buildingNodeTable) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventNodeTableBuilt, eventStageFailed}}
}

// step is which of the composition's steps this state runs.
func (buildingNodeTable) step() string { return stepPlace }

// Enter places the nodes and says how many went where.
func (s *buildingNodeTable) Enter(_ context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	in := s.mg.request
	detail, err := PlaceNodes(s.mg.d, ChainAllocateIn{
		DataDir: s.mg.ws.Dir(),
		BPCount: in.BPCount, ENCount: in.ENCount, PNCount: in.PNCount,
		EndpointSyncMode: in.EndpointSyncMode, Peering: in.Peering,
		TopologyPath: in.TopologyPath, BlueprintPath: in.BlueprintPath,
		Topology: in.Topology, Binaries: in.Binaries, BinaryChains: in.BinaryChains,
		Server:   in.Server,
		AutoSize: in.AutoSize,
	})
	if err != nil {
		s.mg.fail(m, stepPlace, err)
		return nil
	}
	m.SendSelf(nodeTableBuilt{Detail: detail})
	return nil
}
