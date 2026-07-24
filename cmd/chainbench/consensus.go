package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/core/consensus"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

func newConsensusCmd() *cobra.Command {
	var (
		chain  string
		rpcURL string
	)
	cmd := &cobra.Command{
		Use:   "consensus",
		Short: "Query consensus state (validator set) using the chain's RPC namespace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" {
				return fmt.Errorf("--rpc url is required")
			}
			p, err := registry.Get(chain)
			if err != nil {
				return err
			}
			method := p.Manifest().Consensus.ValidatorsMethod
			vals, err := consensus.Validators(cmd.Context(), rpc.Dial(rpcURL), method)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "validators (%s via %s): %d\n", chain, method, len(vals))
			for i, v := range vals {
				fmt.Fprintf(out, "  %d. %s\n", i+1, v)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "chain id (selects the RPC method)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	return cmd
}
