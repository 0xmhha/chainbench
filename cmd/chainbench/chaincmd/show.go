package chaincmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/resourcecmd"

	"github.com/0xmhha/chainbench/internal/app"
)

// newShowCmd answers where a composed network's nodes are, in both directions.
// queryDeps is the app.Deps a read-only query runs with: side notes to stderr.
func queryDeps(cmd *cobra.Command) app.Deps {
	errOut := cmd.ErrOrStderr()
	return app.Deps{Logf: func(format string, args ...any) {
		fmt.Fprintf(errOut, format+"\n", args...)
	}}
}

// It is a query, not a step: it changes nothing. It lives in the net group
// because it reads the composed workspace — the placement that IS, where
// `resource plan` computes the placement that WOULD BE.
func newNetShowCmd() *cobra.Command {
	var workspaceDir, label, host, addr string
	var nodeIdx, port int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Look up where a composed network's nodes are: by node, label, host, or port",
		Long: "Read the composed network's placement. Given no selector it prints the whole\n" +
			"map; given one it answers that question, including the reverse ones — which\n" +
			"node owns this port, what runs on this address.\n\n" +
			"Each node carries two names. The identity (node7) is what reaches disk: the\n" +
			"datadir, the log file, the keyring entry. The alias (en2) is what a test\n" +
			"definition addresses, because a spec written once runs on many topologies.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if workspaceDir == "" {
				return fmt.Errorf("--workspace-dir is required")
			}
			out, err := app.NetMap(cmd.Context(), queryDeps(cmd), app.NetMapIn{
				DataDir: workspaceDir, Node: nodeIdx, Label: label, Host: host, Port: port, Addr: addr,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return resourcecmd.MapJSON(cmd.OutOrStdout(), out)
			}
			resourcecmd.PrintMap(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&workspaceDir, "workspace-dir", "", "workspace directory (where the composition was set up)")
	cmd.Flags().IntVar(&nodeIdx, "node", 0, "select by identity (the 1-based node number)")
	cmd.Flags().StringVar(&label, "label", "", "select by identity (node7) or role alias (en2)")
	cmd.Flags().StringVar(&host, "host", "", "select every node on an address")
	cmd.Flags().IntVar(&port, "port", 0, "select whichever node listens on a port (p2p, etcd, http, ws, auth or metrics)")
	cmd.Flags().StringVar(&addr, "addr", "", "select by an address as a log line prints it (host:port)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the map as JSON")
	return cmd
}
