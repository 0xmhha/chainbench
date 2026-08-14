package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func newStatusCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the launched network's node set (from nodeset.json)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir with a setup's nodeset.json is required")
			}
			ns, err := session.LoadLocalNodeSet(dataDir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "chain: %s   network: %s   nodes: %d\n", ns.Chain, ns.Network, len(ns.Nodes))
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tROLE\tRPC\tPID")
			for _, n := range ns.Nodes {
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", n.Index, n.Role, n.RPCURL, n.PID)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root with nodeset.json")
	return cmd
}
