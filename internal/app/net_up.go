package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/target"
	"github.com/0xmhha/chainbench/internal/netcompose"
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
	// UpProvision composes and writes the artifacts (genesis, configs, argv)
	// but starts nothing, so an external launcher can boot from the result.
	UpProvision UpStage = "provision"
	// UpStart additionally initializes the datadirs and launches the nodes.
	UpStart UpStage = "start"
)

// NetUpIn describes the network to compose. It is the union of the step inputs,
// in the order the steps consume them.
type NetUpIn struct {
	// DataDir is the workspace directory.
	DataDir string
	// Stage is how far to go; empty means UpStart.
	Stage UpStage

	// Chain identity (step: new).
	Chain        string
	ManifestPath string
	TemplatePath string
	KeysDir      string
	Target       target.TargetSpec
	// Binary is the node executable. Required for UpStart; for a remote target
	// it is a path on that host.
	Binary string

	// Layout (step: allocate).
	Validators       int
	Endpoints        int
	EndpointSyncMode string
	TopologyPath     string
	// Peering is the peer graph to wire ("mesh" default, "proxied").
	Peering string
	// Server selects where the nodes run and on what ports, from the server
	// inventory. Its zero value uses the built-in local plan.
	Server ServerRef
	// Docker treats the servers as local docker containers (dials translated
	// through the localmap next to the inventory); recorded at the new step.
	Docker bool

	// Identities (step: keys).
	KeysSource string

	// Genesis customization (step: genesis).
	ChainID     int64
	GenesisSet  []string
	OverlayPath string

	// LaunchSet are launch-argv overrides (step: launchopts).
	LaunchSet []string
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

// upStep is one named step of the macro.
type upStep struct {
	name string
	fn   func() (string, error)
}

// NetUp runs the composition steps in order and returns what each recorded.
func NetUp(ctx context.Context, d Deps, in NetUpIn) (NetUpOut, error) {
	if in.DataDir == "" {
		return NetUpOut{}, errors.New("app: net up needs a workspace directory")
	}
	stage := in.Stage
	if stage == "" {
		stage = UpStart
	}
	if stage != UpProvision && stage != UpStart {
		return NetUpOut{}, fmt.Errorf("app: unknown stage %q (want %s or %s)", stage, UpProvision, UpStart)
	}
	if stage == UpStart && in.Binary == "" {
		return NetUpOut{}, errors.New("app: net up --stage=start needs a node binary")
	}

	// The composite holds the workspace for its whole run. Each step it calls
	// takes the lock too, but a run cannot conflict with itself (session
	// Acquire is re-entrant per process); what this closes is the gap between
	// steps, where another run used to slip in and compose over a half-built
	// network.
	lockWS, err := netcompose.Open(in.DataDir, d.Clock)
	if err != nil {
		return NetUpOut{}, err
	}
	held, prev, lockState, err := lockWS.Acquire(d.command())
	if err != nil {
		return NetUpOut{}, err
	}
	defer func() { _ = held.Release() }()
	if lockState == session.LockStale {
		d.logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up", prev.Describe())
	}

	var out NetUpOut
	// record runs one step and appends its detail, stopping the whole run on the
	// first failure so a later step never composes on top of a broken one.
	record := func(name string, fn func() (string, error)) error {
		detail, err := fn()
		if err != nil {
			return fmt.Errorf("app: net up: %s: %w", name, err)
		}
		out.Steps = append(out.Steps, name+": "+detail)
		return nil
	}

	steps := []upStep{
		{"new", func() (string, error) {
			r, err := NetNew(ctx, d, NetNewIn{
				DataDir: in.DataDir, Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir,
				Target: in.Target, ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath,
				Docker: in.Docker,
			})
			return r.Detail, err
		}},
		// Allocate precedes keys: the key step sizes the identity set from the
		// node table, so the layout has to exist first.
		{"allocate", func() (string, error) {
			r, err := NetAllocate(ctx, d, NetAllocateIn{
				DataDir: in.DataDir, Validators: in.Validators, Endpoints: in.Endpoints,
				EndpointSyncMode: in.EndpointSyncMode, TopologyPath: in.TopologyPath, Peering: in.Peering,
				Server: in.Server,
			})
			return r.Detail, err
		}},
		{"keys", func() (string, error) {
			r, err := NetKeys(ctx, d, NetKeysIn{
				DataDir: in.DataDir, Source: in.KeysSource,
			})
			return r.Detail, err
		}},
		{"genesis", func() (string, error) {
			r, err := NetGenesis(ctx, d, NetGenesisIn{
				DataDir: in.DataDir, ChainID: in.ChainID, Set: in.GenesisSet, OverlayPath: in.OverlayPath,
			})
			return r.Detail, err
		}},
		{"config", func() (string, error) {
			r, err := NetConfig(ctx, d, NetConfigIn{DataDir: in.DataDir})
			return r.Detail, err
		}},
		{"launchopts", func() (string, error) {
			r, err := NetLaunchOpts(ctx, d, NetLaunchOptsIn{DataDir: in.DataDir, Set: in.LaunchSet})
			return r.Detail, err
		}},
		{"provision", func() (string, error) {
			r, err := NetProvision(ctx, d, NetProvisionIn{DataDir: in.DataDir})
			return r.Detail, err
		}},
	}
	if stage == UpStart {
		steps = append(steps,
			upStep{"init", func() (string, error) {
				r, err := NetInit(ctx, d, NetInitIn{DataDir: in.DataDir, Binary: in.Binary})
				return r.Detail, err
			}},
			upStep{"start", func() (string, error) {
				r, err := NetStart(ctx, d, NetStartIn{DataDir: in.DataDir, Binary: in.Binary})
				return r.Detail, err
			}},
		)
	}

	for _, s := range steps {
		if err := record(s.name, s.fn); err != nil {
			return out, err
		}
	}

	nodes, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: in.DataDir})
	if err != nil {
		return out, err
	}
	out.Nodes = nodes
	return out, nil
}
