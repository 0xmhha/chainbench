package netmapcmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newPoolCmd reports the resource a network may be allocated from. It is what
// answers "why was that refused" without the operator reading the server set
// and doing the arithmetic.
func newPoolCmd() *cobra.Command {
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
			out, err := app.NetPool(cmd.Context(), deps(cmd), app.NetPoolIn{
				DataDir: dataDir, Server: sf.ref(),
			})
			if err != nil {
				return err
			}
			if asJSON {
				return emitJSON(cmd.OutOrStdout(), out)
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
