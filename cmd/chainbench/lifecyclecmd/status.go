package lifecyclecmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewStatus() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a composed network's node set (from its workspace)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.NetworkStatus(cmd.Context(), app.Deps{}, app.NetworkStatusIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			ns := res.Nodes
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "chain: %s   network: %s   nodes: %d\n", ns.Chain, ns.Network, len(ns.Nodes))
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tROLE\tRPC\tPID\tPROCESS")
			for _, n := range ns.Nodes {
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\n", n.Index, n.Role, n.RPCURL, n.PID, processState(res.Alive, n.Index, n.PID))
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory")
	return surface.ReadOnly(cmd)
}

// processState says what the machine answered about a node's recorded pid.
//
// "unknown" is a real answer and not a nicer word for dead: a machine that
// could not be asked has not told us the node is gone, and reporting that as
// stopped would send an operator to restart something that is running.
func processState(alive map[int]bool, index, pid int) string {
	if pid <= 0 {
		return "not started"
	}
	live, asked := alive[index]
	switch {
	case !asked:
		return "unknown"
	case live:
		return "running"
	default:
		return "gone"
	}
}
