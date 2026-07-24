package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/genesis"
	"github.com/0xmhha/chainbench/pkg/core/keys"
	"github.com/0xmhha/chainbench/pkg/core/nodeconfig"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

func newSetupCmd() *cobra.Command {
	var (
		chain      string
		validators int
		endpoints  int
		dataDir    string
		keysDir    string
		binaryPath string
		provision  bool
		launch     bool
		dryRun     bool
	)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Plan (and, when wired, launch) a local chain network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := registry.Get(chain)
			if err != nil {
				return err
			}
			override := config.Values{}
			if cmd.Flags().Changed("validators") {
				override["nodes.validators"] = strconv.Itoa(validators)
			}
			if cmd.Flags().Changed("endpoints") {
				override["nodes.endpoints"] = strconv.Itoa(endpoints)
			}
			cfg := config.Resolve(nil, override)

			root := dataDir
			if !filepath.IsAbs(root) {
				root = filepath.Clean(root)
			}
			plan, err := setup.BuildPlan(cfg, p, root)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "chain:    %s (family %s, binary %s, chain_id %d)\n",
				p.Manifest().ID, p.Manifest().ConsensusFamily, p.Manifest().Binary, p.Manifest().ChainID)
			fmt.Fprintf(out, "network:  %s\n", plan.Network)
			fmt.Fprintf(out, "dataRoot: %s\n", plan.DataRoot)
			hasTmpl := len(p.GenesisTemplate()) > 0
			fmt.Fprintf(out, "genesis:  template=%v (engine=%q)\n", hasTmpl, p.Manifest().Genesis.EngineField)

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tROLE\tHOST\tP2P\tHTTP\tWS")
			for _, n := range plan.Nodes {
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\n",
					n.Index, n.Role, n.Host, n.Ports.P2P, n.Ports.HTTP, n.Ports.WS)
			}
			if err := w.Flush(); err != nil {
				return err
			}

			if provision || launch {
				preset, err := keys.LoadPreset(keysDir)
				if err != nil {
					return err
				}
				sub := preset.Take(cfg.Int("nodes.validators", len(preset.Validators)))
				gen, err := genesis.Build(p, genesis.Inputs{
					Validators: sub.Validators,
					BLSKeys:    sub.BLSKeys,
					ExtraData:  sub.ExtraData,
				})
				if err != nil {
					return err
				}
				if err := os.MkdirAll(plan.DataRoot, 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(plan.GenesisPath, gen, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(out, "genesis written: %s (%d validators)\n", plan.GenesisPath, len(sub.Validators))

				// Per-node TOML configs. Static nodes are every preset node's
				// enode at its planned p2p port.
				var staticNodes []string
				for _, spec := range plan.Nodes {
					if nk := nodeKeyFor(preset, spec.Index); nk != nil {
						staticNodes = append(staticNodes, nodeconfig.Enode(nk.PublicKey, spec.Host, spec.Ports.P2P))
					}
				}
				ns := p.Manifest().Consensus.RPCNamespace
				for _, spec := range plan.Nodes {
					toml := nodeconfig.Generate(nodeconfig.Params{
						Role:         spec.Role,
						Ports:        spec.Ports,
						KeystoreDir:  filepath.Join(plan.DataRoot, "keystores", fmt.Sprintf("node%d", spec.Index)),
						RPCNamespace: ns,
						StaticNodes:  staticNodes,
					})
					if err := os.WriteFile(spec.ConfigPath, toml, 0o644); err != nil {
						return err
					}
				}
				fmt.Fprintf(out, "configs written: %d node TOML files\n", len(plan.Nodes))
			}

			if launch {
				bin, err := resolveBinary(binaryPath, p.Manifest().Binary)
				if err != nil {
					return err
				}
				ctx := cmd.Context()
				for i := range plan.Nodes {
					plan.Nodes[i].Binary = bin
					if err := driver.InitDatadir(ctx, bin, plan.Nodes[i].DataDir, plan.GenesisPath); err != nil {
						return err
					}
				}
				plan.Genesis = nil // already written by the provision step above
				ns, err := setup.Run(ctx, plan, driver.NewLocalDriver(), nil)
				if err != nil {
					return err
				}
				if err := state.SaveNodeSet(plan.DataRoot, ns); err != nil {
					return err
				}
				fmt.Fprintf(out, "launched %d node(s); state: %s\n",
					len(ns.Nodes), filepath.Join(plan.DataRoot, "nodeset.json"))
				return nil
			}

			if !dryRun {
				return fmt.Errorf("live launch needs --launch (with --binary or a %s on PATH). Use --provision to write artifacts, --dry-run to plan", p.Manifest().Binary)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().IntVar(&validators, "validators", 0, "override validator count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "override endpoint count")
	cmd.Flags().StringVar(&dataDir, "data-dir", "data", "data root directory")
	cmd.Flags().StringVar(&keysDir, "keys-dir", "keys/preset", "preset keys directory (for --provision)")
	cmd.Flags().StringVar(&binaryPath, "binary", "", "node binary path (for --launch); default: chain binary on PATH")
	cmd.Flags().BoolVar(&provision, "provision", false, "write genesis.json + node configs from preset keys")
	cmd.Flags().BoolVar(&launch, "launch", false, "init datadirs and launch the nodes (implies --provision)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not launch")
	return cmd
}

// resolveBinary returns the executable path for launch: the explicit path if
// given, otherwise the chain's binary looked up on PATH.
func resolveBinary(explicit, chainBinary string) (string, error) {
	name := explicit
	if name == "" {
		name = chainBinary
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find node binary %q: %w (build it or pass --binary)", name, err)
	}
	return path, nil
}

// nodeKeyFor returns the preset node key for a 1-based node index, or nil.
func nodeKeyFor(p keys.Preset, index int) *keys.NodeKey {
	for i := range p.Nodes {
		if p.Nodes[i].Index == index {
			return &p.Nodes[i]
		}
	}
	return nil
}
