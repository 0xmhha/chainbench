package verb

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
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

// markStepFailed writes a failed step into the composition record.
//
// Best effort on purpose, and silent when it cannot write: this runs while a
// composition is already failing, and a second error about the bookkeeping
// would bury the first one — which is the error the operator came for.
func markStepFailed(d chainsetup.Deps, dataDir, name string, cause error) {
	ws, err := chainsetup.Open(dataDir, d.Clock)
	if err != nil {
		return
	}
	ws.MarkStepFailed(name, cause)
	_ = ws.Save()
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

// upSteps is the reuse path's step table: one closure per name in UpStepNames.
//
// It calls the step bodies rather than the `chain <step>` verbs, because those
// go through the composition machine now and the machine does not report the
// lifecycle states this table replays. Two callers of one body, for as long as
// this path exists: it is the last thing still walking the old table, and it
// goes when the reconciliation becomes a state.
func upSteps(ctx context.Context, d chainsetup.Deps, in chainsetup.ChainUpIn) map[string]func() (chainsetup.StepOut, error) {
	body := func(fn func(ws *chainsetup.Workspace) (chainsetup.StepOut, error)) func() (chainsetup.StepOut, error) {
		return func() (chainsetup.StepOut, error) {
			return chainsetup.InWorkspace(d, in.DataDir, fn)
		}
	}
	line := func(fn func(ws *chainsetup.Workspace) (string, error)) func() (chainsetup.StepOut, error) {
		return body(func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
			detail, err := fn(ws)
			return chainsetup.StepOut{Detail: detail}, err
		})
	}
	return map[string]func() (chainsetup.StepOut, error){
		"new": line(func(ws *chainsetup.Workspace) (string, error) {
			detail, err := ws.New(chainsetup.NewOpts{
				Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir, Target: in.Target,
				ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath,
				Docker: in.Docker, WorkspaceConfigPath: in.WorkspaceConfigPath,
			})
			if err != nil {
				return "", err
			}
			// The request is the one fact of a composition otherwise nowhere on
			// disk; it is what a resume composes from.
			return detail, ws.RecordRequest(in)
		}),
		// Place precedes keys: the key step sizes the identity set from the
		// node table, so the layout has to exist first.
		"place": func() (chainsetup.StepOut, error) {
			detail, err := chainsetup.PlaceNodes(d, chainsetup.ChainAllocateIn{
				DataDir: in.DataDir, BPCount: in.BPCount, ENCount: in.ENCount, PNCount: in.PNCount,
				EndpointSyncMode: in.EndpointSyncMode, Peering: in.Peering,
				TopologyPath: in.TopologyPath, BlueprintPath: in.BlueprintPath,
				Topology: in.Topology, Binaries: in.Binaries, BinaryChains: in.BinaryChains,
				Server: in.Server, AutoSize: in.AutoSize,
			})
			return chainsetup.StepOut{Detail: detail}, err
		},
		"keys": body(func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
			opts, err := chainsetup.KeysOptsFor(in.BlueprintPath, in.KeysSource, in.KeysNodes, in.KeysValidators)
			if err != nil {
				return chainsetup.StepOut{}, err
			}
			return ws.Keys(ctx, opts)
		}),
		"genesis": body(func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
			opts, err := chainsetup.GenesisOptsFor(chainsetup.ChainGenesisIn{
				DataDir: in.DataDir, ChainID: in.ChainID, Set: in.GenesisSet,
				OverlayPath: in.OverlayPath, GenesisExisting: in.GenesisExisting,
				PerBinary: in.GenesisPerBinary, Fork: in.GenesisFork,
			})
			if err != nil {
				return chainsetup.StepOut{}, err
			}
			return ws.Genesis(ctx, opts)
		}),
		"config": line(func(ws *chainsetup.Workspace) (string, error) {
			for _, scope := range chainsetup.SortedScopes(in.ConfigSet) {
				if err := ws.RecordConfigSet(scope, in.ConfigSet[scope]); err != nil {
					return "", fmt.Errorf("chainsetup: config: %w", err)
				}
			}
			return ws.Config(ctx)
		}),
		"build": line(func(ws *chainsetup.Workspace) (string, error) {
			for _, scope := range chainsetup.SortedScopes(in.LaunchScoped) {
				if err := ws.RecordLaunchSet(scope, in.LaunchScoped[scope]); err != nil {
					return "", fmt.Errorf("chainsetup: launchopts: %w", err)
				}
			}
			if err := ws.RecordLaunchCommand(in.LaunchSet); err != nil {
				return "", fmt.Errorf("chainsetup: launchopts: %w", err)
			}
			return ws.LaunchOpts()
		}),
		"deploy": body(func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
			return ws.Provision(ctx)
		}),
		"init": line(func(ws *chainsetup.Workspace) (string, error) {
			return ws.Init(ctx, in.Binary)
		}),
		"start": body(func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
			return ws.Start(ctx, in.Binary)
		}),
	}
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

	// The composite holds the workspace for its whole run. Each step it calls
	// takes the lock too, but a run cannot conflict with itself (session
	// Acquire is re-entrant per process); what this closes is the gap between
	// steps, where another run used to slip in and compose over a half-built
	// network.
	lockWS, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return ChainUpOut{}, err
	}
	lockWS.SetEnv(d.Env)
	lockWS.SetDriver(d.Driver)
	held, prev, lockState, err := lockWS.Acquire(d.Owner())
	if err != nil {
		return ChainUpOut{}, err
	}
	defer func() { _ = held.Release() }()
	if lockState == session.LockStale {
		d.Logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up", prev.Describe())
	}

	// reuse-if-matching reconciles a running network node by node. Its baseline
	// — what each node hashed to and whether it answers — must be captured now,
	// before the compose steps re-run and reset the node table. A first up over
	// an empty workspace yields an empty snapshot, which composes everything.
	reuseMode := mode == resource.ChainReuseIfMatching && stage == chainsetup.UpStart
	var snap chainsetup.ReuseSnapshot
	if reuseMode {
		snap = lockWS.SnapshotForReuse(ctx)
	}

	var out ChainUpOut
	// record runs one step and appends its detail, stopping the whole run on the
	// first failure so a later step never composes on top of a broken one.
	//
	// A failure is written into the record before it is returned. The step verbs
	// mark themselves only on success, which meant a composition that died left
	// the record saying nothing at all about the step it died in: the reader saw
	// the last step that WORKED and had to guess what came next. Now the record
	// names the step, the time, and the error.
	record := func(name string, fn func() (chainsetup.StepOut, error)) ([]lifecycle.Status, error) {
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

	// Two machines for the length of this series. A composition walks the state
	// machine; reuse-if-matching still walks the old table, because its
	// reconciliation is written against that table's Request and moving it is
	// its own commit. Nothing else tells the two apart: both run the same nine
	// step bodies in the same order and record the same thing.
	if reuseMode {
		if err := reconcilingUp(ctx, d, in, from, stage, snap, run, &out); err != nil {
			return out, err
		}
	} else {
		// Every stage does its own work now, so nothing of this function's is
		// handed in: the machine is given the workspace and the request, and
		// what it reports is read back afterwards.
		mgr := chainsetup.NewManager(d, lockWS)
		cerr := mgr.Compose(ctx, in, from)
		// Read before the error is returned: how far a dead run got is the
		// first thing its reader wants.
		out.Steps = append(out.Steps, mgr.Steps()...)
		if cerr != nil {
			return out, cerr
		}
	}

	nodes, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: in.DataDir})
	if err != nil {
		return out, err
	}
	out.Nodes = nodes
	return out, nil
}

// reconcilingUp composes through the old transition table.
//
// It is what a reuse-if-matching run still uses. The reconciliation between the
// keys and the genesis asks the old machine to move, so it cannot be handed to
// the new one until it is rewritten as a state of its own; until then this is
// the path it takes, unchanged.
func reconcilingUp(
	ctx context.Context,
	d chainsetup.Deps,
	in chainsetup.ChainUpIn,
	from string,
	stage chainsetup.UpStage,
	snap chainsetup.ReuseSnapshot,
	run composeRun,
	out *ChainUpOut,
) error {
	start, err := startFor(from)
	if err != nil {
		return err
	}
	target, err := targetFor(stage)
	if err != nil {
		return err
	}
	handlers := handlersFor(composeStages(true), run)
	handlers[lifecycle.ReconcileChain] = chainsetup.ReconcileHandler(ctx, d, in, snap, func(line string) { out.Steps = append(out.Steps, line) })
	m, err := lifecycle.New(start, target, handlers)
	if err != nil {
		return err
	}
	return m.Run(ctx)
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
