package app

import (
	"context"

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
	NetNewIn         = chainsetupmod.NetNewIn
	NetNewOut        = chainsetupmod.NetNewOut
	NetStatusIn      = chainsetupmod.NetStatusIn
	NetStatusOut     = chainsetupmod.NetStatusOut
	StepOut          = chainsetupmod.StepOut
	NetKeysIn        = chainsetupmod.NetKeysIn
	NetAllocateIn    = chainsetupmod.NetAllocateIn
	NetGenesisIn     = chainsetupmod.NetGenesisIn
	NetConfigIn      = chainsetupmod.NetConfigIn
	NetLaunchOptsIn  = chainsetupmod.NetLaunchOptsIn
	NetLaunchOptsOut = chainsetupmod.NetLaunchOptsOut
	NetProvisionIn   = chainsetupmod.NetProvisionIn
	NetInitIn        = chainsetupmod.NetInitIn
	NetStartIn       = chainsetupmod.NetStartIn
	NetStopIn        = chainsetupmod.NetStopIn
	NetRestartIn     = chainsetupmod.NetRestartIn
	NetResumeIn      = chainsetupmod.NetResumeIn
	NetResumeOut     = chainsetupmod.NetResumeOut
	NetRmIn          = chainsetupmod.NetRmIn
	NetLogsIn        = chainsetupmod.NetLogsIn
	NetLogsOut       = chainsetupmod.NetLogsOut
	NetHealthIn      = chainsetupmod.NetHealthIn
	NetHealthOut     = chainsetupmod.NetHealthOut

	NetVerifyValidatorsIn  = chainsetupmod.NetVerifyValidatorsIn
	NetVerifyValidatorsOut = chainsetupmod.NetVerifyValidatorsOut
	ValidatorCheck         = chainsetupmod.ValidatorCheck

	NetBaselineIn  = chainsetupmod.NetBaselineIn
	NetBaselineOut = chainsetupmod.NetBaselineOut
	NetEnodesIn    = chainsetupmod.NetEnodesIn
	NetEnodesOut   = chainsetupmod.NetEnodesOut
	// State is a workspace's recorded progress: which steps have run and what
	// each produced. A surface renders it; it is not a use case's input.
	State            = chainsetupmod.State
	UpStage          = chainsetupmod.UpStage
	NetUpIn          = chainsetupmod.NetUpIn
	NetUpOut         = chainsetupmod.NetUpOut
	NetworkStatusIn  = chainsetupmod.NetworkStatusIn
	NetworkStatusOut = chainsetupmod.NetworkStatusOut
	NetworkStopIn    = chainsetupmod.NetworkStopIn
	NetworkStopOut   = chainsetupmod.NetworkStopOut
	NodeStopIn       = chainsetupmod.NodeStopIn
	NodeStartIn      = chainsetupmod.NodeStartIn
	NodeStartOut     = chainsetupmod.NodeStartOut
	NetworkRemoveIn  = chainsetupmod.NetworkRemoveIn
	NetworkRemoveOut = chainsetupmod.NetworkRemoveOut
)

const (
	UpDeploy = chainsetupmod.UpDeploy
	UpStart  = chainsetupmod.UpStart
)

// chainsetupDeps adapts this layer's dependency set to the module's.
func (d Deps) chainsetupDeps() chainsetupmod.Deps {
	return chainsetupmod.Deps{Clock: d.Clock, Env: d.Env, Command: d.command(), Report: d.Logf, Driver: d.Driver}
}

func NetAllocate(ctx context.Context, d Deps, in NetAllocateIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetAllocate(ctx, d.chainsetupDeps(), in)
}

func NetConfig(ctx context.Context, d Deps, in NetConfigIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetConfig(ctx, d.chainsetupDeps(), in)
}

func NetGenesis(ctx context.Context, d Deps, in NetGenesisIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetGenesis(ctx, d.chainsetupDeps(), in)
}

func NetHealth(ctx context.Context, d Deps, in NetHealthIn) (chainsetupmod.NetHealthOut, error) {
	return chainsetupmod.NetHealth(ctx, d.chainsetupDeps(), in)
}

// VerifyValidators checks the running chain recognizes exactly the composed
// keys as its validators — the runtime validator verification behind
// `verify --validators`, shared by both surfaces.
func VerifyValidators(ctx context.Context, d Deps, in NetVerifyValidatorsIn) (NetVerifyValidatorsOut, error) {
	return chainsetupmod.NetVerifyValidators(ctx, d.chainsetupDeps(), in)
}

// BaselineCheck compares a composition against its environment's approved
// baseline. It writes nothing — approving is a separate, explicit act.
func BaselineCheck(ctx context.Context, d Deps, in NetBaselineIn) (NetBaselineOut, error) {
	return chainsetupmod.NetBaselineCheck(ctx, d.chainsetupDeps(), in)
}

// BaselineApprove records this composition as the environment's approved
// baseline, the one path that writes one.
func BaselineApprove(ctx context.Context, d Deps, in NetBaselineIn) (NetBaselineOut, error) {
	return chainsetupmod.NetBaselineApprove(ctx, d.chainsetupDeps(), in)
}

func NetInit(ctx context.Context, d Deps, in NetInitIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetInit(ctx, d.chainsetupDeps(), in)
}

func NetKeys(ctx context.Context, d Deps, in NetKeysIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetKeys(ctx, d.chainsetupDeps(), in)
}

func NetLaunchOpts(ctx context.Context, d Deps, in NetLaunchOptsIn) (chainsetupmod.NetLaunchOptsOut, error) {
	return chainsetupmod.NetLaunchOpts(ctx, d.chainsetupDeps(), in)
}

func NetLogs(ctx context.Context, d Deps, in NetLogsIn) (chainsetupmod.NetLogsOut, error) {
	return chainsetupmod.NetLogs(ctx, d.chainsetupDeps(), in)
}

func NetNew(ctx context.Context, d Deps, in NetNewIn) (chainsetupmod.NetNewOut, error) {
	return chainsetupmod.NetNew(ctx, d.chainsetupDeps(), in)
}

func NetProvision(ctx context.Context, d Deps, in NetProvisionIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetProvision(ctx, d.chainsetupDeps(), in)
}

func NetRestart(ctx context.Context, d Deps, in NetRestartIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetRestart(ctx, d.chainsetupDeps(), in)
}

func NetRm(ctx context.Context, d Deps, in NetRmIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetRm(ctx, d.chainsetupDeps(), in)
}

func NetStart(ctx context.Context, d Deps, in NetStartIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetStart(ctx, d.chainsetupDeps(), in)
}

func NetStatus(ctx context.Context, d Deps, in NetStatusIn) (chainsetupmod.NetStatusOut, error) {
	return chainsetupmod.NetStatus(ctx, d.chainsetupDeps(), in)
}

func NetStop(ctx context.Context, d Deps, in NetStopIn) (chainsetupmod.StepOut, error) {
	return chainsetupmod.NetStop(ctx, d.chainsetupDeps(), in)
}

func NetUp(ctx context.Context, d Deps, in NetUpIn) (chainsetupmod.NetUpOut, error) {
	return chainsetupmod.NetUp(ctx, d.chainsetupDeps(), in)
}

func NetworkRemove(ctx context.Context, d Deps, in NetworkRemoveIn) (chainsetupmod.NetworkRemoveOut, error) {
	return chainsetupmod.NetworkRemove(ctx, d.chainsetupDeps(), in)
}

func NetworkStatus(ctx context.Context, d Deps, in NetworkStatusIn) (chainsetupmod.NetworkStatusOut, error) {
	return chainsetupmod.NetworkStatus(ctx, d.chainsetupDeps(), in)
}

func NetworkStop(ctx context.Context, d Deps, in NetworkStopIn) (chainsetupmod.NetworkStopOut, error) {
	return chainsetupmod.NetworkStop(ctx, d.chainsetupDeps(), in)
}

func NodeStart(ctx context.Context, d Deps, in NodeStartIn) (chainsetupmod.NodeStartOut, error) {
	return chainsetupmod.NodeStart(ctx, d.chainsetupDeps(), in)
}

// NodeStop stops one node by index.
func NodeStop(ctx context.Context, d Deps, in NodeStopIn) error {
	return chainsetupmod.NodeStop(ctx, d.chainsetupDeps(), in)
}

// NetResume recovers a workspace whose run died: reconcile pids with the
// machine, continue from the first unfinished step, bring back the nodes
// that should be running.
func NetResume(ctx context.Context, d Deps, in NetResumeIn) (chainsetupmod.NetResumeOut, error) {
	return chainsetupmod.NetResume(ctx, d.chainsetupDeps(), in)
}

// NetEnodes reports each node's enode URL, which is what a surface prints when
// an operator needs to peer something by hand.
func NetEnodes(ctx context.Context, d Deps, in NetEnodesIn) (NetEnodesOut, error) {
	return chainsetupmod.NetEnodes(ctx, d.chainsetupDeps(), in)
}

// DefaultWorkspaceDir is where a composition lands when the operator names no
// workspace. It is here rather than in each surface because a CLI run and an
// MCP call that both omit it must land in the same place; two surfaces
// computing their own default is how one of them starts composing somewhere
// the other cannot find.
func DefaultWorkspaceDir(d Deps) (string, error) {
	return chainsetupmod.DefaultWorkspaceDir(d.now)
}
