package chainsetup

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// teardownGrace is how long a built environment's teardown waits for a graceful
// stop before escalating.
const teardownGrace = 5 * time.Second

// TeardownFunc tears a built environment down at the end of a session.
type TeardownFunc func(ctx context.Context) error

// BuildEnvFunc provisions and brings up a network for a spec, returning the node
// set and a teardown. It has the same shape as Deps.BuildEnv so a wiring can be
// assigned to it directly.
type BuildEnvFunc func(ctx context.Context, env session.Environment, spec testspec.Spec) (node.NodeSet, TeardownFunc, error)

// BuildDeps injects BuildEnv's collaborators so the composition is unit-testable
// without real chain binaries: the launcher's launch and health hooks decide
// whether a bring-up runs a process or a fake, and Provision decides what lands
// on disk.
type BuildDeps struct {
	// Plugin is the target chain.
	Plugin registry.ChainPlugin
	// Pool is the resource the network is allocated from (addresses x port
	// slots). resource.Assign consumes it.
	Pool resource.Pool
	// Genesis sources the network genesis bytes.
	Genesis GenesisSource
	// Supervisor brings the network up behind a health gate and tears it down.
	Supervisor launcher.Launcher
	// Options tunes bring-up (health gating, retries).
	Options launcher.Options
	// Caps are the advertised capabilities recorded on the plan.
	Caps []string
	// Reqs derives per-node placement requests (role/binary/sync) from a spec.
	Reqs func(spec testspec.Spec) []node.LaunchReq
	// Provision materializes the plan's on-disk files (genesis, per-node config,
	// keys). It is injected because file content is chain/preset-specific; nil
	// skips provisioning (e.g. an attach-only or test build).
	Provision func(ctx context.Context, plan driver.Plan) error
	// ProvisionExtra writes the genesis step's by-products to the target,
	// keyed by the name they take there. A family with none never calls it.
	ProvisionExtra func(ctx context.Context, plan driver.Plan, files map[string][]byte) error
}

// NewBuildEnv composes the network build: allocate placements, source genesis,
// assemble the plan, provision on-disk files, then bring the network up behind
// the launcher's health gate. The returned teardown stops the nodes and
// removes their data dirs. It is the production wiring for Deps.BuildEnv.
func NewBuildEnv(d BuildDeps) BuildEnvFunc {
	return func(ctx context.Context, env session.Environment, spec testspec.Spec) (node.NodeSet, TeardownFunc, error) {
		reqs := d.Reqs(spec)
		if len(reqs) == 0 {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: spec resolved to no nodes")
		}

		assigned, err := resource.Assign(d.Pool, netmapRequests(reqs))
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: allocate: %w", err)
		}
		placements := assigned.Placements()
		if len(placements) != len(reqs) {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: allocator returned %d placements for %d requests", len(placements), len(reqs))
		}

		gen, err := d.Genesis.Genesis(ctx, d.Plugin, GenesisRequest{Validators: countValidators(reqs), Nodes: assigned})
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: genesis: %w", err)
		}

		placed := make([]PlacedNode, len(reqs))
		for i := range reqs {
			placed[i] = PlacedNode{Req: reqs[i], Placement: placements[i]}
		}
		plan, err := AssemblePlan(d.Plugin, placed, gen.Genesis, env.DataPath(), d.Caps)
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: plan: %w", err)
		}

		if d.Provision != nil {
			if err := d.Provision(ctx, plan); err != nil {
				return node.NodeSet{}, nil, fmt.Errorf("engine: build env: provision: %w", err)
			}
		}
		// The genesis step's by-products go to the target beside the genesis:
		// a wemix bring-up reads its governance config back during
		// deploy-governance, and reconstructing it there is how two steps come
		// to disagree about what the network was configured with.
		if len(gen.Extra) > 0 && d.ProvisionExtra != nil {
			if err := d.ProvisionExtra(ctx, plan, gen.Extra); err != nil {
				return node.NodeSet{}, nil, fmt.Errorf("engine: build env: genesis artifacts: %w", err)
			}
		}

		// The family orders the bring-up. A wbft network declares one phase
		// and launches exactly as it did before; a poa network puts its
		// producer first so the etcd cluster can form while it is alone.
		opts := d.Options
		if len(opts.Phases) == 0 {
			roles := make([]node.Role, 0, len(plan.Nodes))
			for _, spec := range plan.Nodes {
				roles = append(roles, spec.Role)
			}
			opts.Phases = d.Plugin.Family().BringUpPhases(roles)
		}

		ns, diag, err := d.Supervisor.BringUp(ctx, plan, opts)
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: build env: bring up (%s): %w", diag.Mode, err)
		}

		teardown := func(ctx context.Context) error {
			return d.Supervisor.Teardown(ctx, ns, launcher.TeardownOpts{RemoveDataDir: true, Grace: teardownGrace})
		}
		return ns, teardown, nil
	}
}

// countValidators counts placement requests whose role produces blocks.
func countValidators(reqs []node.LaunchReq) int {
	n := 0
	for _, r := range reqs {
		if node.Is(r.Role, node.RoleBP) {
			n++
		}
	}
	return n
}
