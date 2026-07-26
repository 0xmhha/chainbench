package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// LaunchOptions carries the runtime bindings a Plan needs to actually run: the
// two binaries (resolved from the golden profile), their consensus families
// (for role flags), where node data lives, and how a datadir is initialized.
// Nothing here is baked in — the binaries and paths come from the profile.
type LaunchOptions struct {
	// DataRoot is the parent of each node's datadir (DataRoot/node<n>).
	DataRoot string
	// FromBinary runs the producers (pre-fork), ToBinary the validators (post-fork).
	FromBinary, ToBinary string
	// FromFamily/ToFamily supply each node's role launch flags.
	FromFamily, ToFamily registry.ConsensusFamily
	// Host is the address nodes bind/advertise; defaults to 127.0.0.1.
	Host string
	// InitFn initializes a node's datadir from the shared genesis using the
	// node's own binary; defaults to driver.InitDatadir. Injectable for tests.
	InitFn func(ctx context.Context, binary, dataDir, genesisPath string) error
	// ProvisionKeys, if set, runs after a node's datadir is initialized and
	// before it launches. It is where external key material (the node key in the
	// binary-specific instance dir, the producer's keystore, static-nodes) is
	// placed. Optional; nil means the datadir is used as initialized.
	ProvisionKeys func(ctx context.Context, spec driver.NodeSpec, producer bool) error
}

func (o LaunchOptions) host() string {
	if o.Host == "" {
		return "127.0.0.1"
	}
	return o.Host
}

// BuildNodeSpecs turns a Plan into the driver NodeSpecs that launch it: each
// producer on the from-binary with the from-family's flags, each validator on
// the to-binary with the to-family's flags, all with the plan's uniform network
// id and collision-free ports. Pure — no disk or process side effects.
func BuildNodeSpecs(plan Plan, opts LaunchOptions) ([]driver.NodeSpec, error) {
	if opts.FromBinary == "" || opts.ToBinary == "" {
		return nil, fmt.Errorf("upgrade: both from and to binaries must be set")
	}
	if opts.FromFamily == nil || opts.ToFamily == nil {
		return nil, fmt.Errorf("upgrade: both from and to consensus families must be set")
	}
	specs := make([]driver.NodeSpec, 0, len(plan.Nodes))
	for _, n := range plan.Nodes {
		binary, fam := opts.ToBinary, opts.ToFamily
		if n.Producer {
			binary, fam = opts.FromBinary, opts.FromFamily
		}
		num := n.Index + 1
		dataDir := filepath.Join(opts.DataRoot, fmt.Sprintf("node%d", num))
		configPath := filepath.Join(opts.DataRoot, fmt.Sprintf("config_node%d.toml", num))
		endpoints := node.Endpoints{P2P: n.Ports.P2P, HTTP: n.Ports.HTTP, WS: n.Ports.WS, Auth: n.Ports.Auth}
		specs = append(specs, driver.NodeSpec{
			Index:      n.Index,
			Role:       n.Role,
			Host:       opts.host(),
			Binary:     binary,
			DataDir:    dataDir,
			ConfigPath: configPath,
			LogPath:    filepath.Join(opts.DataRoot, "logs", fmt.Sprintf("node%d.log", num)),
			Args:       LaunchArgs(n, dataDir, configPath, fam.StartFlags(n.Role)),
			Ports:      endpoints,
		})
	}
	return specs, nil
}

// Bootstrap performs the post-launch producer bring-up (e.g. deploy-governance
// + etcdInit for a wemix producer). It runs once per producer node. Injectable
// so the orchestration is testable without a real chain.
type Bootstrap func(ctx context.Context, producer node.Node) error

// LaunchHandoff runs the full post-plan sequence that a live handoff needs:
// Launch every node, wire a full peer mesh so the successor validators can reach
// a quorum with each other, then bootstrap each producer. It is the framework
// equivalent of the reproduction script's launch+peer+bootstrap steps. The peer
// caller and bootstrap are injected (defaults: JSON-RPC admin_addPeer, and a
// no-op bootstrap) so it can be exercised without binaries. Provisioning of key
// material into datadirs is the caller's concern (external key layout).
func LaunchHandoff(ctx context.Context, d driver.Driver, plan Plan, opts LaunchOptions, caller PeerCaller, bootstrap Bootstrap) (node.NodeSet, error) {
	ns, err := Launch(ctx, d, plan, opts)
	if err != nil {
		return ns, err
	}
	if caller == nil {
		caller = DefaultPeerCaller()
	}
	endpoints := make([]string, len(ns.Nodes))
	for i, n := range ns.Nodes {
		endpoints[i] = n.RPCURL
	}
	if err := WireMesh(ctx, caller, endpoints, plan.Enodes(opts.host())); err != nil {
		return ns, err
	}
	if bootstrap != nil {
		for i, spec := range plan.Nodes {
			if spec.Producer {
				if err := bootstrap(ctx, ns.Nodes[i]); err != nil {
					return ns, fmt.Errorf("upgrade: bootstrap producer node%d: %w", spec.Index+1, err)
				}
			}
		}
	}
	return ns, nil
}

// Launch runs a handoff network: it writes the shared genesis, initializes each
// node's datadir with that node's own binary (so go-wemix and go-wbft each lay
// out their chaindata correctly from identical genesis bytes), then provisions
// and launches every node concurrently through the driver. Producers and
// validators run at the same time — this is a concurrent handoff, not a binary
// swap: producers mine up to the fork, validators sync and take over after it.
// It returns the launched NodeSet.
func Launch(ctx context.Context, d driver.Driver, plan Plan, opts LaunchOptions) (node.NodeSet, error) {
	initFn := opts.InitFn
	if initFn == nil {
		initFn = driver.InitDatadir
	}
	ns := node.NodeSet{Chain: plan.To.ID, Network: "local"}

	if err := os.MkdirAll(opts.DataRoot, 0o755); err != nil {
		return ns, fmt.Errorf("upgrade: data root: %w", err)
	}
	genesisPath := filepath.Join(opts.DataRoot, "genesis.json")
	if err := os.WriteFile(genesisPath, plan.Genesis, 0o644); err != nil {
		return ns, fmt.Errorf("upgrade: write genesis: %w", err)
	}

	specs, err := BuildNodeSpecs(plan, opts)
	if err != nil {
		return ns, err
	}
	for i, spec := range specs {
		if err := initFn(ctx, spec.Binary, spec.DataDir, genesisPath); err != nil {
			return ns, fmt.Errorf("upgrade: init node%d (%s): %w", spec.Index+1, spec.Binary, err)
		}
		if opts.ProvisionKeys != nil {
			if err := opts.ProvisionKeys(ctx, spec, plan.Nodes[i].Producer); err != nil {
				return ns, fmt.Errorf("upgrade: provision keys node%d: %w", spec.Index+1, err)
			}
		}
		if err := d.Provision(ctx, spec); err != nil {
			return ns, fmt.Errorf("upgrade: provision node%d: %w", spec.Index+1, err)
		}
		h, err := d.Launch(ctx, spec)
		if err != nil {
			return ns, fmt.Errorf("upgrade: launch node%d (%s): %w", spec.Index+1, spec.Binary, err)
		}
		ns.Nodes = append(ns.Nodes, node.Node{
			Index:  spec.Index,
			Role:   spec.Role,
			Host:   spec.Host,
			RPCURL: fmt.Sprintf("http://%s:%d", spec.Host, spec.Ports.HTTP),
			Ports:  spec.Ports,
			PID:    h.PID,
		})
	}
	return ns, nil
}
