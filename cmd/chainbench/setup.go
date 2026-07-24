package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/genesis"
	"github.com/0xmhha/chainbench/pkg/core/keys"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

func newSetupCmd() *cobra.Command {
	var (
		chain      string
		validators int
		endpoints  int
		dataDir    string
		keysDir    string
		provision  bool
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

			if provision {
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
			}

			if !dryRun {
				return fmt.Errorf("live launch is not yet wired: it needs per-node config + a built %s binary (network absorption). Use --provision to write genesis, --dry-run to plan", p.Manifest().Binary)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().IntVar(&validators, "validators", 0, "override validator count")
	cmd.Flags().IntVar(&endpoints, "endpoints", 0, "override endpoint count")
	cmd.Flags().StringVar(&dataDir, "data-dir", "data", "data root directory")
	cmd.Flags().StringVar(&keysDir, "keys-dir", "keys/preset", "preset keys directory (for --provision)")
	cmd.Flags().BoolVar(&provision, "provision", false, "write genesis.json from preset keys")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not launch")
	return cmd
}
