package verb

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// ChainUp composes a whole network in one call by running the step use cases in
// order. It is the step stack's answer to `setup --launch`: the same nine steps
// an operator can run one at a time, driven end to end, so there is one
// bring-up implementation rather than two.
//
// Every step still records itself in the workspace, so a run that fails part
// way leaves an inspectable composition the operator can resume by hand.

// ChainUp runs the composition steps in order and returns what each recorded.
func ChainUp(ctx context.Context, d chainsetup.Deps, in chainsetup.ChainUpIn) (ChainUpOut, error) {
	return composeFrom(ctx, d, in, "")
}

// ChainUpOut reports each step's recorded detail, in order, and the resulting
// network.
type ChainUpOut struct {
	// Steps is one "name: detail" line per step that ran.
	Steps []string
	// Nodes is the composed network. Its PIDs are set only when the run reached
	// UpStart.
	Nodes NetworkStatusOut
}

// upPlan is a validated up request: the stage it runs to and how it treats an
// existing composition, both resolved from defaults and checked once.
type upPlan struct {
	stage chainsetup.UpStage
	mode  resource.ChainMode
}

// planUp validates the request and resolves its defaults. It writes nothing:
// every refusal here happens before the workspace is opened, which is the point
// of doing it in one place up front.
func planUp(in chainsetup.ChainUpIn) (upPlan, error) {
	if in.DataDir == "" {
		return upPlan{}, errors.New("chainsetup: chain up needs a workspace directory")
	}
	stage := in.Stage
	if stage == "" {
		stage = chainsetup.UpStart
	}
	if stage != chainsetup.UpDeploy && stage != chainsetup.UpStart {
		return upPlan{}, fmt.Errorf("chainsetup: unknown stage %q (want %s or %s)", stage, chainsetup.UpDeploy, chainsetup.UpStart)
	}
	if stage == chainsetup.UpStart && in.Binary == "" {
		return upPlan{}, errors.New("chainsetup: chain up --stage=start needs a node binary")
	}
	// Before the request is recorded, not before it is used.
	//
	// The node table's keys are checked again at place, which is where they turn
	// into node records. That is too late for this path: `up` writes the request
	// itself onto the workspace first (it is what a resume composes from), and
	// the request carries the topology whole — so an inline key was already in
	// chain-record.json by the time place refused it. Nothing is written until this
	// returns.
	if err := chainsetup.CheckTopologyKeyRefs(in.Topology); err != nil {
		return upPlan{}, err
	}
	// execution.chain selects how this up treats an existing composition. attach
	// does not compose or launch, so it has no meaning for up; reuse-if-matching
	// reconciles a running network node by node; fresh is the default and
	// composes as it always has.
	mode, err := upChainMode(in)
	if err != nil {
		return upPlan{}, err
	}
	if mode == resource.ChainAttach {
		return upPlan{}, errors.New("chainsetup: chain up: execution.chain=attach does not compose or launch a network — bring the chain up separately and use the attach/run path")
	}
	return upPlan{stage: stage, mode: mode}, nil
}

// composeFrom runs the composition from the named step on (every step when
// from is empty). Steps before it are assumed done — the resume verb decides
// that from the workspace's record.
//
// It validates the request, opens and holds the workspace, and hands the step
// bodies to the machine that walks them. It does not decide the order, which
// stage comes after which, or where a run stops; those are the machine's, and
// this is what starts it.
func composeFrom(ctx context.Context, d chainsetup.Deps, in chainsetup.ChainUpIn, from string) (ChainUpOut, error) {
	up, err := planUp(in)
	if err != nil {
		return ChainUpOut{}, err
	}
	// Before the workspace is opened, so what this request records, compares
	// and launches with is one form of the same reference.
	if err := placeUpRequest(&in); err != nil {
		return ChainUpOut{}, err
	}
	stage, mode := up.stage, up.mode

	lockWS, release, err := holdWorkspace(d, in.DataDir)
	if err != nil {
		return ChainUpOut{}, err
	}
	defer release()

	var out ChainUpOut

	// The machine is given the workspace and the request; every stage does its
	// own work, so nothing of this function's is handed in.
	mgr := newManagerFor(ctx, d, lockWS, stage, mode)
	cerr := mgr.Compose(ctx, in, from)
	// Read before the error is returned: how far a dead run got is the first
	// thing its reader wants.
	out.Steps = append(out.Steps, mgr.Steps()...)
	if cerr != nil {
		return out, cerr
	}

	nodes, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: in.DataDir})
	if err != nil {
		return out, err
	}
	out.Nodes = nodes
	return out, nil
}

// holdWorkspace opens the workspace and holds its lock until release is
// called.
//
// A composition holds the workspace for its whole run. Each step it calls
// takes the lock too, but a run cannot conflict with itself (session Acquire
// is re-entrant per process); what this closes is the gap between steps, where
// another run used to slip in and compose over a half-built network.
func holdWorkspace(d chainsetup.Deps, dir string) (*chainsetup.Workspace, func(), error) {
	ws, err := chainsetup.Open(dir, d.Clock)
	if err != nil {
		return nil, nil, err
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)
	held, prev, lockState, err := ws.Acquire(d.Owner())
	if err != nil {
		return nil, nil, err
	}
	if lockState == session.LockStale {
		d.Logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up", prev.Describe())
	}
	return ws, func() { _ = held.Release() }, nil
}

// newManagerFor is the machine for a composition in the given execution mode.
//
// reuse-if-matching reconciles a running network node by node. Its baseline —
// what each node hashed to and whether it answers — must be captured now,
// before the compose steps re-run and reset the node table. A first up over an
// empty workspace yields an empty snapshot, which composes everything.
func newManagerFor(ctx context.Context, d chainsetup.Deps, ws *chainsetup.Workspace, stage chainsetup.UpStage, mode resource.ChainMode) *chainsetup.Manager {
	mgr := chainsetup.NewManager(d, ws)
	if mode == resource.ChainReuseIfMatching && stage == chainsetup.UpStart {
		mgr.ReuseFrom(ws.SnapshotForReuse(ctx))
	}
	return mgr
}

// placeUpRequest places the request's binary references through the environment
// file it names, if it names one.
func placeUpRequest(in *chainsetup.ChainUpIn) error {
	if in.WorkspaceConfigPath == "" {
		return chainsetup.PlaceRequest(in, nil)
	}
	wc, err := resource.LoadWorkspaceConfig(in.WorkspaceConfigPath)
	if err != nil {
		return fmt.Errorf("chainsetup: chain up: %w", err)
	}
	return chainsetup.PlaceRequest(in, &wc)
}

// upChainMode reads how this up should treat an existing composition from the
// workspace-config's execution.chain. No config, or an empty value, is fresh —
// the default that composes as it always has.
func upChainMode(in chainsetup.ChainUpIn) (resource.ChainMode, error) {
	if in.WorkspaceConfigPath == "" {
		return resource.ChainFresh, nil
	}
	wc, err := resource.LoadWorkspaceConfig(in.WorkspaceConfigPath)
	if err != nil {
		return "", fmt.Errorf("chainsetup: chain up: %w", err)
	}
	if wc.Execution.Chain == "" {
		return resource.ChainFresh, nil
	}
	return wc.Execution.Chain, nil
}
