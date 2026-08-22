package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
)

// genesisFilePerm is the genesis.json file mode.
const genesisFilePerm os.FileMode = 0o644

// configFilePerm is the node config file mode.
const configFilePerm os.FileMode = 0o644

// LocalLauncher arms and launches a plan on the local host from a preset key
// set. For each node it renders the config, installs the preset identity
// (devp2p nodekey, and for validators the unlock account), initializes the
// datadir from the network genesis, and launches the process. It implements the
// supervisor launch seam, so NewBuildEnv brings a real network up on the
// allocator-assigned ports.
//
// On-disk files (genesis, per-node config) are materialized through a
// provision.FileStore (upload-if-absent), so a rerun reuses existing files and a
// remote sink can later ship them to another host without changing this type.
type LocalLauncher struct {
	// Peering selects the peer graph the nodes are wired into. The zero value
	// is netmap.Mesh, which is what every composition did before the policy had
	// a name.
	Peering netmap.Peering

	// Plugin is the target chain (supplies the RPC namespace and miner recommit
	// form for config rendering).
	Plugin registry.ChainPlugin
	// Binary is the resolved node executable path.
	Binary string
	// KeysDir holds the preset (metadata.json + node<i>/ identity dirs +
	// password).
	KeysDir string
	// Driver launches the nodes; nil defaults to the local driver.
	Driver driver.Driver
	// Files materializes on-disk files; nil defaults to the local filesystem.
	Files provision.FileStore
	// LaunchOverrides are high-precedence launch knobs (env.launch / case
	// layers) applied to every node's argv after the role-derived modules.
	LaunchOverrides []launchopt.Override
}

// Launch arms and launches every node in plan and returns the running node set
// plus the processes to track for teardown.
func (l LocalLauncher) Launch(ctx context.Context, plan driver.Plan) (supervisor.LaunchResult, error) {
	res, _, err := l.LaunchArmed(ctx, plan)
	return res, err
}

// LaunchArmed is Launch plus the armed node specs it used. A caller that has to
// relaunch one node later (fault injection) needs its exact arming — config
// path, identity flags, datadir — and re-deriving it would risk launching a
// subtly different node than the one that was stopped.
//
// It is the composition of the three exported phases below. They are separate so
// a caller checking the bring-up can report each one on its own: "which step
// failed" is a different question from "did it come up", and only the phases
// answer it.
func (l LocalLauncher) LaunchArmed(ctx context.Context, plan driver.Plan) (supervisor.LaunchResult, []driver.NodeSpec, error) {
	specs, err := l.Arm(plan)
	if err != nil {
		return supervisor.LaunchResult{}, nil, err
	}
	if err := l.Materialize(ctx, plan, specs); err != nil {
		return supervisor.LaunchResult{}, specs, err
	}
	res, err := l.InitAndLaunch(ctx, plan, specs)
	return res, specs, err
}

// Arm loads the preset and produces each node's launch spec: rendered config,
// identity flags, resolved binary. Pure apart from reading the preset. Identity
// paths (nodekey, keystore, password) are rooted at keyBase(plan): the local
// KeysDir for a local launch, or the remote keys dir a remote launch ships them
// to.
func (l LocalLauncher) Arm(plan driver.Plan) ([]driver.NodeSpec, error) {
	preset, err := keyring.LoadPreset(l.KeysDir)
	if err != nil {
		return nil, fmt.Errorf("engine: launcher: %w", err)
	}
	return armSpecs(l.Plugin, preset, plan, l.Binary, l.keyBase(plan), l.Peering, l.LaunchOverrides)
}

// Materialize writes the genesis and per-node config through the file sink
// (upload-if-absent), locally or to a remote host. When the driver ships files
// to another host (a FileProvisioner), each node's preset identity files are
// also shipped to the remote keys dir the armed specs reference.
func (l LocalLauncher) Materialize(ctx context.Context, plan driver.Plan, specs []driver.NodeSpec) error {
	sink := l.Files
	if sink == nil {
		sink = provision.LocalFileStore{}
	}
	if err := materialize(ctx, provision.New(sink), plan, specs); err != nil {
		return err
	}
	if fp, remote := l.fileProvisioner(); remote {
		if err := shipIdentities(ctx, fp, l.KeysDir, l.keyBase(plan), specs); err != nil {
			return fmt.Errorf("engine: launcher: ship identities: %w", err)
		}
	}
	return nil
}

// Provision arms the plan and materializes its on-disk files (genesis, per-node
// config, and — for a remote driver — the shipped identities) without
// initializing datadirs or launching. It is the provision-only path behind
// `setup --provision`. The returned specs are the armed specs materialize wrote.
func (l LocalLauncher) Provision(ctx context.Context, plan driver.Plan) ([]driver.NodeSpec, error) {
	specs, err := l.Arm(plan)
	if err != nil {
		return nil, err
	}
	if err := l.Materialize(ctx, plan, specs); err != nil {
		return specs, err
	}
	return specs, nil
}

// fileProvisioner reports whether the launcher's driver ships files to another
// host (the remote-launch case) and, if so, returns that capability.
func (l LocalLauncher) fileProvisioner() (driver.FileProvisioner, bool) {
	if l.Driver == nil {
		return nil, false
	}
	fp, ok := l.Driver.(driver.FileProvisioner)
	return fp, ok
}

// keyBase is where a node's identity files (nodekey, keystore, password) live at
// launch: the local KeysDir for a local launch, or a keys/ dir under the data
// root for a remote launch (where shipIdentities places them, and where the
// remote config keystore and launch args then point).
func (l LocalLauncher) keyBase(plan driver.Plan) string {
	if _, remote := l.fileProvisioner(); remote {
		return filepath.Join(plan.DataRoot, "keys")
	}
	return l.KeysDir
}

// shipIdentities copies each node's preset identity files — the devp2p nodekey,
// the validator keystore, and the shared password — from the local keysDir to
// keyBase on the driver's host via the FileProvisioner, so a remote launch finds
// them at the paths its config keystore and launch args reference. A local
// launch never calls it (keyBase == keysDir).
func shipIdentities(ctx context.Context, fp driver.FileProvisioner, keysDir, keyBase string, specs []driver.NodeSpec) error {
	if pw, err := os.ReadFile(filepath.Join(keysDir, "password")); err == nil {
		if err := fp.ProvisionFile(ctx, filepath.Join(keyBase, "password"), pw, 0o600); err != nil {
			return err
		}
	}
	for _, spec := range specs {
		src := filepath.Join(keysDir, fmt.Sprintf("node%d", spec.Index))
		dst := filepath.Join(keyBase, fmt.Sprintf("node%d", spec.Index))
		if nk, err := os.ReadFile(filepath.Join(src, "nodekey")); err == nil {
			if err := fp.ProvisionFile(ctx, filepath.Join(dst, "nodekey"), nk, 0o600); err != nil {
				return err
			}
		}
		entries, err := os.ReadDir(filepath.Join(src, "keystore"))
		if err != nil {
			continue // no keystore (e.g. an endpoint node)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			b, err := os.ReadFile(filepath.Join(src, "keystore", e.Name()))
			if err != nil {
				return err
			}
			if err := fp.ProvisionFile(ctx, filepath.Join(dst, "keystore", e.Name()), b, 0o600); err != nil {
				return err
			}
		}
	}
	return nil
}

// InitAndLaunch initializes each node's datadir from the shared genesis and
// starts it, returning the node set and the processes to track.
func (l LocalLauncher) InitAndLaunch(ctx context.Context, plan driver.Plan, specs []driver.NodeSpec) (supervisor.LaunchResult, error) {
	d := l.Driver
	if d == nil {
		d = driver.NewLocalDriver()
	}
	// Init through the driver's Initializer capability when present (both the
	// local and remote drivers implement it), so a remote driver ships the
	// genesis and runs init on its host; otherwise fall back to a local init.
	initer, canInit := d.(driver.Initializer)

	res := supervisor.LaunchResult{Nodes: node.NodeSet{
		Chain: plan.Chain, Network: plan.Network, Capabilities: plan.Capabilities,
	}}
	for _, spec := range specs {
		if canInit {
			if err := initer.InitDatadir(ctx, spec, plan.Genesis); err != nil {
				return res, fmt.Errorf("engine: launcher: init node%d: %w", spec.Index, err)
			}
		} else if err := driver.InitDatadir(ctx, l.Binary, spec.DataDir, plan.GenesisPath); err != nil {
			return res, fmt.Errorf("engine: launcher: init node%d: %w", spec.Index, err)
		}
		h, err := d.Launch(ctx, spec)
		if err != nil {
			return res, fmt.Errorf("engine: launcher: launch node%d: %w", spec.Index, err)
		}
		res.Nodes.Nodes = append(res.Nodes.Nodes, node.Node{
			Index: spec.Index, Role: spec.Role, Host: spec.Host,
			RPCURL: fmt.Sprintf("http://%s:%d", spec.Host, spec.Ports.HTTP),
			Ports:  spec.Ports, PID: h.PID,
		})
		res.Procs = append(res.Procs, procman.Proc{
			PID: h.PID, Label: fmt.Sprintf("node%d", spec.Index),
			DataDir: spec.DataDir, Host: spec.Host,
		})
	}
	return res, nil
}

// materialize writes the network genesis and each node's rendered config
// through the provisioner (upload-if-absent, so a rerun reuses existing files).
// Identity files (nodekey, keystore, password) already live in the preset dir
// and are referenced by path rather than copied.
func materialize(ctx context.Context, pv *provision.Provisioner, plan driver.Plan, specs []driver.NodeSpec) error {
	if len(plan.Genesis) > 0 {
		in := provision.NodeInputs{
			DataDir: plan.DataRoot,
			Files:   []provision.File{{Path: filepath.Base(plan.GenesisPath), Content: plan.Genesis, Mode: genesisFilePerm}},
		}
		if _, err := pv.Provision(ctx, in); err != nil {
			return fmt.Errorf("engine: launcher: genesis: %w", err)
		}
	}
	for _, spec := range specs {
		if len(spec.ConfigContent) == 0 {
			continue
		}
		in := provision.NodeInputs{
			DataDir: filepath.Dir(spec.ConfigPath),
			Files:   []provision.File{{Path: filepath.Base(spec.ConfigPath), Content: spec.ConfigContent, Mode: configFilePerm}},
		}
		if _, err := pv.Provision(ctx, in); err != nil {
			return fmt.Errorf("engine: launcher: config node%d: %w", spec.Index, err)
		}
	}
	return nil
}

// armSpecs fills each plan node spec with its rendered config, its full launch
// argv, and its resolved binary. It is pure (no I/O) so the arming — the part
// most prone to mistakes (wrong static-node ports, missing unlock) — is
// unit-testable. Static-node enodes use the plan's (allocator-assigned) p2p
// ports so peering matches the launched layout.
//
// The argv is assembled here and only here (launchopt Builder), replacing the
// previous split between AssemblePlan's common flags and this function's
// identity appends — see docs/dev/architecture/code-graph.md §3.
func armSpecs(plugin registry.ChainPlugin, preset keyring.Preset, plan driver.Plan, binary, keysDir string, peering netmap.Peering, overrides []launchopt.Override) ([]driver.NodeSpec, error) {
	// Who a node dials is netmap's policy now; this function only knows how to
	// spell a peer, because an enode needs key material and netmap holds none.
	placed, err := PlanMap(plan)
	if err != nil {
		return nil, fmt.Errorf("engine: launcher: %w", err)
	}
	if peering == "" {
		peering = netmap.Mesh
	}
	enode := func(pl netmap.Placement) (string, bool) {
		nk, ok := preset.Node(pl.Index)
		if !ok {
			return "", false
		}
		return nodeconfig.Enode(nk.PublicKey, pl.Host, pl.Ports.P2P), true
	}

	m := plugin.Manifest()
	out := make([]driver.NodeSpec, len(plan.Nodes))
	for i, spec := range plan.Nodes {
		staticNodes, err := peering.StaticNodes(placed, netmap.LabelFor(spec.Index), enode)
		if err != nil {
			return nil, fmt.Errorf("engine: launcher: node%d peers: %w", spec.Index, err)
		}
		nodeDir := filepath.Join(keysDir, fmt.Sprintf("node%d", spec.Index))
		spec.ConfigContent = nodeconfig.Generate(nodeconfig.Params{
			Role:          spec.Role,
			Ports:         spec.Ports,
			KeystoreDir:   filepath.Join(nodeDir, "keystore"),
			RPCNamespace:  m.Consensus.RPCNamespace,
			MinerRecommit: m.MinerRecommit,
			SyncMode:      spec.SyncMode,
			StaticNodes:   staticNodes,
		})

		args, err := NodeLaunchArgs(plugin, preset, spec, keysDir, overrides)
		if err != nil {
			return nil, fmt.Errorf("engine: launcher: node%d: %w", spec.Index, err)
		}
		spec.Args = args
		spec.Binary = binary
		out[i] = spec
	}
	return out, nil
}

// NodeLaunchArgs assembles one node's full launch argv through the launchopt
// Builder. This is THE argv assembly site (docs/dev/architecture/code-graph.md
// §3): armSpecs uses it for engine runs, and the netcompose step surface uses
// it for `net launchopts`/`net start`, so a step-composed node launches with
// exactly the argv an engine run would produce.
func NodeLaunchArgs(plugin registry.ChainPlugin, preset keyring.Preset, spec driver.NodeSpec, keysDir string, overrides []launchopt.Override) ([]string, error) {
	policy, err := launchopt.ParseFamilyFlags(plugin.Family().StartFlags(spec.Role))
	if err != nil {
		return nil, err
	}
	nodeDir := filepath.Join(keysDir, fmt.Sprintf("node%d", spec.Index))
	id := launchopt.Identity{
		NodeKeyFile:         filepath.Join(nodeDir, "nodekey"),
		AllowInsecureUnlock: policy.AllowInsecureUnlock,
	}
	if netmap.Is(spec.Role, node.RoleBP) {
		if nk, ok := preset.Node(spec.Index); ok {
			id.Unlock = nk.Address
			id.PasswordFile = filepath.Join(keysDir, "password")
			id.Etherbase = nk.Address
		}
	}
	return launchopt.New(launchopt.DialectFor(plugin.Manifest().ID),
		id,
		launchopt.Storage{DataDir: spec.DataDir, ConfigFile: spec.ConfigPath},
		// The manifest's network id is emitted rather than left to the
		// binary's default. It was never emitted at all, so a chain whose
		// devp2p network id differs from its genesis chain id — which the
		// handoff produces, since it forces the chain id — ran on whichever
		// the binary inferred. Setting it from the manifest makes the argv say
		// what the chain declares; an operator's --network-id still wins,
		// because that override arrives on a later layer.
		launchopt.P2P{Port: spec.Ports.P2P, NetworkID: plugin.Manifest().NetworkID},
		launchopt.HTTPRPC{Enabled: true, Port: spec.Ports.HTTP},
		launchopt.WSRPC{Enabled: true, Port: spec.Ports.WS},
		launchopt.RPCPolicy{DeprecatedPersonal: policy.DeprecatedPersonal, UnprotectedTxs: policy.UnprotectedTxs},
		launchopt.Mining{Mine: policy.Mine},
	).WithOverrides(overrides...).Build()
}

// PlanMap reads a launch plan as a placement map, so the peering policy and the
// address lookups run off one representation instead of each caller walking
// plan.Nodes its own way.
//
// The plan's port set has no etcd port (node.Endpoints predates netmap.Ports,
// NM-b), so the map carries what the plan knows and no more — the derived etcd
// port arrives when the port type is swapped.
func PlanMap(plan driver.Plan) (*netmap.Map, error) {
	placements := make([]netmap.Placement, 0, len(plan.Nodes))
	ordinals := map[node.Role]int{}
	for _, spec := range plan.Nodes {
		role, err := netmap.NormalizeRole(string(spec.Role))
		if err != nil {
			return nil, fmt.Errorf("node%d: %w", spec.Index, err)
		}
		ordinals[role]++
		placements = append(placements, netmap.Placement{
			Index: spec.Index,
			Label: netmap.LabelFor(spec.Index),
			Role:  role,
			Ord:   ordinals[role],
			Host:  spec.Host,
			Ports: netmap.Ports{
				P2P: spec.Ports.P2P, HTTP: spec.Ports.HTTP, WS: spec.Ports.WS,
				Auth: spec.Ports.Auth, Metrics: spec.Ports.Metrics,
			},
			DataDir: spec.DataDir,
		})
	}
	return netmap.NewMap(placements)
}
