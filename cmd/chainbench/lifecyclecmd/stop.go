// Package lifecyclecmd owns a composed network AFTER it is up: stopping it,
// reporting what is running, removing what a run left behind (clean), and
// judging whether it is still one healthy chain (verify, consensus, baseline).
//
// verify and baseline sit here rather than with the read verbs in chaincmd
// because they do not describe a composition, they pass judgement on a running
// one — the distinction that made a forked network report healthy until the
// agreement check was added.
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
