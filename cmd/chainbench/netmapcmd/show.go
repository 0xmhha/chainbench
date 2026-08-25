package netmapcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newShowCmd answers where a composed network's nodes are, in both directions.
// It is a query, not a step: it changes nothing, which is why it lives here
// and not in the composed sequence, even though it reads the same workspace.
func newShowCmd() *cobra.Command {
	var dataDir, label, host, addr string
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
			if dataDir == "" {
				return fmt.Errorf("--data-dir is required")
			}
			out, err := app.NetMap(cmd.Context(), deps(cmd), app.NetMapIn{
				DataDir: dataDir, Node: nodeIdx, Label: label, Host: host, Port: port, Addr: addr,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return emitJSON(cmd.OutOrStdout(), out)
			}
			printMap(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "local workspace directory")
	cmd.Flags().IntVar(&nodeIdx, "node", 0, "select by identity (the 1-based node number)")
	cmd.Flags().StringVar(&label, "label", "", "select by identity (node7) or role alias (en2)")
	cmd.Flags().StringVar(&host, "host", "", "select every node on an address")
	cmd.Flags().IntVar(&port, "port", 0, "select whichever node listens on a port (p2p, etcd, http, ws, auth or metrics)")
	cmd.Flags().StringVar(&addr, "addr", "", "select by an address as a log line prints it (host:port)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the map as JSON")
	return cmd
}
