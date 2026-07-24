package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/registry"
)

func newChainsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chains",
		Short: "List the registered chains",
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "CHAIN\tFAMILY\tBINARY\tCHAIN_ID\tNAMESPACE")
			for _, id := range registry.Names() {
				p, err := registry.Get(id)
				if err != nil {
					return err
				}
				m := p.Manifest()
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					m.ID, m.ConsensusFamily, m.Binary, m.ChainID, m.Consensus.RPCNamespace)
			}
			return w.Flush()
		},
	}
}
