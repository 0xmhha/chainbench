package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/hardfork"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/state"
)

func newHardforkCmd() *cobra.Command {
	var (
		dataDir  string
		toChain  string
		toBinary string
		block    int64
		dryRun   bool
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
			// A same-chain hardfork (e.g. stablenet pre-fork -> stablenet
			// post-fork) swaps one binary build for another that activates the
			// fork at --block. Both resolve to the chain's manifest binary name,
			// so require an explicit post-fork path or there is nothing to swap.
			if toChain == ns.Chain && toBinary == "" {
				return fmt.Errorf("same-chain hardfork (%s -> %s) requires --to-binary <post-fork build>", ns.Chain, toChain)
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
				bin, err := resolveBinary(toBinary, to.Manifest().Binary)
				if err != nil {
					return err
				}
				specs, err := state.LoadNodeSpecs(dataDir)
				if err != nil {
					return fmt.Errorf("hardfork: load node specs (run setup with --launch first): %w", err)
				}
				newNS, err := plan.Execute(cmd.Context(), driver.NewLocalDriver(), specs, bin)
				if err != nil {
					return err
				}
				if err := state.SaveNodeSet(dataDir, newNS); err != nil {
					return err
				}
				// Keep nodespecs.json consistent so later node/hardfork ops use the
				// post-fork binary (identity/config args are unchanged).
				for i := range specs {
					specs[i].Binary = bin
				}
				if err := state.SaveNodeSpecs(dataDir, specs); err != nil {
					return err
				}
				fmt.Fprintf(out, "upgraded %d node(s) to %s (%s); state updated\n",
					len(newNS.Nodes), plan.ToChain, plan.ToBinary)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json (the running from-chain)")
	cmd.Flags().StringVar(&toChain, "to-chain", "", "target chain id to upgrade to")
	cmd.Flags().StringVar(&toBinary, "to-binary", "", "target node binary path (default: chain binary on PATH)")
	cmd.Flags().Int64Var(&block, "block", 0, "hardfork activation block")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not execute")
	return cmd
}
