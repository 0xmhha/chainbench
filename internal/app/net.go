package app

import (
	"context"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"

	chainsetupmod "github.com/0xmhha/chainbench/internal/chainsetup"
)

// The net verbs live in the chainsetup module; app wraps them thinly so that
// every surface — CLI, MCP and DSL alike — reaches the feature through this
// layer (architecture-v2 §2, revised 2026-09-05).
//
// Thinly is the whole contract. These wrappers adapt a dependency set and
// forward; they decide nothing. A wrapper that starts judging is how the two
// surfaces above it drift apart while both still appear to go through app.

type (
	ChainNewIn         = verb.ChainNewIn
	ChainNewOut        = verb.ChainNewOut
	ChainStatusIn      = verb.ChainStatusIn
	ChainStatusOut     = verb.ChainStatusOut
	StepOut            = chainsetupmod.StepOut
	ChainKeysIn        = verb.ChainKeysIn
	ChainAllocateIn    = verb.ChainAllocateIn
	ChainGenesisIn     = chainsetupmod.ChainGenesisIn
	ChainConfigIn      = verb.ChainConfigIn
	ChainLaunchOptsIn  = verb.ChainLaunchOptsIn
	ChainLaunchOptsOut = verb.ChainLaunchOptsOut
	ChainProvisionIn   = verb.ChainProvisionIn
	ChainInitIn        = verb.ChainInitIn
	ChainStartIn       = verb.ChainStartIn
	ChainStopIn        = verb.ChainStopIn
	ChainRestartIn     = verb.ChainRestartIn
	ChainResumeIn      = verb.ChainResumeIn
	ChainResumeOut     = verb.ChainResumeOut
	ChainRmIn          = verb.ChainRmIn
	ChainLogsIn        = verb.ChainLogsIn
	ChainLogsOut       = verb.ChainLogsOut
	ChainHealthIn      = verb.ChainHealthIn
	ChainHealthOut     = verb.ChainHealthOut

	ChainVerifyValidatorsIn  = chainsetupmod.ChainVerifyValidatorsIn
	ChainVerifyValidatorsOut = chainsetupmod.ChainVerifyValidatorsOut
	ValidatorCheck           = chainsetupmod.ValidatorCheck

	ChainBaselineIn  = chainsetupmod.ChainBaselineIn
	ChainBaselineOut = chainsetupmod.ChainBaselineOut
	ChainEnodesIn    = verb.ChainEnodesIn
	ChainEnodesOut   = verb.ChainEnodesOut
	// State is a workspace's recorded progress: which steps have run and what
	// each produced. A surface renders it; it is not a use case's input.
	State            = chainsetupmod.State
	UpStage          = chainsetupmod.UpStage
	ChainUpIn        = chainsetupmod.ChainUpIn
	ChainUpOut       = verb.ChainUpOut
	NetworkStatusIn  = verb.NetworkStatusIn
	NetworkStatusOut = verb.NetworkStatusOut
	NetworkStopIn    = verb.NetworkStopIn
	NetworkStopOut   = verb.NetworkStopOut
	NodeStopIn       = verb.NodeStopIn
	NodeStartIn      = verb.NodeStartIn
	NodeStartOut     = verb.NodeStartOut
	NetworkRemoveIn  = verb.NetworkRemoveIn
	NetworkRemoveOut = verb.NetworkRemoveOut
)

const (
	UpDeploy = chainsetupmod.UpDeploy
	UpStart  = chainsetupmod.UpStart
)

// chainsetupDeps adapts this layer's dependency set to the module's.
func (d Deps) chainsetupDeps() chainsetupmod.Deps {
	return chainsetupmod.Deps{Clock: d.Clock, Env: d.Env, Command: d.command(), Report: d.Logf, Driver: d.Driver}
}

func ChainAllocate(ctx context.Context, d Deps, in ChainAllocateIn) (chainsetupmod.StepOut, error) {
	return verb.ChainAllocate(ctx, d.chainsetupDeps(), in)
}

func ChainConfig(ctx context.Context, d Deps, in ChainConfigIn) (chainsetupmod.StepOut, error) {
	return verb.ChainConfig(ctx, d.chainsetupDeps(), in)
}

func ChainGenesis(ctx context.Context, d Deps, in ChainGenesisIn) (chainsetupmod.StepOut, error) {
	return verb.ChainGenesis(ctx, d.chainsetupDeps(), in)
}

func ChainHealth(ctx context.Context, d Deps, in ChainHealthIn) (verb.ChainHealthOut, error) {
	return verb.ChainHealth(ctx, d.chainsetupDeps(), in)
}

// VerifyValidators checks the running chain recognizes exactly the composed
// keys as its validators — the runtime validator verification behind
// `verify --validators`, shared by both surfaces.
func VerifyValidators(ctx context.Context, d Deps, in ChainVerifyValidatorsIn) (ChainVerifyValidatorsOut, error) {
	return chainsetupmod.ChainVerifyValidators(ctx, d.chainsetupDeps(), in)
}

// BaselineCheck compares a composition against its environment's approved
// baseline. It writes nothing — approving is a separate, explicit act.
func BaselineCheck(ctx context.Context, d Deps, in ChainBaselineIn) (ChainBaselineOut, error) {
	return chainsetupmod.ChainBaselineCheck(ctx, d.chainsetupDeps(), in)
}

// BaselineApprove records this composition as the environment's approved
// baseline, the one path that writes one.
func BaselineApprove(ctx context.Context, d Deps, in ChainBaselineIn) (ChainBaselineOut, error) {
	return chainsetupmod.ChainBaselineApprove(ctx, d.chainsetupDeps(), in)
}

func ChainInit(ctx context.Context, d Deps, in ChainInitIn) (chainsetupmod.StepOut, error) {
	return verb.ChainInit(ctx, d.chainsetupDeps(), in)
}

func ChainKeys(ctx context.Context, d Deps, in ChainKeysIn) (chainsetupmod.StepOut, error) {
	return verb.ChainKeys(ctx, d.chainsetupDeps(), in)
}

func ChainLaunchOpts(ctx context.Context, d Deps, in ChainLaunchOptsIn) (verb.ChainLaunchOptsOut, error) {
	return verb.ChainLaunchOpts(ctx, d.chainsetupDeps(), in)
}

func ChainLogs(ctx context.Context, d Deps, in ChainLogsIn) (verb.ChainLogsOut, error) {
	return verb.ChainLogs(ctx, d.chainsetupDeps(), in)
}

func ChainNew(ctx context.Context, d Deps, in ChainNewIn) (verb.ChainNewOut, error) {
	return verb.ChainNew(ctx, d.chainsetupDeps(), in)
}

func ChainProvision(ctx context.Context, d Deps, in ChainProvisionIn) (chainsetupmod.StepOut, error) {
	return verb.ChainProvision(ctx, d.chainsetupDeps(), in)
}

func ChainRestart(ctx context.Context, d Deps, in ChainRestartIn) (chainsetupmod.StepOut, error) {
	return verb.ChainRestart(ctx, d.chainsetupDeps(), in)
}

func ChainRm(ctx context.Context, d Deps, in ChainRmIn) (chainsetupmod.StepOut, error) {
	return verb.ChainRm(ctx, d.chainsetupDeps(), in)
}

func ChainStart(ctx context.Context, d Deps, in ChainStartIn) (chainsetupmod.StepOut, error) {
	return verb.ChainStart(ctx, d.chainsetupDeps(), in)
}

func ChainStatus(ctx context.Context, d Deps, in ChainStatusIn) (verb.ChainStatusOut, error) {
	return verb.ChainStatus(ctx, d.chainsetupDeps(), in)
}

func ChainStop(ctx context.Context, d Deps, in ChainStopIn) (chainsetupmod.StepOut, error) {
	return verb.ChainStop(ctx, d.chainsetupDeps(), in)
}

func ChainUp(ctx context.Context, d Deps, in ChainUpIn) (verb.ChainUpOut, error) {
	return verb.ChainUp(ctx, d.chainsetupDeps(), in)
}

func NetworkRemove(ctx context.Context, d Deps, in NetworkRemoveIn) (verb.NetworkRemoveOut, error) {
	return verb.NetworkRemove(ctx, d.chainsetupDeps(), in)
}

func NetworkStatus(ctx context.Context, d Deps, in NetworkStatusIn) (verb.NetworkStatusOut, error) {
	return verb.NetworkStatus(ctx, d.chainsetupDeps(), in)
}

func NetworkStop(ctx context.Context, d Deps, in NetworkStopIn) (verb.NetworkStopOut, error) {
	return verb.NetworkStop(ctx, d.chainsetupDeps(), in)
}

func NodeStart(ctx context.Context, d Deps, in NodeStartIn) (verb.NodeStartOut, error) {
	return verb.NodeStart(ctx, d.chainsetupDeps(), in)
}

// NodeStop stops one node by index.
func NodeStop(ctx context.Context, d Deps, in NodeStopIn) error {
	return verb.NodeStop(ctx, d.chainsetupDeps(), in)
}

// ChainResume recovers a workspace whose run died: reconcile pids with the
// machine, continue from the first unfinished step, bring back the nodes
// that should be running.
func ChainResume(ctx context.Context, d Deps, in ChainResumeIn) (verb.ChainResumeOut, error) {
	return verb.ChainResume(ctx, d.chainsetupDeps(), in)
}

// ChainEnodes reports each node's enode URL, which is what a surface prints when
// an operator needs to peer something by hand.
func ChainEnodes(ctx context.Context, d Deps, in ChainEnodesIn) (ChainEnodesOut, error) {
	return verb.ChainEnodes(ctx, d.chainsetupDeps(), in)
}

// DefaultWorkspaceDir is where a composition lands when the operator names no
// workspace. It is here rather than in each surface because a CLI run and an
// MCP call that both omit it must land in the same place; two surfaces
// computing their own default is how one of them starts composing somewhere
// the other cannot find.
func DefaultWorkspaceDir(d Deps) (string, error) {
	return chainsetupmod.DefaultWorkspaceDir(d.now)
}
