package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func newAccountCmd() *cobra.Command {
	acct := &cobra.Command{
		Use:   "account",
		Short: "Inspect account state over RPC (validator identities live under `validator`)",
	}
	acct.AddCommand(newAccountStateCmd())
	return acct
}

func newAccountStateCmd() *cobra.Command {
	var (
		rpcURL string
		addr   string
	)
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Report an account's balance, nonce, and whether it has code",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" || addr == "" {
				return fmt.Errorf("--rpc and --address are required")
			}
			c := rpc.Dial(rpcURL)
			ctx := cmd.Context()
			bal, err := c.BalanceAt(ctx, addr)
			if err != nil {
				return err
			}
			nonce, err := c.NonceAt(ctx, addr)
			if err != nil {
				return err
			}
			code, err := c.CodeAt(ctx, addr)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "address:  %s\nbalance:  %s wei\nnonce:    %d\ncontract: %v\n",
				addr, bal.String(), nonce, hasCode(code))
			return nil
		},
	}
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&addr, "address", "", "account address (0x-hex)")
	return cmd
}

// hasCode reports whether an eth_getCode result is a non-empty contract.
func hasCode(code string) bool {
	return code != "" && code != "0x" && code != "0x0"
}
