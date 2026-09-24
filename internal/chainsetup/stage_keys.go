package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The names of the three ways a composition gets its identities.
const (
	nameChainEnsureKeysFromPreset    statemachine.StateName = "CHAIN_ENSURE_KEYS_FROM_PRESET"
	nameChainEnsureKeysGenerate      statemachine.StateName = "CHAIN_ENSURE_KEYS_GENERATE"
	nameChainEnsureKeysFromBlueprint statemachine.StateName = "CHAIN_ENSURE_KEYS_FROM_BLUEPRINT"
)

// ensuringKeys is the first stage with more than one way of doing its work.
//
// Which way is not a detail. A record that says only "keys" cannot answer why
// two compositions of the same request produced different validator addresses,
// and the answer is usually that one took its identities from a preset and the
// other from a node table that named them. So the way is a state, and the
// record keeps it.
//
// The choice is made here rather than by the stage before, because making it
// needs work: a ring that lives on a server has to be fetched before the node
// table can be read against it. Enter prepares and chooses, says which way in a
// message, and Process turns that into the leaf that acts.
type ensuringKeys struct {
	statemachine.Base
	mg *Manager

	// What choosing decided, for the leaf to carry out. Held here rather than
	// on the Manager because it is this stage's, and a leaf reads it from the
	// parent it hangs under.
	src store.KeySource
	n   int

	leaves map[keyWay]*keysLeaf
}

// newEnsuringKeys builds the stage and the three ways it can go.
func newEnsuringKeys(mg *Manager) *ensuringKeys {
	s := &ensuringKeys{mg: mg}
	s.leaves = map[keyWay]*keysLeaf{
		wayPreset:    {parent: s, way: wayPreset, name: nameChainEnsureKeysFromPreset},
		wayGenerated: {parent: s, way: wayGenerated, name: nameChainEnsureKeysGenerate},
		wayDeclared:  {parent: s, way: wayDeclared, name: nameChainEnsureKeysFromBlueprint},
	}
	return s
}

// Name says what this state is called.
func (ensuringKeys) Name() statemachine.StateName { return nameChainEnsureKeys }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (ensuringKeys) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: []statemachine.What{eventKeySourceChosen}, Emits: []statemachine.What{eventKeySourceChosen, eventStageFailed}}
}

// step is which of the composition's steps this state runs.
func (ensuringKeys) step() string { return stepKeys }

// leafStates is the ways this stage can go, in the order the tree shows them.
func (s *ensuringKeys) leafStates() []statemachine.State {
	return []statemachine.State{s.leaves[wayPreset], s.leaves[wayGenerated], s.leaves[wayDeclared]}
}

// Enter gets what choosing needs and chooses.
func (s *ensuringKeys) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	in := s.mg.request
	opts, err := KeysOptsFor(in.BlueprintPath, in.KeysSource, in.KeysNodes, in.KeysValidators)
	if err != nil {
		s.mg.fail(m, stepKeys, err)
		return nil
	}
	type choice struct {
		src store.KeySource
		way keyWay
		n   int
	}
	c, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (choice, error) {
		src, way, n, serr := ws.keySource(ctx, opts)
		return choice{src, way, n}, serr
	})
	if err != nil {
		s.mg.fail(m, stepKeys, err)
		return nil
	}
	s.src, s.n = c.src, c.n
	m.SendSelf(keySourceChosen{Way: c.way})
	return nil
}

// Process turns the choice into the state that carries it out.
func (s *ensuringKeys) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	c, ok := msg.(keySourceChosen)
	if !ok {
		return false, nil
	}
	leaf, ok := s.leaves[c.Way]
	if !ok {
		// Unreachable while keySource returns one of the three, and said out
		// loud rather than assumed: a fourth way added there without a state
		// here would otherwise stop the composition with no explanation.
		s.mg.fail(m, stepKeys, fmt.Errorf("chainsetup: keys: no state for the %q source", c.Way))
		return true, nil
	}
	m.TransitionTo(leaf)
	return true, nil
}

// keysLeaf writes the ring the chosen source describes.
//
// One type for the three ways, because what they do is the same call on three
// different descriptions of where the identities come from — the difference is
// already in the source the parent built. Three types would be three copies of
// one body, distinguished only by a name.
type keysLeaf struct {
	statemachine.Base
	parent *ensuringKeys
	way    keyWay
	name   statemachine.StateName
}

// Name says what this state is called.
func (l *keysLeaf) Name() statemachine.StateName { return l.name }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (l *keysLeaf) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventKeysEnsured, eventStageFailed}}
}

// Enter writes the ring and says what it made.
func (l *keysLeaf) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := l.parent.mg
	mg.recordPath(l)
	detail, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
		return ws.EnsureKeys(ctx, l.parent.src, l.parent.n)
	})
	if err != nil {
		mg.fail(m, stepKeys, err)
		return nil
	}
	m.SendSelf(keysEnsured{Detail: detail})
	return nil
}
