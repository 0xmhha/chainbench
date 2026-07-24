package main

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/verify"
)

func newVerifyCmd() *cobra.Command {
	var (
		chain   string
		rpcURLs []string
		delay   time.Duration
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify an existing network is producing blocks (requirement #7, #9)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(rpcURLs) == 0 {
				return fmt.Errorf("at least one --rpc url is required")
			}
			eps := make([]attach.Endpoint, len(rpcURLs))
			for i, u := range rpcURLs {
				eps[i] = attach.Endpoint{RPCURL: u}
			}
			ns, err := attach.Build(chain, "attached", eps)
			if err != nil {
				return err
			}
			rep, err := verify.Run(cmd.Context(), ns, verify.Options{ProgressDelay: delay}, nil)
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
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (optional metadata)")
	cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "node RPC URL (repeatable)")
	cmd.Flags().DurationVar(&delay, "progress-delay", 2*time.Second, "wait between block-height samples")
	return cmd
}
