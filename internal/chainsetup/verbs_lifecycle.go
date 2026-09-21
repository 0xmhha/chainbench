package chainsetup

import (
	"context"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// The lifecycle verbs, as a surface calls them: provision, init, start, stop,
// restart, rm, logs, health.
//
// Each is a thin wrapper — take the workspace, call the one step, return what
// it said. They are thin on purpose: a surface that needs more than this is
// asking for something the step should be doing.

type NetProvisionIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
}

// NetProvision verifies the launch inputs are present on the target
// (skip-if-exists semantics: present files are reused, missing ones are named).
func NetProvision(ctx context.Context, d Deps, in NetProvisionIn) (StepOut, error) {
	return InWorkspace(d, in.DataDir, func(ws *Workspace) (StepOut, error) {
		return ws.Provision(ctx)
	})
}

// NetInitIn initializes datadirs.
type NetInitIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Binary  string `cb:"binary" help:"node binary path (default: the workspace's)"`
}

// NetInit runs `<binary> init` for each node's datadir from the built genesis.
func NetInit(ctx context.Context, d Deps, in NetInitIn) (StepOut, error) {
	detail, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Init(ctx, in.Binary)
	})
	return StepOut{Detail: detail}, err
}

// NetStartIn launches the composed network.
type NetStartIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Binary  string `cb:"binary" help:"node binary path (default: the workspace's)"`
}

// NetStart launches every stopped node and records the PIDs.
func NetStart(ctx context.Context, d Deps, in NetStartIn) (StepOut, error) {
	return InWorkspace(d, in.DataDir, func(ws *Workspace) (StepOut, error) {
		return ws.Start(ctx, in.Binary)
	})
}

// NetStopIn identifies the workspace.
type NetStopIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
}

// NetStop terminates every running node by its recorded PID.
func NetStop(ctx context.Context, d Deps, in NetStopIn) (StepOut, error) {
	detail, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Stop(ctx)
	})
	return StepOut{Detail: detail}, err
}

// NetRestartIn bounces one node.
type NetRestartIn struct {
	DataDir string
	Node    int
}

// NetRestart stops and relaunches one node with its recorded arming.
func NetRestart(ctx context.Context, d Deps, in NetRestartIn) (StepOut, error) {
	detail, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Restart(ctx, in.Node)
	})
	return StepOut{Detail: detail}, err
}

// NetRmIn identifies the workspace.
type NetRmIn struct {
	DataDir string
}

// NetRm removes the composed data plane (stopped nodes only).
func NetRm(ctx context.Context, d Deps, in NetRmIn) (StepOut, error) {
	detail, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Rm(ctx)
	})
	return StepOut{Detail: detail}, err
}

// NetLogsIn selects one node's log tail.
type NetLogsIn struct {
	DataDir string
	Node    int
	Lines   int
}

// NetLogsOut is the requested log tail.
type NetLogsOut struct {
	Text string
}

// NetLogs returns the last N lines of one node's log. Read-only.
func NetLogs(ctx context.Context, d Deps, in NetLogsIn) (NetLogsOut, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetLogsOut{}, err
	}
	ws.SetEnv(d.Env)
	text, err := ws.Logs(ctx, in.Node, in.Lines)
	return NetLogsOut{Text: text}, err
}

// NetHealthIn identifies the workspace.
type NetHealthIn struct {
	DataDir string
}

// NetHealthOut is the per-node probe table.
type NetHealthOut struct {
	Nodes []NodeHealth
}

// NetHealth probes every node's HTTP RPC for its latest block. Read-only.
func NetHealth(ctx context.Context, d Deps, in NetHealthIn) (NetHealthOut, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetHealthOut{}, err
	}
	ws.SetEnv(d.Env)
	nodes, err := ws.Health(ctx)
	return NetHealthOut{Nodes: nodes}, err
}

// sortedScopes orders config-override scopes deterministically, most general
// first, so recording is reproducible regardless of map iteration order. Ties
// within a rank are broken by name for the same reason.
func sortedScopes(m map[string][]string) []string {
	if len(m) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(m))
	for k := range m {
		scopes = append(scopes, k)
	}
	sort.Slice(scopes, func(i, j int) bool {
		ri, rj := node.ScopeRank(scopes[i]), node.ScopeRank(scopes[j])
		if ri != rj {
			return ri < rj
		}
		return scopes[i] < scopes[j]
	})
	return scopes
}
