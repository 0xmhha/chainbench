package resourcecmd

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/internal/serverflag"

	"github.com/0xmhha/chainbench/cmd/chainbench/internal/mapview"

	"github.com/0xmhha/chainbench/internal/app"
)

// newPoolCmd reports the resource a network may be allocated from. It is what
// answers "why was that refused" without the operator reading the server set
// and doing the arithmetic.
func newPoolCmd() *cobra.Command {
	var workspaceDir string
	var asJSON bool
	var sf serverflag.Flags
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Show the addresses and port slots a network may be composed from",
		Long: "Report the pool: which addresses are available, how many port slots each\n" +
			"holds, how many nodes that is in total, and how many a workspace already\n" +
			"uses. Credentials are never part of the answer — the pool says where nodes\n" +
			"may run, not how to log in.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := app.NetPool(cmd.Context(), deps(cmd), app.NetPoolIn{
				DataDir: workspaceDir, Server: sf.Ref(),
			})
			if err != nil {
				return err
			}
			if asJSON {
				return mapview.JSON(cmd.OutOrStdout(), out)
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "HOST")
			for _, h := range out.Hosts {
				fmt.Fprintf(w, "%s\n", h)
			}
			_ = w.Flush()
			fmt.Fprintf(cmd.OutOrStdout(), "%d host(s) x %d slot(s) = %d node(s); %d used, %d free\nports: %s\n",
				len(out.Hosts), out.Slots, out.Cap, out.Used, out.Free, out.Source)
			// Who holds what: "0 free" alone tells an operator nothing about
			// which workspace to remove.
			names := make([]string, 0, len(out.ByNetwork))
			for n := range out.ByNetwork {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s holds %d\n", n, out.ByNetwork[n])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&workspaceDir, "workspace-dir", "", "a workspace to count in addition to those under ~/.chainbench (optional)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the pool as JSON")
	sf.Bind(cmd)
	return cmd
}
