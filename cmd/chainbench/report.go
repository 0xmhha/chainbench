package main

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/obs"
)

func newReportCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show stored run/test results (from a setup's data dir)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir is required")
			}
			store, err := obs.NewFileStore(filepath.Join(dataDir, "runs.json"))
			if err != nil {
				return err
			}
			runs := store.ListRuns()
			out := cmd.OutOrStdout()
			if len(runs) == 0 {
				fmt.Fprintln(out, "no runs recorded")
				return nil
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tPHASE\tCHAIN\tSTATUS")
			var ok, failed, skipped int
			for _, r := range runs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Phase, r.Chain, r.Status)
				switch r.Status {
				case obs.RunSucceeded:
					ok++
				case obs.RunFailed:
					failed++
				case obs.RunSkipped:
					skipped++
				}
			}
			if err := w.Flush(); err != nil {
				return err
			}
			fmt.Fprintf(out, "\ntotal=%d ok=%d failed=%d skipped=%d\n", len(runs), ok, failed, skipped)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with runs.json")
	return cmd
}
