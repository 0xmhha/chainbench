package main

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/pipeline/verify"
)

func newVerifyCmd() *cobra.Command {
	var (
		chain        string
		dataDir      string
		rpcURLs      []string
		delay        time.Duration
		readyTimeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a network is producing blocks (from --rpc or a setup's --data-dir)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ns, err := resolveNodeSet(dataDir, chain, rpcURLs)
			if err != nil {
				return err
			}
			bus, closeBus := obsBus()
			defer closeBus()
			rep, err := verify.Run(cmd.Context(), ns, verify.Options{
				ProgressDelay: delay,
				ReadyTimeout:  readyTimeout,
			}, bus)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "producing: %v\n", rep.Producing)
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tRPC\tCHAIN_ID\tBLOCK\tPEERS\tSYNCING\tOK")
			for _, n := range rep.Nodes {
				fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%d\t%v\t%v\n",
					n.Index, n.RPCURL, n.ChainID, n.BlockNumber, n.PeerCount, n.Syncing, n.OK)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (optional metadata, used with --rpc)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "load the network from a setup's nodeset.json")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "node RPC URL (repeatable)")
	cmd.Flags().DurationVar(&delay, "progress-delay", 2*time.Second, "wait between block-height samples")
	cmd.Flags().DurationVar(&readyTimeout, "ready-timeout", 45*time.Second, "how long to wait for the network to start producing blocks (0 = single check, no wait)")
	return cmd
}
