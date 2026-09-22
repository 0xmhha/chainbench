package verb

import (
	"context"
	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// The lifecycle verbs, as a surface calls them: provision, init, start, stop,
// restart, rm, logs, health.
//
// Each is a thin wrapper — take the workspace, call the one step, return what
// it said. They are thin on purpose: a surface that needs more than this is
// asking for something the step should be doing.

type ChainProvisionIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
}

// ChainProvision verifies the launch inputs are present on the target
// (skip-if-exists semantics: present files are reused, missing ones are named).
func ChainProvision(ctx context.Context, d chainsetup.Deps, in ChainProvisionIn) (chainsetup.StepOut, error) {
	return step(ctx, d, in.DataDir, "deploy", chainsetup.ChainUpIn{})
}

// ChainInitIn initializes datadirs.
type ChainInitIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Binary  string `cb:"binary" help:"node binary path (default: the workspace's)"`
}

// ChainInit runs `<binary> init` for each node's datadir from the built genesis.
func ChainInit(ctx context.Context, d chainsetup.Deps, in ChainInitIn) (chainsetup.StepOut, error) {
	return step(ctx, d, in.DataDir, "init", chainsetup.ChainUpIn{Binary: in.Binary})
}

// ChainStartIn launches the composed network.
type ChainStartIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Binary  string `cb:"binary" help:"node binary path (default: the workspace's)"`
}

// ChainStart launches every stopped node and records the PIDs.
func ChainStart(ctx context.Context, d chainsetup.Deps, in ChainStartIn) (chainsetup.StepOut, error) {
	return step(ctx, d, in.DataDir, "start", chainsetup.ChainUpIn{Binary: in.Binary})
}

// ChainStopIn identifies the workspace.
type ChainStopIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
}

// ChainStop terminates every running node by its recorded PID.
func ChainStop(ctx context.Context, d chainsetup.Deps, in ChainStopIn) (chainsetup.StepOut, error) {
	detail, err := chainsetup.WithWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (string, error) {
		return ws.Stop(ctx)
	})
	return chainsetup.StepOut{Detail: detail}, err
}

// ChainRestartIn bounces one node.
type ChainRestartIn struct {
	DataDir string
	Node    int
}

// ChainRestart stops and relaunches one node with its recorded arming.
func ChainRestart(ctx context.Context, d chainsetup.Deps, in ChainRestartIn) (chainsetup.StepOut, error) {
	detail, err := chainsetup.WithWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (string, error) {
		return ws.Restart(ctx, in.Node)
	})
	return chainsetup.StepOut{Detail: detail}, err
}

// ChainRmIn identifies the workspace.
type ChainRmIn struct {
	DataDir string
}

// ChainRm removes the composed data plane (stopped nodes only).
func ChainRm(ctx context.Context, d chainsetup.Deps, in ChainRmIn) (chainsetup.StepOut, error) {
	detail, err := chainsetup.WithWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (string, error) {
		return ws.Rm(ctx)
	})
	return chainsetup.StepOut{Detail: detail}, err
}

// ChainLogsIn selects one node's log tail.
type ChainLogsIn struct {
	DataDir string
	Node    int
	Lines   int
}

// ChainLogsOut is the requested log tail.
type ChainLogsOut struct {
	Text string
}

// ChainLogs returns the last N lines of one node's log. Read-only.
func ChainLogs(ctx context.Context, d chainsetup.Deps, in ChainLogsIn) (ChainLogsOut, error) {
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return ChainLogsOut{}, err
	}
	ws.SetEnv(d.Env)
	text, err := ws.Logs(ctx, in.Node, in.Lines)
	return ChainLogsOut{Text: text}, err
}

// ChainHealthIn identifies the workspace.
type ChainHealthIn struct {
	DataDir string
}

// ChainHealthOut is the per-node probe table.
type ChainHealthOut struct {
	Nodes []chainsetup.NodeHealth
}

// ChainHealth probes every node's HTTP RPC for its latest block. Read-only.
func ChainHealth(ctx context.Context, d chainsetup.Deps, in ChainHealthIn) (ChainHealthOut, error) {
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return ChainHealthOut{}, err
	}
	ws.SetEnv(d.Env)
	nodes, err := ws.Health(ctx)
	return ChainHealthOut{Nodes: nodes}, err
}
