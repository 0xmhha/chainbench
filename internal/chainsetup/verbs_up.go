package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
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

// NetUp runs the composition steps in order and returns what each recorded.
func NetUp(ctx context.Context, d Deps, in NetUpIn) (NetUpOut, error) {
	return netUpFrom(ctx, d, in, "")
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

// markStepFailed writes a failed step into the composition record.
//
// Best effort on purpose, and silent when it cannot write: this runs while a
// composition is already failing, and a second error about the bookkeeping
// would bury the first one — which is the error the operator came for.
func markStepFailed(d Deps, dataDir, name string, cause error) {
	ws, err := Open(dataDir, d.Clock)
	if err != nil {
		return
	}
	ws.MarkStepFailed(name, cause)
	_ = ws.Save()
}

// upPlan is a validated up request: the stage it runs to and how it treats an
// existing composition, both resolved from defaults and checked once.
type upPlan struct {
	stage UpStage
	mode  resource.ChainMode
}

// planUp validates the request and resolves its defaults. It writes nothing:
// every refusal here happens before the workspace is opened, which is the point
// of doing it in one place up front.
func planUp(in NetUpIn) (upPlan, error) {
	if in.DataDir == "" {
		return upPlan{}, errors.New("chainsetup: chain up needs a workspace directory")
	}
	stage := in.Stage
	if stage == "" {
		stage = UpStart
	}
	if stage != UpDeploy && stage != UpStart {
		return upPlan{}, fmt.Errorf("chainsetup: unknown stage %q (want %s or %s)", stage, UpDeploy, UpStart)
	}
	if stage == UpStart && in.Binary == "" {
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
	if err := checkTopologyKeyRefs(in.Topology); err != nil {
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

// upSteps is the composition's step table: one closure per name in
// upStepNames, each wrapping the same verb the matching `chain <step>` command
// calls. It is built once and read by the runner below, so the order the run
// follows and the work each step does stay separate things.
func upSteps(ctx context.Context, d Deps, in NetUpIn) map[string]func() (StepOut, error) {
	return map[string]func() (StepOut, error){
		"new": func() (StepOut, error) {
			r, err := NetNew(ctx, d, NetNewIn{
				DataDir: in.DataDir, Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir,
				Target: in.Target, ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath,
				Docker: in.Docker, WorkspaceConfigPath: in.WorkspaceConfigPath,
			})
			if err != nil {
				return StepOut{}, err
			}
			// The request is the one fact of a composition otherwise nowhere
			// on disk; it is what a resume composes from.
			if err := recordRequest(d, in); err != nil {
				return StepOut{}, err
			}
			return StepOut{Detail: r.Detail}, nil
		},
		// Place precedes keys: the key step sizes the identity set from the
		// node table, so the layout has to exist first.
		"place": func() (StepOut, error) {
			r, err := NetAllocate(ctx, d, NetAllocateIn{
				DataDir: in.DataDir, BPCount: in.BPCount, ENCount: in.ENCount, PNCount: in.PNCount,
				EndpointSyncMode: in.EndpointSyncMode, TopologyPath: in.TopologyPath,
				BlueprintPath: in.BlueprintPath,
				Topology:      in.Topology, Binaries: in.Binaries, BinaryChains: in.BinaryChains, Peering: in.Peering,
				Server:   in.Server,
				AutoSize: in.AutoSize,
			})
			return r, err
		},
		"keys": func() (StepOut, error) {
			r, err := NetKeys(ctx, d, NetKeysIn{
				DataDir: in.DataDir, Source: in.KeysSource, BlueprintPath: in.BlueprintPath,
				Validators: in.KeysValidators,
			})
			return r, err
		},
		"genesis": func() (StepOut, error) {
			r, err := NetGenesis(ctx, d, NetGenesisIn{
				DataDir: in.DataDir, ChainID: in.ChainID, Set: in.GenesisSet, OverlayPath: in.OverlayPath,
				GenesisExisting: in.GenesisExisting, PerBinary: in.GenesisPerBinary,
				Fork: in.GenesisFork,
			})
			return r, err
		},
		"config": func() (StepOut, error) {
			r, err := NetConfig(ctx, d, NetConfigIn{DataDir: in.DataDir, ScopedSet: in.ConfigSet})
			return r, err
		},
		"build": func() (StepOut, error) {
			r, err := NetLaunchOpts(ctx, d, NetLaunchOptsIn{
				DataDir: in.DataDir, Set: in.LaunchSet, ScopedSet: in.LaunchScoped,
			})
			return StepOut{Detail: r.Detail}, err
		},
		"deploy": func() (StepOut, error) {
			r, err := NetProvision(ctx, d, NetProvisionIn{DataDir: in.DataDir})
			return r, err
		},
		"init": func() (StepOut, error) {
			r, err := NetInit(ctx, d, NetInitIn{DataDir: in.DataDir, Binary: in.Binary})
			return r, err
		},
		"start": func() (StepOut, error) {
			r, err := NetStart(ctx, d, NetStartIn{DataDir: in.DataDir, Binary: in.Binary})
			return r, err
		},
	}
}

// netUpFrom runs the composition from the named step on (every step when
// from is empty). Steps before it are assumed done — the resume verb decides
// that from the workspace's record.
func netUpFrom(ctx context.Context, d Deps, in NetUpIn, from string) (NetUpOut, error) {
	up, err := planUp(in)
	if err != nil {
		return NetUpOut{}, err
	}
	// Before the workspace is opened, so what this request records, compares
	// and launches with is one form of the same reference.
	if err := placeUpRequest(&in); err != nil {
		return NetUpOut{}, err
	}
	stage, mode := up.stage, up.mode

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
	//
	// A failure is written into the record before it is returned. The step verbs
	// mark themselves only on success, which meant a composition that died left
	// the record saying nothing at all about the step it died in: the reader saw
	// the last step that WORKED and had to guess what came next. Now the record
	// names the step, the time, and the error.
	record := func(name string, fn func() (StepOut, error)) ([]lifecycle.Status, error) {
		r, err := fn()
		if err != nil {
			werr := fmt.Errorf("chainsetup: chain up: %s: %w", name, err)
			markStepFailed(d, in.DataDir, name, werr)
			// The path comes back with the error: how far the step got is what
			// decides which state its failure is reached from.
			return r.Passed, werr
		}
		out.Steps = append(out.Steps, name+": "+r.Detail)
		// Empty when the step's work has not moved into its handler: the handler
		// then walks the path it assumed instead of the one that happened.
		return r.Passed, nil
	}

	steps := upSteps(ctx, d, in)
	run := func(name string) ([]lifecycle.Status, error) { return record(name, steps[name]) }

	// The composition is walked by its state. What used to decide how far to go
	// is the machine's target, what decides which stage comes next is the
	// transition table, and reconciling against a running network is a state
	// between the keys and the genesis rather than a comparison on a step name
	// in the middle of a loop.
	start, err := startFor(from)
	if err != nil {
		return out, err
	}
	target, err := targetFor(stage)
	if err != nil {
		return out, err
	}
	handlers := handlersFor(composeStages(reuseMode), run)
	if reuseMode {
		handlers[lifecycle.ReconcileChain] = reconcileHandler(ctx, d, in, snap, func(line string) { out.Steps = append(out.Steps, line) })
	}
	m, err := lifecycle.New(start, target, handlers)
	if err != nil {
		return out, err
	}
	if err := m.Run(ctx); err != nil {
		return out, err
	}

	nodes, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: in.DataDir})
	if err != nil {
		return out, err
	}
	out.Nodes = nodes
	return out, nil
}

// placeUpRequest places the request's binary references through the environment
// file it names, if it names one.
func placeUpRequest(in *NetUpIn) error {
	if in.WorkspaceConfigPath == "" {
		return PlaceRequest(in, nil)
	}
	wc, err := resource.LoadWorkspaceConfig(in.WorkspaceConfigPath)
	if err != nil {
		return fmt.Errorf("chainsetup: chain up: %w", err)
	}
	return PlaceRequest(in, &wc)
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
