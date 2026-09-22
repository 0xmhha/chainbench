package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// buildingNodeCommand assembles each node's launch argv.
//
// One state, and one assembly site: every node's command line is built here,
// so a flag the composition adds and a flag a launch uses are the same thing.
// A second place that built argv is how a node ends up started with options
// nothing recorded.
type buildingNodeCommand struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (buildingNodeCommand) Name() statemachine.StateName { return nameBuildingNodeCommand }

// step is which of the composition's steps this state runs.
func (buildingNodeCommand) step() string { return stepBuild }

// Enter records the launch overrides and assembles the commands with them.
//
// The scoped sets are recorded most-general-first and the invocation's own
// overrides last, because scope narrowness and the priority line are two
// different orders and they disagreed: a declaration that named a role used to
// beat a flag the operator typed, on the very nodes they typed it for.
func (s *buildingNodeCommand) Enter(_ context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	in := s.mg.request
	detail, err := WithWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (string, error) {
		for _, scope := range SortedScopes(in.LaunchScoped) {
			if rerr := ws.RecordLaunchSet(scope, in.LaunchScoped[scope]); rerr != nil {
				return "", fmt.Errorf("chainsetup: launchopts: %w", rerr)
			}
		}
		if rerr := ws.RecordLaunchCommand(in.LaunchSet); rerr != nil {
			return "", fmt.Errorf("chainsetup: launchopts: %w", rerr)
		}
		return ws.LaunchOpts()
	})
	if err != nil {
		s.mg.fail(m, stepBuild, err)
		return nil
	}
	m.SendSelf(nodeCommandBuilt{Detail: detail})
	return nil
}
