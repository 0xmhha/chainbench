package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/hardfork"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// What a composed network can be asked to do.
const (
	nameStopping     statemachine.StateName = "Stopping"
	nameRestarting   statemachine.StateName = "Restarting"
	nameSwapping     statemachine.StateName = "Swapping"
	nameHardforking  statemachine.StateName = "Hardforking"
	nameCrossingFork statemachine.StateName = "CrossingFork"
	nameRemoving     statemachine.StateName = "Removing"
)

// operation is one thing a composed network can be asked to do.
//
// Each carries its own arguments rather than a message doing it: a swap names a
// node and a binary, a hardfork carries a plan, and a union of all of them
// would be a struct where most fields are empty most of the time. The message
// says which operation; the state that will run it already holds what it needs.
type operation interface {
	statemachine.State
	// verb is the name verbNeeds declares this operation's conditions under.
	verb() string
}

// stopping takes every running node down.
type stopping struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (stopping) Name() statemachine.StateName { return nameStopping }

// verb is the name this operation's conditions are declared under.
func (stopping) verb() string { return "Stop" }

// Enter stops the nodes and says how many went down.
func (s *stopping) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.operated(m, s, func(ws *Workspace) (string, error) { return ws.Stop(ctx) })
	return nil
}

// removing takes the composed network's data away.
type removing struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (removing) Name() statemachine.StateName { return nameRemoving }

// verb is the name this operation's conditions are declared under.
func (removing) verb() string { return "Rm" }

// Enter removes what the composition left behind.
func (s *removing) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.operated(m, s, func(ws *Workspace) (string, error) { return ws.Rm(ctx) })
	return nil
}

// restarting bounces one node.
type restarting struct {
	statemachine.Base
	mg    *Manager
	index int
}

// Name says what this state is called.
func (restarting) Name() statemachine.StateName { return nameRestarting }

// verb is the name this operation's conditions are declared under.
func (restarting) verb() string { return "Restart" }

// Enter restarts the node this operation named.
func (s *restarting) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.operated(m, s, func(ws *Workspace) (string, error) { return ws.Restart(ctx, s.index) })
	return nil
}

// swapping replaces one node's binary.
type swapping struct {
	statemachine.Base
	mg   *Manager
	opts SwapNodeOpts
}

// Name says what this state is called.
func (swapping) Name() statemachine.StateName { return nameSwapping }

// verb is the name this operation's conditions are declared under.
func (swapping) verb() string { return "SwapNode" }

// Enter swaps the node this operation named.
func (s *swapping) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.operated(m, s, func(ws *Workspace) (string, error) { return ws.SwapNode(ctx, s.opts) })
	return nil
}

// hardforking swaps the network's binary at a fork block, keeping node data.
type hardforking struct {
	statemachine.Base
	mg     *Manager
	plan   hardfork.SwapPlan
	binary string

	// nodes is what the swap produced, for the caller that asked for it.
	nodes node.NodeSet
}

// Name says what this state is called.
func (hardforking) Name() statemachine.StateName { return nameHardforking }

// verb is the name this operation's conditions are declared under.
func (hardforking) verb() string { return "Hardfork" }

// Enter runs the swap.
func (s *hardforking) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.operated(m, s, func(ws *Workspace) (string, error) {
		ns, err := ws.Hardfork(ctx, s.plan, s.binary)
		if err != nil {
			return "", err
		}
		s.nodes = ns
		return fmt.Sprintf("%d node(s) swapped", len(ns.Nodes)), nil
	})
	return nil
}
