package catalogcmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewChains() *cobra.Command {
	return surface.ReadOnly(&cobra.Command{
		Use:   "chains",
		Short: "List the registered chains",
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "CHAIN\tFAMILY\tBINARY\tCHAIN_ID\tNAMESPACE")
			for _, id := range app.Chains(surface.Deps(cmd)) {
				p, err := app.Chain(surface.Deps(cmd), id)
				if err != nil {
					return err
				}
				m := p.Manifest()
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					m.ID, m.ConsensusFamily, m.Binary, m.ChainID, m.Consensus.RPCNamespace)
			}
			return w.Flush()
		},
	})
}
