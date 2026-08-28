package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/machine"
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
	DataDir string `json:"dataDir,omitempty"`
	// Stage is how far to go; empty means UpStart.
	Stage UpStage `json:"stage,omitempty"`

	// Chain identity (step: new).
	Chain        string       `json:"chain,omitempty"`
	ManifestPath string       `json:"manifestPath,omitempty"`
	TemplatePath string       `json:"templatePath,omitempty"`
	KeysDir      string       `json:"keysDir,omitempty"`
	Target       machine.Spec `json:"target,omitempty"`
	// Binary is the node executable. Required for UpStart; for a remote target
	// it is a path on that host.
	Binary string `json:"binary,omitempty"`

	// Layout (step: allocate).
	Validators       int    `json:"validators,omitempty"`
	Endpoints        int    `json:"endpoints,omitempty"`
	EndpointSyncMode string `json:"endpointSyncMode,omitempty"`
	TopologyPath     string `json:"topologyPath,omitempty"`
	// Peering is the peer graph to wire ("mesh" default, "proxied").
	Peering string `json:"peering,omitempty"`
	// Server selects where the nodes run and on what ports, from the server
	// server set. Its zero value uses the built-in local plan.
	Server resource.ServerRef `json:"server,omitempty"`
	// Docker treats the servers as local docker containers (dials translated
	// through the localmap next to the server set); recorded at the new step.
	Docker bool `json:"docker,omitempty"`

	// Identities (step: keys).
	KeysSource string `json:"keysSource,omitempty"`

	// Genesis customization (step: genesis).
	ChainID     int64    `json:"chainID,omitempty"`
	GenesisSet  []string `json:"genesisSet,omitempty"`
	OverlayPath string   `json:"overlayPath,omitempty"`

	// LaunchSet are launch-argv overrides (step: launchopts).
	LaunchSet []string `json:"launchSet,omitempty"`
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
var upStepNames = []string{"new", "allocate", "keys", "genesis", "config", "launchopts", "provision", "init", "start"}

// NetUp runs the composition steps in order and returns what each recorded.
func NetUp(ctx context.Context, d Deps, in NetUpIn) (NetUpOut, error) {
	return netUpFrom(ctx, d, in, "")
}

// netUpFrom runs the composition from the named step on (every step when
// from is empty). Steps before it are assumed done — the resume verb decides
// that from the workspace's record.
func netUpFrom(ctx context.Context, d Deps, in NetUpIn, from string) (NetUpOut, error) {
	if in.DataDir == "" {
		return NetUpOut{}, errors.New("chainsetup: net up needs a workspace directory")
	}
	stage := in.Stage
	if stage == "" {
		stage = UpStart
	}
	if stage != UpProvision && stage != UpStart {
		return NetUpOut{}, fmt.Errorf("chainsetup: unknown stage %q (want %s or %s)", stage, UpProvision, UpStart)
	}
	if stage == UpStart && in.Binary == "" {
		return NetUpOut{}, errors.New("chainsetup: net up --stage=start needs a node binary")
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
			return fmt.Errorf("chainsetup: net up: %s: %w", name, err)
		}
		out.Steps = append(out.Steps, name+": "+detail)
		return nil
	}

	steps := map[string]func() (string, error){
		"new": func() (string, error) {
			r, err := NetNew(ctx, d, NetNewIn{
				DataDir: in.DataDir, Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir,
				Target: in.Target, ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath,
				Docker: in.Docker,
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
		// Allocate precedes keys: the key step sizes the identity set from the
		// node table, so the layout has to exist first.
		"allocate": func() (string, error) {
			r, err := NetAllocate(ctx, d, NetAllocateIn{
				DataDir: in.DataDir, Validators: in.Validators, Endpoints: in.Endpoints,
				EndpointSyncMode: in.EndpointSyncMode, TopologyPath: in.TopologyPath, Peering: in.Peering,
				Server: in.Server,
			})
			return r.Detail, err
		},
		"keys": func() (string, error) {
			r, err := NetKeys(ctx, d, NetKeysIn{DataDir: in.DataDir, Source: in.KeysSource})
			return r.Detail, err
		},
		"genesis": func() (string, error) {
			r, err := NetGenesis(ctx, d, NetGenesisIn{
				DataDir: in.DataDir, ChainID: in.ChainID, Set: in.GenesisSet, OverlayPath: in.OverlayPath,
			})
			return r.Detail, err
		},
		"config": func() (string, error) {
			r, err := NetConfig(ctx, d, NetConfigIn{DataDir: in.DataDir})
			return r.Detail, err
		},
		"launchopts": func() (string, error) {
			r, err := NetLaunchOpts(ctx, d, NetLaunchOptsIn{DataDir: in.DataDir, Set: in.LaunchSet})
			return r.Detail, err
		},
		"provision": func() (string, error) {
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
		if stage == UpProvision && (name == "init" || name == "start") {
			break
		}
		if err := record(name, steps[name]); err != nil {
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
