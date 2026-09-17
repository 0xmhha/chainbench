package upgrade

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
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
	// FromChain/ToChain are the two chains a node may run, and each supplies
	// both halves of its launch: what its consensus asks for (a producer seals)
	// and which flag vocabulary its binary accepts. They travel as plugins
	// rather than as families because the vocabulary is the manifest's answer,
	// and a handoff that carried only the family left it unset — every node
	// then assembled argv against an empty dialect.
	FromChain, ToChain registry.ChainPlugin
	// Host is the address nodes bind/advertise; defaults to 127.0.0.1.
	Host string
	// StaticNodes are the enode URLs every node dials. They go into each
	// node's config file, which is where a geth-family binary reads them:
	// the static-nodes.json this used to write is deprecated, and go-wbft says
	// so in its log and ignores the file.
	StaticNodes []string
	// Identity says which account a node unlocks and seals with, and where its
	// password file is. A node that seals nothing returns empty.
	//
	// It is a field of the configuration rather than a launch override because
	// both renderers need it: the config file names the keystore directory and
	// the command line names the account.
	Identity func(spec NodeSpec, producer bool) (unlock, passwordFile string)
	// InitFn initializes a node's datadir from the shared genesis using the
	// node's own binary; defaults to process.InitDatadir. Injectable for tests.
	InitFn func(ctx context.Context, binary, dataDir, genesisPath string) error
	// ProvisionKeys, if set, runs after a node's datadir is initialized and
	// before it launches. It is where external key material (the node key in the
	// binary-specific instance dir, the producer's keystore, static-nodes) is
	// placed. Optional; nil means the datadir is used as initialized.
	ProvisionKeys func(ctx context.Context, spec process.NodeSpec, producer bool) error
	// Overrides, if set, returns a node's high-precedence launch knobs, applied
	// through the launchopt Builder after the standard and family layers. It is
	// where account-specific and RPC-namespace knobs go: the producer's
	// unlock/etherbase/password, and the http.api set (admin is required for the
	// mesh's admin_addPeer). Typed keys, so an unsupported knob is a classified
	// assembly error instead of a silently ignored flag. Optional.
	Overrides func(spec NodeSpec, producer bool) []nodeconfig.Override
	// Only, when non-empty, restricts the launch to these 0-based node
	// positions. It is how a caller launches in phases: a poa network's etcd
	// cluster forms only while its producer is alone, so the producer goes up,
	// the bootstrap runs, and the rest follow. Empty launches every node, which
	// is what a family with nothing to order asks for.
	Only []int
	// Machine, when set, resolves which machine node i runs on: its file store
	// and its driver. It wins over Files and over the driver passed to Launch.
	//
	// It exists because a handoff spread across a server set is not one machine.
	// A driver is bound to the host its transport was built for and never reads
	// NodeSpec.Host, so without this every node launches wherever the single
	// driver points — measured: a five-node placement across five servers put all
	// five datadirs and the one surviving process on the first server, the rest
	// colliding on its ports. The composition path resolves a machine per node
	// (chainsetup.machineFor) and this is the same seam.
	Machine func(index int) (filestore.Store, process.Driver, error)
	// Files is where the shared genesis is written. Nil is the local
	// filesystem, which is what a local handoff wants and what this used to do
	// unconditionally — the boundary exists so a caller running against a remote
	// target can send the genesis where the nodes are rather than to the
	// machine driving them — the defect the remote-provision path used to
	// have, where a remote network's files landed on the operator's resource.
	Files filestore.Store
}

// wants reports whether position i is in this launch.
func (o LaunchOptions) wants(i int) bool { return selected(o.Only, i) }

// machineFor is where node i's files and processes go: its own machine when the
// caller resolves one, the single pair otherwise.
func (o LaunchOptions) machineFor(i int, d process.Driver) (filestore.Store, process.Driver, error) {
	if o.Machine == nil {
		return o.files(), d, nil
	}
	return o.Machine(i)
}

// selected reports whether a phase that names these 0-based positions includes
// this one; naming none means all of them.
//
// One rule, because two callers depend on agreeing exactly: the launcher uses
// it to decide what to start, and the vacancy check uses it to decide whose
// ports to check. Were they to disagree, the check would clear the ports of
// nodes that are not starting while missing the ones that are.
func selected(only []int, i int) bool {
	if len(only) == 0 {
		return true
	}
	for _, n := range only {
		if n == i {
			return true
		}
	}
	return false
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
func BuildNodeSpecs(plan Plan, opts LaunchOptions) ([]process.NodeSpec, error) {
	specs, _, err := buildLaunch(plan, opts)
	return specs, err
}

// buildLaunch returns both halves of a launch: the driver specs and the
// configuration each was rendered from.
//
// One Spec per node feeds both renderers, so the file and the command line
// cannot disagree about the same node. Before this the handoff rendered only
// argv and wrote no file at all, which is why its nodekey and its static nodes
// had to sit where each binary looks for them by convention — and why one of
// the two binaries ignored the file it was given.
func buildLaunch(plan Plan, opts LaunchOptions) ([]process.NodeSpec, []nodeconfig.Spec, error) {
	if opts.FromBinary == "" || opts.ToBinary == "" {
		return nil, nil, fmt.Errorf("upgrade: both from and to binaries must be set")
	}
	if opts.FromChain == nil || opts.ToChain == nil {
		return nil, nil, fmt.Errorf("upgrade: both from and to chains must be set")
	}
	specs := make([]process.NodeSpec, 0, len(plan.Nodes))
	configs := make([]nodeconfig.Spec, 0, len(plan.Nodes))
	for _, n := range plan.Nodes {
		binary, chain := opts.ToBinary, opts.ToChain
		if n.Producer {
			binary, chain = opts.FromBinary, opts.FromChain
		}
		num := n.Index + 1
		dataDir := filepath.Join(opts.DataRoot, fmt.Sprintf("node%d", num))
		configPath := filepath.Join(opts.DataRoot, fmt.Sprintf("config_node%d.toml", num))
		// The node's own host when the plan placed it on one; the launch's single
		// host otherwise. A handoff over a server set has a host per node.
		host := n.Host
		if host == "" {
			host = opts.host()
		}
		httpHost := "127.0.0.1"
		if n.Host != "" {
			httpHost = "0.0.0.0"
		}
		cfg := nodeconfig.Spec{
			// Through the node's own plugin, so the RPC namespace, the flag
			// vocabulary and the miner's recommit form are the ones ITS binary
			// speaks. The network id is the plan's: a handoff's devp2p id is not
			// either manifest's.
			Chain:       chainFactsOf(chain, n),
			Role:        n.Role,
			Ports:       n.Ports,
			DataDir:     dataDir,
			ConfigPath:  configPath,
			NodekeyPath: filepath.Join(dataDir, "nodekey"),
			KeystoreDir: filepath.Join(dataDir, "keystore"),
			StaticNodes: opts.StaticNodes,
			HTTPHost:    httpHost,
		}
		if opts.Identity != nil {
			cfg.Unlock, cfg.PasswordFile = opts.Identity(n, n.Producer)
		}
		var overrides []nodeconfig.Override
		if opts.Overrides != nil {
			overrides = opts.Overrides(n, n.Producer)
		}
		args, err := nodeconfig.Argv(cfg, overrides...)
		if err != nil {
			return nil, nil, fmt.Errorf("upgrade: node%d: %w", num, err)
		}
		specs = append(specs, process.NodeSpec{
			Index:      n.Index,
			Role:       n.Role,
			Host:       host,
			RPCURL:     n.RPCURL,
			Binary:     binary,
			DataDir:    dataDir,
			ConfigPath: configPath,
			LogPath:    filepath.Join(opts.DataRoot, "logs", fmt.Sprintf("node%d.log", num)),
			Args:       args,
			Ports:      n.Ports,
		})
		configs = append(configs, cfg)
	}
	return specs, configs, nil
}

// chainFactsOf is one node's chain facts: its own plugin's answers, with the
// plan's devp2p network id over the manifest's. A handoff runs one network on
// two chains, and the id belongs to the network.
func chainFactsOf(plugin registry.ChainPlugin, n NodeSpec) nodeconfig.Chain {
	c := nodeconfig.ChainOf(plugin, n.Role)
	c.NetworkID = n.NetworkID
	if n.Chain != "" {
		c.ID = n.Chain
	}
	return c
}

// Launch runs a handoff network: it writes the shared genesis, initializes each
// node's datadir with that node's own binary (so go-wemix and go-wbft each lay
// out their chaindata correctly from identical genesis bytes), then provisions
// and launches every node concurrently through the process. Producers and
// validators run at the same time — this is a concurrent handoff, not a binary
// swap: producers mine up to the fork, validators sync and take over after it.
// It returns the launched NodeSet.
func Launch(ctx context.Context, d process.Driver, plan Plan, opts LaunchOptions) (node.NodeSet, error) {
	initFn := opts.InitFn
	if initFn == nil {
		initFn = process.InitDatadir
	}
	ns := node.NodeSet{Chain: plan.To.ID, Network: "local"}

	genesisPath := filepath.Join(opts.DataRoot, "genesis.json")

	specs, configs, err := buildLaunch(plan, opts)
	if err != nil {
		return ns, err
	}
	// The genesis goes to every machine that will hold a node, not once.
	//
	// One write was right while the whole network shared a filesystem. Across a
	// server set each machine needs its own copy, because the binary that reads it
	// runs over there. Machines are told apart by the node's HOST rather than by
	// its store: a store is an interface value that need not be comparable (the
	// remote one is a struct holding funcs, and using it as a map key panics), and
	// the host is the fact that actually says which machine this is.
	written := map[string]bool{}
	for i := range specs {
		if !opts.wants(i) {
			continue
		}
		at := specs[i].Host
		if written[at] {
			continue
		}
		files, _, merr := opts.machineFor(i, d)
		if merr != nil {
			return ns, fmt.Errorf("upgrade: node%d machine: %w", i+1, merr)
		}
		if err := files.Write(ctx, genesisPath, plan.Genesis, 0o644); err != nil {
			return ns, fmt.Errorf("upgrade: write genesis for node%d: %w", i+1, err)
		}
		written[at] = true
	}
	// Each node's config goes to its own machine, beside the genesis. It holds
	// the node's keystore directory, its endpoints, its RPC modules and its
	// static peers — the facts a geth-family binary reads from a file rather
	// than from argv, and the ones this launch used to leave to each binary's
	// own conventions.
	for i := range specs {
		if !opts.wants(i) {
			continue
		}
		files, _, merr := opts.machineFor(i, d)
		if merr != nil {
			return ns, fmt.Errorf("upgrade: node%d machine: %w", i+1, merr)
		}
		if werr := files.Write(ctx, specs[i].ConfigPath, nodeconfig.TOML(configs[i]), 0o644); werr != nil {
			return ns, fmt.Errorf("upgrade: write config for node%d: %w", i+1, werr)
		}
	}
	// A driver that can initialize a datadir itself is asked to. The remote
	// driver is one — it ships the genesis and runs `init` on the host — and
	// skipping the question is why a remote handoff tried to mkdir the target's
	// data root on the operator's own machine. The local fallback stays for a
	// driver that has no opinion, and an explicit InitFn still wins so a test can
	// substitute one.
	for i, spec := range specs {
		if !opts.wants(i) {
			continue
		}
		// This node's machine: its own when the caller resolves one, the single
		// driver otherwise. A driver never reads spec.Host, so asking here is the
		// only thing that sends a node to the server the plan placed it on.
		_, nodeDriver, merr := opts.machineFor(i, d)
		if merr != nil {
			return ns, fmt.Errorf("upgrade: node%d machine: %w", spec.Index+1, merr)
		}
		init, canInit := nodeDriver.(process.Initializer)
		switch {
		case opts.InitFn != nil:
			if err := opts.InitFn(ctx, spec.Binary, spec.DataDir, genesisPath); err != nil {
				return ns, fmt.Errorf("upgrade: init node%d (%s): %w", spec.Index+1, spec.Binary, err)
			}
		case canInit:
			if err := init.InitDatadir(ctx, spec, plan.Genesis); err != nil {
				return ns, fmt.Errorf("upgrade: init node%d (%s) on the target: %w", spec.Index+1, spec.Binary, err)
			}
		default:
			if err := initFn(ctx, spec.Binary, spec.DataDir, genesisPath); err != nil {
				return ns, fmt.Errorf("upgrade: init node%d (%s): %w", spec.Index+1, spec.Binary, err)
			}
		}
		if opts.ProvisionKeys != nil {
			if err := opts.ProvisionKeys(ctx, spec, plan.Nodes[i].Producer); err != nil {
				return ns, fmt.Errorf("upgrade: provision keys node%d: %w", spec.Index+1, err)
			}
		}
		if err := nodeDriver.Provision(ctx, spec); err != nil {
			return ns, fmt.Errorf("upgrade: provision node%d: %w", spec.Index+1, err)
		}
		h, err := nodeDriver.Launch(ctx, spec)
		if err != nil {
			return ns, fmt.Errorf("upgrade: launch node%d (%s): %w", spec.Index+1, spec.Binary, err)
		}
		ns.Nodes = append(ns.Nodes, process.NodeOf(spec, h.PID))
	}
	return ns, nil
}
