package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

func newStopCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the nodes a setup launched (from nodeset.json PIDs)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.NetworkStop(cmd.Context(), app.Deps{}, app.NetworkStopIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			printStopFailures(cmd, res.Failed)
			fmt.Fprintf(cmd.OutOrStdout(), "stopped %d node(s)\n", res.Stopped)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json")
	return cmd
}

// printStopFailures reports the per-node stop errors on stderr. They are
// diagnostics, not a failed command: the rest of the network still stopped.
func printStopFailures(cmd *cobra.Command, failed []error) {
	for _, e := range failed {
		fmt.Fprintln(cmd.ErrOrStderr(), e)
	}
}
