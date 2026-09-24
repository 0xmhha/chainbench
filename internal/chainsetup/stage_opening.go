package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// openingWorkspace is the composition's first stage: the workspace learns which
// chain it is for, and keeps the request that asked for it.
//
// One state, not a choice between several. The stage says what it did in its
// own message rather than the adapter's generic one, which is what lets the
// parent's report name it — and what will let a later reader of a record see
// which way a stage with a choice went.
type openingWorkspace struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (openingWorkspace) Name() statemachine.StateName { return nameChainOpenWorkspace }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (openingWorkspace) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventWorkspaceOpened, eventStageFailed}}
}

// step is which of the composition's steps this state runs.
func (openingWorkspace) step() string { return stepNew }

// Enter records the chain and the request, and says what it found.
//
// Both writes happen on one workspace and are saved once. They used to be two
// opens and two saves — the verb wrote the chain, then a second pass wrote the
// request — and a run that died between them left a workspace naming a chain it
// had no request for, which is a composition a resume cannot continue.
func (s *openingWorkspace) Enter(_ context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	in := s.mg.request
	detail, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (string, error) {
		detail, nerr := ws.New(NewOpts{
			Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir, Target: in.Target,
			ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath, Docker: in.Docker,
			WorkspaceConfigPath: in.WorkspaceConfigPath,
			// The set a `chain new --server-set` named. An empty one is ignored
			// by New, so an up that names its set at the placement step instead
			// does not record "no server set" over it.
			ServerSet: in.Server.SetPath,
		})
		if nerr != nil {
			return "", nerr
		}
		// The request is the one fact of a composition otherwise nowhere on
		// disk; it is what a resume composes from.
		return detail, ws.RecordRequest(in)
	})
	if err != nil {
		s.mg.fail(m, stepNew, err)
		return nil
	}
	m.SendSelf(workspaceOpened{Detail: detail})
	return nil
}
