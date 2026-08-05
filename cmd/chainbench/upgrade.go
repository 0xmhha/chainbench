package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// newUpgradeCmd drives the concurrent consensus-family handoff (go-wemix+etcd ->
// go-wbft) framework in pkg/consensus/upgrade from a golden profile. Unlike the
// `hardfork` command (an in-place binary swap for a homogeneous fork), this
// composes a plan where producers and validators run concurrently.
func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Plan a concurrent consensus-family handoff from a golden profile",
	}
	cmd.AddCommand(newUpgradeGenesisCmd(), newUpgradeRunCmd())
	return cmd
}

// buildPlanFromProfile is the shared front half: load the golden profile, read
// the from-chain base genesis, resolve both chain plugins, and build the plan.
func buildPlanFromProfile(profilePath, fromGenesisPath string) (upgrade.Plan, error) {
	p, err := upgrade.LoadProfile(profilePath)
	if err != nil {
		return upgrade.Plan{}, err
	}
	fromGenesis, err := os.ReadFile(fromGenesisPath)
	if err != nil {
		return upgrade.Plan{}, fmt.Errorf("read from-genesis: %w", err)
	}
	in, err := p.Inputs(fromGenesis)
	if err != nil {
		return upgrade.Plan{}, err
	}
	from, err := registry.Get(p.Upgrade.From)
	if err != nil {
		return upgrade.Plan{}, fmt.Errorf("from-chain %q: %w", p.Upgrade.From, err)
	}
	to, err := registry.Get(p.Upgrade.To)
	if err != nil {
		return upgrade.Plan{}, fmt.Errorf("to-chain %q: %w", p.Upgrade.To, err)
	}
	return upgrade.BuildPlan(from, to, in)
}

func newUpgradeGenesisCmd() *cobra.Command {
	var profilePath, fromGenesis, out string
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Build the merged handoff genesis (from-chain base + successor fork section)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profilePath == "" || fromGenesis == "" {
				return fmt.Errorf("--profile and --from-genesis are required")
			}
			plan, err := buildPlanFromProfile(profilePath, fromGenesis)
			if err != nil {
				return err
			}
			if out != "" {
				if err := os.WriteFile(out, plan.Genesis, 0o644); err != nil {
					return err
				}
			}
			o := cmd.OutOrStdout()
			fmt.Fprintf(o, "handoff: %s -> %s at %s block; %d node(s)\n",
				plan.From.ID, plan.To.ID, plan.AtFork, len(plan.Nodes))
			w := tabwriter.NewWriter(o, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tCHAIN\tROLE\tNETID\tP2P\tHTTP\tETCD")
			for _, n := range plan.Nodes {
				role := "validator"
				if n.Producer {
					role = "producer"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\t%d\n",
					n.Index+1, n.Chain, role, n.NetworkID, n.Ports.P2P, n.Ports.HTTP, n.Ports.Etcd)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if out != "" {
				fmt.Fprintf(o, "merged genesis written to %s (%d bytes)\n", out, len(plan.Genesis))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "golden upgrade profile (profiles/*.yaml)")
	cmd.Flags().StringVar(&fromGenesis, "from-genesis", "", "from-chain base genesis (e.g. gwemix wemix genesis output)")
	cmd.Flags().StringVar(&out, "out", "", "write the merged genesis to this path")
	return cmd
}
