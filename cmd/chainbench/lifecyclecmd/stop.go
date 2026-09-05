package lifecyclecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewStop() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a composed network's nodes (by the pids its workspace records)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.NetworkStop(cmd.Context(), app.Deps{}, app.NetworkStopIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped %d node(s)\n", res.Stopped)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory")
	return cmd
}
