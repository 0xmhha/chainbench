package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/state"
)

func newStopCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the nodes a setup launched (from nodeset.json PIDs)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir with a setup's nodeset.json is required")
			}
			ns, err := state.LoadNodeSet(dataDir)
			if err != nil {
				return err
			}
			stopped, errs := setup.StopNodeSet(cmd.Context(), driver.NewLocalDriver(), ns)
			for _, e := range errs {
				fmt.Fprintln(cmd.ErrOrStderr(), e)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped %d node(s)\n", stopped)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json")
	return cmd
}
