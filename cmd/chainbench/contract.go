package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func newContractCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "contract",
		Short: "Deploy and call contracts",
	}
	c.AddCommand(newContractDeployCmd(), newContractCallCmd())
	return c
}

func newContractDeployCmd() *cobra.Command {
	var (
		chain    string
		rpcURL   string
		fromKey  string
		bytecode string
		value    string
	)
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a contract from creation bytecode and print its address",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" || fromKey == "" || bytecode == "" {
				return fmt.Errorf("--rpc, --from-key and --bytecode are required")
			}
			key, err := decodeHex(fromKey)
			if err != nil {
				return fmt.Errorf("bad --from-key: %w", err)
			}
			code, err := decodeHex(bytecode)
			if err != nil {
				return fmt.Errorf("bad --bytecode: %w", err)
			}
			wei, err := parseWei(value)
			if err != nil {
				return err
			}
			ap, err := accounts.ForChain(chain)
			if err != nil {
				return err
			}
			w, err := ap.OpenWallet(cmd.Context(), key, rpcURL)
			if err != nil {
				return err
			}
			hash, addr, err := w.Deploy(cmd.Context(), code, wei)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "tx:       %s\ncontract: %s\n", hash, addr)
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "chain id")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&fromKey, "from-key", "", "deployer private key (hex)")
	cmd.Flags().StringVar(&bytecode, "bytecode", "", "contract creation bytecode (0x-hex)")
	cmd.Flags().StringVar(&value, "value", "0", "value in wei to endow (decimal)")
	return cmd
}

func newContractCallCmd() *cobra.Command {
	var (
		rpcURL string
		to     string
		data   string
	)
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Read-only contract call (eth_call), printing the 0x-hex result",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" || to == "" {
				return fmt.Errorf("--rpc and --to are required")
			}
			res, err := rpc.Dial(rpcURL).EthCall(cmd.Context(), to, data)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&to, "to", "", "contract address (0x-hex)")
	cmd.Flags().StringVar(&data, "data", "", "calldata (0x-hex)")
	return cmd
}
