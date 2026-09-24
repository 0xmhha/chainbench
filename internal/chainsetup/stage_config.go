package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// buildingNodeConfig writes each node's config file.
//
// One state: the overrides may come from several scopes, but they are applied
// in one order onto one render, and there is no branch a reader of a record
// would want back.
type buildingNodeConfig struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (buildingNodeConfig) Name() statemachine.StateName { return nameBuildingNodeConfig }

// step is which of the composition's steps this state runs.
func (buildingNodeConfig) step() string { return stepConfig }

// Enter records the overrides and renders the configs with them applied.
//
// Recording and rendering are one step so that a config a run asked for and a
// config on disk cannot disagree: an override written but not applied is a
// knob an operator set and a node never saw.
func (s *buildingNodeConfig) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	set := s.mg.request.ConfigSet
	detail, err := WithWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (string, error) {
		for _, scope := range SortedScopes(set) {
			if rerr := ws.RecordConfigSet(scope, set[scope]); rerr != nil {
				return "", fmt.Errorf("chainsetup: config: %w", rerr)
			}
		}
		return ws.Config(ctx)
	})
	if err != nil {
		s.mg.fail(m, stepConfig, err)
		return nil
	}
	m.SendSelf(nodeConfigBuilt{Detail: detail})
	return nil
}
