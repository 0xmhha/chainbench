package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/hardfork"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

func newHardforkCmd() *cobra.Command {
	var (
		dataDir string
		toChain string
		block   int64
		dryRun  bool
	)
	cmd := &cobra.Command{
		Use:   "hardfork",
		Short: "Plan a chain upgrade (swap binary at a fork block, keeping node data)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir with a setup's nodeset.json is required")
			}
			if toChain == "" {
				return fmt.Errorf("--to-chain is required")
			}
			ns, err := state.LoadNodeSet(dataDir)
			if err != nil {
				return err
			}
			from, err := registry.Get(ns.Chain)
			if err != nil {
				return fmt.Errorf("from-chain %q: %w", ns.Chain, err)
			}
			to, err := registry.Get(toChain)
			if err != nil {
				return err
			}
			plan, err := hardfork.BuildPlan(ns, from, to, block, dataDir)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "hardfork: %s -> %s  (binary %s -> %s)  at block %d\n",
				plan.FromChain, plan.ToChain, plan.FromBinary, plan.ToBinary, plan.Block)
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tDATADIR\tSWAP")
			for _, s := range plan.Swaps {
				fmt.Fprintf(w, "%d\t%s\t%s -> %s\n", s.Index, s.DataDir, s.FromBinary, s.ToBinary)
			}
			if err := w.Flush(); err != nil {
				return err
			}

			if !dryRun {
				return fmt.Errorf("hardfork execution (stop nodes + relaunch on %s at block %d) needs PID tracking and a built %s binary; not yet wired",
					plan.ToBinary, plan.Block, plan.ToBinary)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json (the running from-chain)")
	cmd.Flags().StringVar(&toChain, "to-chain", "", "target chain id to upgrade to")
	cmd.Flags().Int64Var(&block, "block", 0, "hardfork activation block")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not execute")
	return cmd
}
