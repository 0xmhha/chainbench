package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
	"os"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// Net step use cases. Each opens the workspace, runs one netcompose step, and
// persists the state — the exact function both the CLI subcommand and the MCP
// tool call. Read-only steps (logs, health) do not save.

// withWorkspace opens the workspace with the injected deps, runs one mutating
// step, and saves the state — whether or not the step succeeded.
//
// Saving on the failure path is the point. A step that launches nodes and then
// fails (or is interrupted) has already changed the world: those processes are
// running and hold ports. Discarding the state because the step returned an
// error discards their pids with it, and pids nobody recorded are orphans —
// `chain stop` finds nothing, the next run fails with "address already in use",
// and there is no record of who left them.
//
// A save failure on the error path is reported alongside the original error
// rather than replacing it: the step's failure is what the caller asked about,
// and losing the record is a second, separate problem.
func withWorkspace(d Deps, dataDir string, fn func(*Workspace) (string, error)) (string, error) {
	ws, err := Open(dataDir, d.Clock)
	if err != nil {
		return "", err
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)

	// One run at a time per workspace. A second run would compose over the
	// first's half-built network and blame the collision on the chain; the
	// refusal names who holds it instead. A lock left by a run that died is
	// taken over — that run is gone, and its wreckage is what the operator is
	// here to clear — but never in silence.
	held, prev, state, lerr := ws.Acquire(d.command())
	if lerr != nil {
		return "", lerr
	}
	defer func() { _ = held.Release() }()
	if state == session.LockStale {
		d.logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up; `chain status` shows what is there", prev.Describe())
	}

	detail, stepErr := fn(ws)
	saveErr := ws.Save()
	switch {
	case stepErr != nil && saveErr != nil:
		return "", fmt.Errorf("%w (and the workspace could not be saved: %v — processes this step started may not be recorded)", stepErr, saveErr)
	case stepErr != nil:
		return "", stepErr
	case saveErr != nil:
		return "", saveErr
	}
	return detail, nil
}

// Placement bounds shared by the composition steps: a BFT network needs at
// least one validator, and a local port band holds this many nodes.
const (
	minValidators = 1
	portBand      = 100
)

// StepOut is the common result of one mutating step: its recorded detail line.
type StepOut struct {
	Detail string
}

// NetKeysIn selects where node identities come from.
type NetKeysIn struct {
	DataDir    string
	Source     string // preset (default) | generate
	Nodes      int
	Validators int
}

// NetKeys ensures the workspace's key set exists and covers the node count.
func NetKeys(ctx context.Context, d Deps, in NetKeysIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Keys(ctx, KeysOpts{
			Source: in.Source, Nodes: in.Nodes, Validators: in.Validators,
		})
	})
	return StepOut{Detail: detail}, err
}

// NetAllocateIn sizes the network.
type NetAllocateIn struct {
	DataDir    string
	Validators int
	Endpoints  int
	// Peering is the peer graph ("mesh" default, "proxied").
	Peering string
	// EndpointSyncMode switches endpoints off full sync ("snap"/"archive") so a
	// re-sync test can exercise that path. Empty leaves every node on full.
	EndpointSyncMode string
	// TopologyPath is a per-node layout YAML (role, sync mode, bootnode). It
	// replaces the counts, which cannot express a per-node choice.
	TopologyPath string
	// Server selects where the nodes are placed and on what ports, from the
	// operator's server set. Its zero value uses the built-in local plan.
	Server resource.ServerRef
	// Topology, when set, is the inline per-node layout (role/sync/bootnode/
	// binary), the DSL's way to declare what --topology gives a file. It wins
	// over Validators/Endpoints.
	Topology *node.Topology
	// Binaries maps per-node binary names to paths (with a per-node topology).
	Binaries map[string]string
}

// NetAllocate builds the node table (roles, paths, deterministic ports).
func NetAllocate(_ context.Context, d Deps, in NetAllocateIn) (StepOut, error) {
	topo := in.Topology
	if topo == nil && in.TopologyPath != "" {
		loaded, err := node.Load(in.TopologyPath)
		if err != nil {
			return StepOut{}, err
		}
		topo = &loaded
	}
	// Allocation is the one moment two runs can hand out the same slot: each
	// derives the set's inventory from the workspaces it can see, and two
	// runs that look before either has saved both see it free. The set's
	// lock is held from the look to the save — a lock, not a second record
	// (the workspaces stay the only record of what is taken).
	setPath := in.Server.SetPath
	if setPath == "" {
		if ws, err := Open(in.DataDir, d.Clock); err == nil {
			setPath = ws.State().ServerSet
		}
	}
	release, err := acquireSetLock(setPath, d)
	if err != nil {
		return StepOut{}, err
	}
	defer release()
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		// A set the workspace already recorded (chain new --server-set) is the
		// default: --docker and its set arrive as a pair, and a later
		// --server-set on this step still wins.
		if in.Server.SetPath == "" {
			in.Server.SetPath = ws.State().ServerSet
		}
		resolved, err := resource.ResolveServer(in.Server, minValidators, portBand)
		if err != nil {
			return "", err
		}
		if resolved.HasTarget {
			if err := ws.Retarget(resolved.Target); err != nil {
				return "", err
			}
		}
		return ws.Allocate(AllocateOpts{
			Validators: in.Validators, Endpoints: in.Endpoints,
			EndpointSyncMode: in.EndpointSyncMode, Topology: topo, Peering: in.Peering,
			Pool: resolved.Pool, SetPath: in.Server.SetPath, Binaries: in.Binaries,
		})
	})
	return StepOut{Detail: detail}, err
}

// NetGenesisIn customizes the built genesis.
type NetGenesisIn struct {
	DataDir string
	ChainID int64
	// Set carries genesis config overrides as key=value on the bare config key,
	// e.g. "bohoBlock=10" to move a fork off genesis.
	Set []string
	// OverlayPath is a JSON overlay file {capabilities, genesis}: the genesis
	// fragment is deep-merged and the capabilities are advertised.
	OverlayPath string
}

// NetGenesis builds the genesis from the key set and writes it to the target.
func NetGenesis(ctx context.Context, d Deps, in NetGenesisIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		opts, err := genesisOpts(in)
		if err != nil {
			return "", err
		}
		return ws.Genesis(ctx, opts)
	})
	return StepOut{Detail: detail}, err
}

// genesisOpts folds the flag-shaped genesis inputs into the step options: the
// key=value overrides and the overlay file's two halves.
func genesisOpts(in NetGenesisIn) (GenesisOpts, error) {
	opts := GenesisOpts{ChainID: in.ChainID}
	for _, kv := range in.Set {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return opts, fmt.Errorf("chainsetup: genesis override expects key=value, got %q", kv)
		}
		if opts.Overrides == nil {
			opts.Overrides = map[string]string{}
		}
		opts.Overrides[k] = v
	}
	if in.OverlayPath == "" {
		return opts, nil
	}
	raw, err := os.ReadFile(in.OverlayPath)
	if err != nil {
		return opts, err
	}
	var overlay struct {
		Capabilities []string        `json:"capabilities"`
		Genesis      json.RawMessage `json:"genesis"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return opts, fmt.Errorf("chainsetup: bad genesis overlay %q: %w", in.OverlayPath, err)
	}
	opts.Overlay = overlay.Genesis
	opts.Capabilities = overlay.Capabilities
	return opts, nil
}

// NetConfigIn identifies the workspace and, optionally, per-node config
// overrides to record before rendering.
type NetConfigIn struct {
	DataDir string
	// Node scopes a Set override to that 1-based node; 0 means every node.
	Node int
	// Set are dot-path "key=value" config-knob overrides to record. Empty
	// renders with whatever overrides the workspace already holds.
	Set []string
	// ScopedSet records overrides for several scopes at once ("all", "node<N>"),
	// the form the up flow and a DSL env pass. It is applied before Set.
	ScopedSet map[string][]string
}

// NetConfig records any per-node overrides, then renders and writes each node's
// TOML config with them applied. Recording and rendering share one step so
// `chain config --node N --set k=v` both persists the override and reflects it.
func NetConfig(ctx context.Context, d Deps, in NetConfigIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.recordConfigSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		if len(in.Set) > 0 {
			scope := "all"
			if in.Node > 0 {
				scope = fmt.Sprintf("node%d", in.Node)
			}
			if err := ws.recordConfigSet(scope, in.Set); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		return ws.Config(ctx)
	})
	return StepOut{Detail: detail}, err
}

// NetLaunchOptsIn customizes the assembled argv.
type NetLaunchOptsIn struct {
	DataDir string
	Set     []string // key=value overrides (bare key for booleans)
}

// NetLaunchOptsOut is the assembled per-node argv table.
type NetLaunchOptsOut struct {
	Detail string
	Nodes  []node.Record
}

// NetLaunchOpts assembles each node's launch argv (the single assembly site)
// and records it, returning the table so the surface can render the commands.
func NetLaunchOpts(_ context.Context, d Deps, in NetLaunchOptsIn) (NetLaunchOptsOut, error) {
	var nodes []node.Record
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		det, err := ws.LaunchOpts(LaunchOptsOpts{Set: in.Set})
		nodes = ws.State().Nodes
		return det, err
	})
	return NetLaunchOptsOut{Detail: detail, Nodes: nodes}, err
}

// NetProvisionIn identifies the workspace.
type NetProvisionIn struct {
	DataDir string
}

// NetProvision verifies the launch inputs are present on the target
// (skip-if-exists semantics: present files are reused, missing ones are named).
func NetProvision(ctx context.Context, d Deps, in NetProvisionIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Provision(ctx)
	})
	return StepOut{Detail: detail}, err
}

// NetInitIn initializes datadirs.
type NetInitIn struct {
	DataDir string
	Binary  string
}

// NetInit runs `<binary> init` for each node's datadir from the built genesis.
func NetInit(ctx context.Context, d Deps, in NetInitIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Init(ctx, in.Binary)
	})
	return StepOut{Detail: detail}, err
}

// NetStartIn launches the composed network.
type NetStartIn struct {
	DataDir string
	Binary  string
}

// NetStart launches every stopped node and records the PIDs.
func NetStart(ctx context.Context, d Deps, in NetStartIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Start(ctx, in.Binary)
	})
	return StepOut{Detail: detail}, err
}

// NetStopIn identifies the workspace.
type NetStopIn struct {
	DataDir string
}

// NetStop terminates every running node by its recorded PID.
func NetStop(ctx context.Context, d Deps, in NetStopIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
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
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
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
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
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

// sortedScopes orders config-override scopes deterministically: "all" first,
// then the node scopes by index, so recording is reproducible regardless of
// map iteration order.
func sortedScopes(m map[string][]string) []string {
	if len(m) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(m))
	for k := range m {
		scopes = append(scopes, k)
	}
	sort.Slice(scopes, func(i, j int) bool {
		if scopes[i] == "all" {
			return true
		}
		if scopes[j] == "all" {
			return false
		}
		return scopes[i] < scopes[j]
	})
	return scopes
}
