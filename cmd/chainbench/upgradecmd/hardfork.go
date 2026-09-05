package upgradecmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewHardfork() *cobra.Command {
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
			planned, err := app.HardforkPlan(cmd.Context(), app.Deps{}, app.HardforkPlanIn{
				DataDir: dataDir, ToChain: toChain, ToBinary: toBinary, Block: block,
			})
			if err != nil {
				return err
			}
			plan := planned.Plan

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

			if dryRun {
				return nil
			}
			bin, err := resolveBinary(toBinary, planned.To.Manifest().Binary)
			if err != nil {
				return err
			}
			res, err := app.HardforkExecute(cmd.Context(), app.Deps{}, app.HardforkExecuteIn{
				Plan: planned, DataDir: dataDir, Binary: bin,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "upgraded %d node(s) to %s (%s); state updated\n",
				len(res.Nodes.Nodes), plan.ToChain, plan.ToBinary)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace of the running from-chain")
	cmd.Flags().StringVar(&toChain, "to-chain", "", "target chain id to upgrade to")
	cmd.Flags().StringVar(&toBinary, "to-binary", "", "target node binary path (default: chain binary on PATH)")
	cmd.Flags().Int64Var(&block, "block", 0, "hardfork activation block")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "plan only; do not execute")
	return cmd
}
