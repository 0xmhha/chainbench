package app

import (
	"context"

	chainsetupmod "github.com/0xmhha/chainbench/internal/chainsetup"
)

// The net verbs live in the chainsetup module; app wraps them thinly so MCP
// reaches every feature through this layer (architecture-v2 §2). CLI calls
// the module directly and does not pass through here.

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
	UpProvision = chainsetupmod.UpProvision
	UpStart     = chainsetupmod.UpStart
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
