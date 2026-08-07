package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
)

// genesisFilePerm is the genesis.json file mode.
const genesisFilePerm os.FileMode = 0o644

// dataRootPerm is the data-root directory mode.
const dataRootPerm os.FileMode = 0o755

// LocalLauncher arms and launches a plan on the local host from a preset key
// set. For each node it renders the config, installs the preset identity
// (devp2p nodekey, and for validators the unlock account), initializes the
// datadir from the network genesis, and launches the process. It implements the
// supervisor launch seam, so NewBuildEnv brings a real network up on the
// allocator-assigned ports.
type LocalLauncher struct {
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
}

// Launch arms and launches every node in plan and returns the running node set
// plus the processes to track for teardown.
func (l LocalLauncher) Launch(ctx context.Context, plan setup.Plan) (supervisor.LaunchResult, error) {
	preset, err := keys.LoadPreset(l.KeysDir)
	if err != nil {
		return supervisor.LaunchResult{}, fmt.Errorf("engine: launcher: %w", err)
	}
	if err := os.MkdirAll(plan.DataRoot, dataRootPerm); err != nil {
		return supervisor.LaunchResult{}, fmt.Errorf("engine: launcher: mkdir data root: %w", err)
	}
	if len(plan.Genesis) > 0 {
		if err := os.WriteFile(plan.GenesisPath, plan.Genesis, genesisFilePerm); err != nil {
			return supervisor.LaunchResult{}, fmt.Errorf("engine: launcher: write genesis: %w", err)
		}
	}

	specs := armSpecs(l.Plugin, preset, plan, l.Binary, l.KeysDir)

	d := l.Driver
	if d == nil {
		d = driver.NewLocalDriver()
	}

	res := supervisor.LaunchResult{Nodes: node.NodeSet{
		Chain: plan.Chain, Network: plan.Network, Capabilities: plan.Capabilities,
	}}
	for _, spec := range specs {
		if err := d.Provision(ctx, spec); err != nil {
			return res, fmt.Errorf("engine: launcher: provision node%d: %w", spec.Index, err)
		}
		if err := driver.InitDatadir(ctx, l.Binary, spec.DataDir, plan.GenesisPath); err != nil {
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

// armSpecs fills each plan node spec with its rendered config, preset identity
// launch args, and resolved binary. It is pure (no I/O) so the arming — the
// part most prone to mistakes (wrong static-node ports, missing unlock) — is
// unit-testable. Static-node enodes use the plan's (allocator-assigned) p2p
// ports so peering matches the launched layout.
func armSpecs(plugin registry.ChainPlugin, preset keys.Preset, plan setup.Plan, binary, keysDir string) []driver.NodeSpec {
	staticNodes := make([]string, 0, len(plan.Nodes))
	for _, spec := range plan.Nodes {
		if nk, ok := preset.Node(spec.Index); ok {
			staticNodes = append(staticNodes, nodeconfig.Enode(nk.PublicKey, spec.Host, spec.Ports.P2P))
		}
	}

	m := plugin.Manifest()
	out := make([]driver.NodeSpec, len(plan.Nodes))
	for i, spec := range plan.Nodes {
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
		spec.Args = append(spec.Args, "--nodekey", filepath.Join(nodeDir, "nodekey"))
		if spec.Role == node.RoleValidator {
			if nk, ok := preset.Node(spec.Index); ok {
				spec.Args = append(spec.Args,
					"--unlock", nk.Address,
					"--password", filepath.Join(keysDir, "password"),
					"--miner.etherbase", nk.Address,
				)
			}
		}
		spec.Binary = binary
		out[i] = spec
	}
	return out
}
