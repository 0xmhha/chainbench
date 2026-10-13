package chaincmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// newNetObserveCmd reports what every recorded node's machine shows now: the
// recorded launch running, a pid that runs something else, a node with no
// process, or one the record lost. It changes nothing.
func newNetObserveCmd() *cobra.Command {
	var dataDir string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "observe",
		Short: "Observe each recorded node's process and binary on its machine, changing nothing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			res, err := app.ChainObserve(cmd.Context(), surface.Deps(cmd), app.ChainObserveIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tROLE\tSTATE\tPID\tBINARY\tREASON")
			for _, n := range res.Nodes {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", n.Label, n.Role, n.State, n.PID, n.Executable, n.Reason)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (where the composition is set up)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the observation as JSON")
	surface.BindLogins(cmd)
	return surface.ReadOnly(cmd)
}
