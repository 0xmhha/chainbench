package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/state"
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
			d := driver.NewLocalDriver()
			out := cmd.OutOrStdout()
			stopped := 0
			for _, n := range ns.Nodes {
				if n.PID <= 0 {
					continue
				}
				if err := d.Stop(cmd.Context(), driver.Handle{Index: n.Index, PID: n.PID}); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "node%d (pid %d): %v\n", n.Index, n.PID, err)
					continue
				}
				stopped++
			}
			fmt.Fprintf(out, "stopped %d node(s)\n", stopped)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json")
	return cmd
}
