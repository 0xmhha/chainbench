package upgrade

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
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
	// Overrides, if set, returns a node's high-precedence launch knobs, applied
	// through the launchopt Builder after the standard and family layers. It is
	// where account-specific and RPC-namespace knobs go: the producer's
	// unlock/etherbase/password, and the http.api set (admin is required for the
	// mesh's admin_addPeer). Typed keys, so an unsupported knob is a classified
	// assembly error instead of a silently ignored flag. Optional.
	Overrides func(spec NodeSpec, producer bool) []nodeconfig.Override
	// Files is where the shared genesis is written. Nil is the local
	// filesystem, which is what a local handoff wants and what this used to do
	// unconditionally — the boundary exists so a caller running against a remote
	// target can send the genesis where the nodes are rather than to the
	// machine driving them — the defect the remote-provision path used to
	// have, where a remote network's files landed on the operator's machine.
	Files filestore.Store
}

func (o LaunchOptions) files() filestore.Store {
	if o.Files == nil {
		return filestore.Local{}
	}
	return o.Files
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
		endpoints := n.Ports
		var overrides []nodeconfig.Override
		if opts.Overrides != nil {
			overrides = opts.Overrides(n, n.Producer)
		}
		args, err := LaunchArgs(n, dataDir, fam.StartFlags(n.Role), overrides...)
		if err != nil {
			return nil, fmt.Errorf("upgrade: node%d: %w", num, err)
		}
		specs = append(specs, driver.NodeSpec{
			Index:      n.Index,
			Role:       n.Role,
			Host:       opts.host(),
			Binary:     binary,
			DataDir:    dataDir,
			ConfigPath: configPath,
			LogPath:    filepath.Join(opts.DataRoot, "logs", fmt.Sprintf("node%d.log", num)),
			Args:       args,
			Ports:      endpoints,
		})
	}
	return specs, nil
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

	// Write creates the parents, so the data root needs no separate mkdir.
	genesisPath := filepath.Join(opts.DataRoot, "genesis.json")
	if err := opts.files().Write(ctx, genesisPath, plan.Genesis, 0o644); err != nil {
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
		ns.Nodes = append(ns.Nodes, driver.NodeOf(spec, h.PID))
	}
	return ns, nil
}
