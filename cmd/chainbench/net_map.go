package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newNetMapCmd answers where nodes are, in both directions. It is a query, not
// a step: it changes nothing, which is why it is not part of the composed
// sequence even though it reads the same workspace.
func newNetMapCmd() *cobra.Command {
	var dataDir, label, host string
	var nodeIdx, port int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "map",
		Short: "Look up where nodes are: by node, label, host, or port",
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
			out, err := app.NetMap(cmd.Context(), app.Deps{}, app.NetMapIn{
				DataDir: dataDir, Node: nodeIdx, Label: label, Host: host, Port: port,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			}
			printMap(cmd, out)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "local workspace directory")
	cmd.Flags().IntVar(&nodeIdx, "node", 0, "select by identity (the 1-based node number)")
	cmd.Flags().StringVar(&label, "label", "", "select by identity (node7) or role alias (en2)")
	cmd.Flags().StringVar(&host, "host", "", "select every node on an address")
	cmd.Flags().IntVar(&port, "port", 0, "select whichever node listens on a port (p2p, etcd, http, ws, auth or metrics)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the map as JSON")
	return cmd
}

func printMap(cmd *cobra.Command, out app.NetMapOut) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\tROLE\tLABEL\tHOST\tP2P\tETCD\tHTTP\tDATADIR")
	for _, e := range out.Entries {
		etcd := "-"
		if e.Etcd != 0 {
			etcd = fmt.Sprint(e.Etcd)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%d\t%s\n",
			e.Label, e.Role, e.Alias, e.Host, e.P2P, etcd, e.HTTP, e.DataDir)
	}
	_ = w.Flush()

	roles := make([]string, 0, len(out.Roles))
	for r := range out.Roles {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	summary := ""
	for i, r := range roles {
		if i > 0 {
			summary += ", "
		}
		summary += fmt.Sprintf("%d %s", out.Roles[r], r)
	}
	shown := ""
	if len(out.Entries) != out.Total {
		shown = fmt.Sprintf("%d of ", len(out.Entries))
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s%d node(s): %s\n", shown, out.Total, summary)
}

// newNetPoolCmd reports the resource a network may be allocated from. It is
// what answers "why was that refused" without the operator reading the
// inventory and doing the arithmetic.
func newNetPoolCmd() *cobra.Command {
	var dataDir string
	var asJSON bool
	var sf serverFlags
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Show the addresses and port slots a network may be composed from",
		Long: "Report the pool: which addresses are available, how many port slots each\n" +
			"holds, how many nodes that is in total, and how many a workspace already\n" +
			"uses. Credentials are never part of the answer — the pool says where nodes\n" +
			"may run, not how to log in.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := app.NetPool(cmd.Context(), app.Deps{}, app.NetPoolIn{
				DataDir: dataDir, Server: sf.ref(),
			})
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "HOST")
			for _, h := range out.Hosts {
				fmt.Fprintf(w, "%s\n", h)
			}
			_ = w.Flush()
			free := out.Cap - out.Used
			fmt.Fprintf(cmd.OutOrStdout(), "%d host(s) x %d slot(s) = %d node(s); %d used, %d free\nports: %s\n",
				len(out.Hosts), out.Slots, out.Cap, out.Used, free, out.Source)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "workspace to count used slots from (optional)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the pool as JSON")
	sf.bind(cmd)
	return cmd
}
