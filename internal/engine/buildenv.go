package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// teardownGrace is how long a built environment's teardown waits for a graceful
// stop before escalating.
const teardownGrace = 5 * time.Second

// BuildEnvFunc provisions and brings up a network for a spec, returning the node
// set and a teardown. It has the same shape as Deps.BuildEnv so a wiring can be
// assigned to it directly.
type BuildEnvFunc func(ctx context.Context, env session.Environment, spec testspec.Spec) (node.NodeSet, TeardownFunc, error)

// BuildDeps injects BuildEnv's collaborators so the composition is unit-testable
// without real chain binaries: the supervisor's launch/health seams decide
// whether a bring-up runs a process or a fake, and Provision decides what lands
// on disk.
type BuildDeps struct {
	// Plugin is the target chain.
	Plugin registry.ChainPlugin
	// Allocator resolves per-node host+ports.
	Allocator place.Allocator
	// Genesis sources the network genesis bytes.
	Genesis GenesisSource
	// Supervisor brings the network up behind a health gate and tears it down.
	Supervisor supervisor.Supervisor
	// Mode and Capacity parameterize allocation.
	Mode     place.Mode
	Capacity place.Capacity
	// Options tunes bring-up (health gating, retries).
	Options supervisor.Options
	// Caps are the advertised capabilities recorded on the plan.
	Caps []string
	// Reqs derives per-node placement requests (role/binary/sync) from a spec.
	Reqs func(spec testspec.Spec) []place.NodeReq
	// Provision materializes the plan's on-disk files (genesis, per-node config,
	// keys). It is injected because file content is chain/preset-specific; nil
	// skips provisioning (e.g. an attach-only or test build).
	Provision func(ctx context.Context, plan driver.Plan) error
}

// NewBuildEnv composes the network build: allocate placements, source genesis,
// assemble the plan, provision on-disk files, then bring the network up behind
// the supervisor's health gate. The returned teardown stops the nodes and
// removes their data dirs. It is the production wiring for Deps.BuildEnv.
func NewBuildEnv(d BuildDeps) BuildEnvFunc {
	return func(ctx context.Context, env session.Environment, spec testspec.Spec) (node.NodeSet, TeardownFunc, error) {
		reqs := d.Reqs(spec)
		if len(reqs) == 0 {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: spec resolved to no nodes")
		}

		placements, err := d.Allocator.Allocate(reqs, d.Mode, d.Capacity)
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: allocate: %w", err)
		}
		if len(placements) != len(reqs) {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: allocator returned %d placements for %d requests", len(placements), len(reqs))
		}

		gen, err := d.Genesis.Genesis(ctx, d.Plugin, countValidators(reqs))
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: genesis: %w", err)
		}

		placed := make([]PlacedNode, len(reqs))
		for i := range reqs {
			placed[i] = PlacedNode{Req: reqs[i], Placement: placements[i]}
		}
		plan, err := AssemblePlan(d.Plugin, placed, gen, env.DataPath(), d.Caps)
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: plan: %w", err)
		}

		if d.Provision != nil {
			if err := d.Provision(ctx, plan); err != nil {
				return node.NodeSet{}, nil, fmt.Errorf("engine: build env: provision: %w", err)
			}
		}

		ns, diag, err := d.Supervisor.BringUp(ctx, plan, d.Options)
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: bring up (%s): %w", diag.Mode, err)
		}

		teardown := func(ctx context.Context) error {
			return d.Supervisor.Teardown(ctx, ns, supervisor.TeardownOpts{RemoveDataDir: true, Grace: teardownGrace})
		}
		return ns, teardown, nil
	}
}

// countValidators counts placement requests whose role produces blocks.
func countValidators(reqs []place.NodeReq) int {
	n := 0
	for _, r := range reqs {
		if r.Role == node.RoleValidator {
			n++
		}
	}
	return n
}
