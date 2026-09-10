package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// NetUp composes a whole network in one call by running the step use cases in
// order. It is the step stack's answer to `setup --launch`: the same nine steps
// an operator can run one at a time, driven end to end, so there is one
// bring-up implementation rather than two.
//
// Every step still records itself in the workspace, so a run that fails part
// way leaves an inspectable composition the operator can resume by hand.

// UpStage is how far NetUp takes the composition.
type UpStage string

const (
	// UpDeploy composes and writes the artifacts (genesis, configs, argv) but
	// starts nothing, so an external launcher can boot from the result. It ends
	// after the deploy step (the old name was "provision").
	UpDeploy UpStage = "deploy"
	// UpStart additionally initializes the datadirs and launches the nodes.
	UpStart UpStage = "start"
)

// NetUpIn describes the network to compose. It is the union of the step inputs,
// in the order the steps consume them.
type NetUpIn struct {
	// DataDir is the workspace directory.
	DataDir string `json:"dataDir,omitempty"`
	// Stage is how far to go; empty means UpStart.
	Stage UpStage `json:"stage,omitempty"`

	// Chain identity (step: new).
	Chain        string        `json:"chain,omitempty"`
	ManifestPath string        `json:"manifestPath,omitempty"`
	TemplatePath string        `json:"templatePath,omitempty"`
	KeysDir      string        `json:"keysDir,omitempty"`
	Target       resource.Spec `json:"target,omitempty"`
	// Binary is the node executable. Required for UpStart; for a remote target
	// it is a path on that host.
	Binary string `json:"binary,omitempty"`

	// Layout (step: allocate).
	Validators       int    `json:"validators,omitempty"`
	Endpoints        int    `json:"endpoints,omitempty"`
	Proxies          int    `json:"proxies,omitempty"`
	EndpointSyncMode string `json:"endpointSyncMode,omitempty"`
	TopologyPath     string `json:"topologyPath,omitempty"`
	// AutoSize fills the validator count to the server set (bp: "max"): one node
	// per server, less the proxies and endpoints. It needs a server-set target.
	AutoSize bool `json:"autoSize,omitempty"`
	// BlueprintPath is a network declaration: the layout AND the keys in one
	// document. It is what lets a network be composed with no preset directory
	// anywhere (N3).
	BlueprintPath string `json:"blueprintPath,omitempty"`
	// Topology, when set, is the inline per-node layout, the DSL's in-memory
	// equivalent of TopologyPath. It wins over Validators/Endpoints.
	Topology *node.Topology `json:"topology,omitempty"`
	// Binaries maps each per-node binary name the topology references to its
	// resolved path. Empty means every node runs Binary.
	Binaries map[string]string `json:"binaries,omitempty"`
	// Peering is the peer graph to wire ("mesh" default, "proxied").
	Peering string `json:"peering,omitempty"`
	// Server selects where the nodes run and on what ports, from the server
	// server set. Its zero value uses the built-in local plan.
	Server resource.ServerRef `json:"server,omitempty"`
	// Docker treats the servers as local docker containers (dials translated
	// through the localmap next to the server set); recorded at the new step.
	Docker bool `json:"docker,omitempty"`
	// WorkspaceConfigPath is the environment file owning the target data root and
	// purpose directories; recorded at new so later steps resolve portable
	// references under it.
	WorkspaceConfigPath string `json:"workspaceConfigPath,omitempty"`

	// Identities (step: keys).
	KeysSource string `json:"keysSource,omitempty"`
	// KeysValidators is how many of a generated key set join the validator set
	// (0 = all). It has effect only when KeysSource is "generate".
	KeysValidators int `json:"keysValidators,omitempty"`

	// Genesis customization (step: genesis).
	ChainID     int64    `json:"chainID,omitempty"`
	GenesisSet  []string `json:"genesisSet,omitempty"`
	OverlayPath string   `json:"overlayPath,omitempty"`
	// GenesisExisting is a reference to a finished genesis file used verbatim
	// (genesis mode "existing"); empty builds from the template as usual.
	GenesisExisting string `json:"genesisExisting,omitempty"`

	// LaunchSet are launch-argv overrides applied to every node (step:
	// launchopts) — the "all" scope.
	LaunchSet []string `json:"launchSet,omitempty"`
	// LaunchScoped are launch-argv overrides by scope ("all", a role like "bp"/
	// "en", or "node<N>"), applied per node most-general-first. It is the DSL
	// env.launch form; the CLI passes LaunchSet.
	LaunchScoped map[string][]string `json:"launchScoped,omitempty"`

	// ConfigSet are per-scope config-knob overrides (step: config), keyed by
	// scope ("all" / "node<N>"). Each value is a list of dot-path "key=value".
	ConfigSet map[string][]string `json:"configSet,omitempty"`
}

// NetUpOut reports each step's recorded detail, in order, and the resulting
// network.
type NetUpOut struct {
	// Steps is one "name: detail" line per step that ran.
	Steps []string
	// Nodes is the composed network. Its PIDs are set only when the run reached
	// UpStart.
	Nodes NetworkStatusOut
}

// upStepNames is the composition order — the one list resume and up share.
var upStepNames = []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init", "start"}

// NetUp runs the composition steps in order and returns what each recorded.
func NetUp(ctx context.Context, d Deps, in NetUpIn) (NetUpOut, error) {
	return netUpFrom(ctx, d, in, "")
}

// netUpFrom runs the composition from the named step on (every step when
// from is empty). Steps before it are assumed done — the resume verb decides
// that from the workspace's record.
func netUpFrom(ctx context.Context, d Deps, in NetUpIn, from string) (NetUpOut, error) {
	if in.DataDir == "" {
		return NetUpOut{}, errors.New("chainsetup: chain up needs a workspace directory")
	}
	stage := in.Stage
	if stage == "" {
		stage = UpStart
	}
	if stage != UpDeploy && stage != UpStart {
		return NetUpOut{}, fmt.Errorf("chainsetup: unknown stage %q (want %s or %s)", stage, UpDeploy, UpStart)
	}
	if stage == UpStart && in.Binary == "" {
		return NetUpOut{}, errors.New("chainsetup: chain up --stage=start needs a node binary")
	}
	// execution.chain selects how this up treats an existing composition. attach
	// does not compose or launch, so it has no meaning for up; reuse-if-matching
	// reconciles a running network node by node (below); fresh is the default and
	// composes as it always has.
	mode, err := upChainMode(in)
	if err != nil {
		return NetUpOut{}, err
	}
	if mode == resource.ChainAttach {
		return NetUpOut{}, errors.New("chainsetup: chain up: execution.chain=attach does not compose or launch a network — bring the chain up separately and use the attach/run path")
	}

	// The composite holds the workspace for its whole run. Each step it calls
	// takes the lock too, but a run cannot conflict with itself (session
	// Acquire is re-entrant per process); what this closes is the gap between
	// steps, where another run used to slip in and compose over a half-built
	// network.
	lockWS, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetUpOut{}, err
	}
	lockWS.SetEnv(d.Env)
	lockWS.SetDriver(d.Driver)
	held, prev, lockState, err := lockWS.Acquire(d.command())
	if err != nil {
		return NetUpOut{}, err
	}
	defer func() { _ = held.Release() }()
	if lockState == session.LockStale {
		d.logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up", prev.Describe())
	}

	// reuse-if-matching reconciles a running network node by node. Its baseline
	// — what each node hashed to and whether it answers — must be captured now,
	// before the compose steps re-run and reset the node table. A first up over
	// an empty workspace yields an empty snapshot, which composes everything.
	reuseMode := mode == resource.ChainReuseIfMatching && stage == UpStart
	var snap reuseSnapshot
	if reuseMode {
		snap = lockWS.snapshotForReuse(ctx)
	}

	var out NetUpOut
	// record runs one step and appends its detail, stopping the whole run on the
	// first failure so a later step never composes on top of a broken one.
	record := func(name string, fn func() (string, error)) error {
		detail, err := fn()
		if err != nil {
			return fmt.Errorf("chainsetup: chain up: %s: %w", name, err)
		}
		out.Steps = append(out.Steps, name+": "+detail)
		return nil
	}

	steps := map[string]func() (string, error){
		"new": func() (string, error) {
			r, err := NetNew(ctx, d, NetNewIn{
				DataDir: in.DataDir, Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir,
				Target: in.Target, ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath,
				Docker: in.Docker, WorkspaceConfigPath: in.WorkspaceConfigPath,
			})
			if err != nil {
				return "", err
			}
			// The request is the one fact of a composition otherwise nowhere
			// on disk; it is what a resume composes from.
			if err := recordRequest(d, in); err != nil {
				return "", err
			}
			return r.Detail, nil
		},
		// Place precedes keys: the key step sizes the identity set from the
		// node table, so the layout has to exist first.
		"place": func() (string, error) {
			r, err := NetAllocate(ctx, d, NetAllocateIn{
				DataDir: in.DataDir, Validators: in.Validators, Endpoints: in.Endpoints, Proxies: in.Proxies,
				EndpointSyncMode: in.EndpointSyncMode, TopologyPath: in.TopologyPath,
				BlueprintPath: in.BlueprintPath,
				Topology:      in.Topology, Binaries: in.Binaries, Peering: in.Peering,
				Server:   in.Server,
				AutoSize: in.AutoSize,
			})
			return r.Detail, err
		},
		"keys": func() (string, error) {
			r, err := NetKeys(ctx, d, NetKeysIn{
				DataDir: in.DataDir, Source: in.KeysSource, BlueprintPath: in.BlueprintPath,
				Validators: in.KeysValidators,
			})
			return r.Detail, err
		},
		"genesis": func() (string, error) {
			r, err := NetGenesis(ctx, d, NetGenesisIn{
				DataDir: in.DataDir, ChainID: in.ChainID, Set: in.GenesisSet, OverlayPath: in.OverlayPath,
				GenesisExisting: in.GenesisExisting,
			})
			return r.Detail, err
		},
		"config": func() (string, error) {
			r, err := NetConfig(ctx, d, NetConfigIn{DataDir: in.DataDir, ScopedSet: in.ConfigSet})
			return r.Detail, err
		},
		"build": func() (string, error) {
			r, err := NetLaunchOpts(ctx, d, NetLaunchOptsIn{
				DataDir: in.DataDir, Set: in.LaunchSet, ScopedSet: in.LaunchScoped,
			})
			return r.Detail, err
		},
		"deploy": func() (string, error) {
			r, err := NetProvision(ctx, d, NetProvisionIn{DataDir: in.DataDir})
			return r.Detail, err
		},
		"init": func() (string, error) {
			r, err := NetInit(ctx, d, NetInitIn{DataDir: in.DataDir, Binary: in.Binary})
			return r.Detail, err
		},
		"start": func() (string, error) {
			r, err := NetStart(ctx, d, NetStartIn{DataDir: in.DataDir, Binary: in.Binary})
			return r.Detail, err
		},
	}

	started := from == ""
	for _, name := range upStepNames {
		if !started {
			if name != from {
				continue
			}
			started = true
		}
		if stage == UpDeploy && (name == "init" || name == "start") {
			break
		}
		if err := record(name, steps[name]); err != nil {
			return out, err
		}
		// After the artifacts are composed (through build) and before anything is
		// deployed or launched, reconcile against the running network: leave the
		// matching nodes up, tear down only the ones that drifted.
		if reuseMode && name == "build" {
			plan, rerr := reconcileUp(ctx, d, in.DataDir, snap)
			if rerr != nil {
				return out, rerr
			}
			out.Steps = append(out.Steps, "reuse: "+plan.describe())
			if plan.Refuse != "" {
				return out, fmt.Errorf("chainsetup: chain up: reuse-if-matching refused: %s", plan.Refuse)
			}
		}
	}

	nodes, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: in.DataDir})
	if err != nil {
		return out, err
	}
	out.Nodes = nodes
	return out, nil
}

// upChainMode reads how this up should treat an existing composition from the
// workspace-config's execution.chain. No config, or an empty value, is fresh —
// the default that composes as it always has.
func upChainMode(in NetUpIn) (resource.ChainMode, error) {
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

// reconcileUp runs the reuse reconciliation against the freshly composed
// workspace and saves the result, returning the plan for the caller to report
// and to stop on a refusal.
func reconcileUp(ctx context.Context, d Deps, dataDir string, snap reuseSnapshot) (reusePlan, error) {
	var plan reusePlan
	_, err := withWorkspace(d, dataDir, func(ws *Workspace) (string, error) {
		p, err := ws.reconcileReuse(ctx, snap)
		if err != nil {
			return "", err
		}
		plan = p
		return p.describe(), nil
	})
	return plan, err
}

// recordRequest writes what the composition was asked for onto the
// workspace. The location is not part of it: the record is where the
// workspace is.
func recordRequest(d Deps, in NetUpIn) error {
	req := in
	req.DataDir = ""
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		ws.state.Request = &req
		return "", nil
	})
	return err
}
