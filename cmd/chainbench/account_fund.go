package main

import (
	"fmt"
	"math/big"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/keyringcmd"
)

// newAccountFundCmd funds an account: it sends amount wei to a recipient from a
// funding account resolved through the shared key model (a private key,
// mnemonic, or a local/remote key file), using the chain's account provider.
// This is the account-layer sibling of `faucet`, but the funding key can come
// from any source, not just an inline hex key.
func newAccountFundCmd() *cobra.Command {
	var (
		chain        string
		manifestPath string
		templatePath string
		rpcURL       string
		to           string
		amount       string
	)
	var src keyringcmd.SourceFlags
	var pf keyringcmd.PasswordFlags
	cmd := &cobra.Command{
		Use:   "fund",
		Short: "Send funds to an account from a funding key (private key, mnemonic, or file)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			amt, ok := new(big.Int).SetString(amount, 10)
			if !ok {
				return fmt.Errorf("bad --amount %q (decimal wei expected)", amount)
			}
			source, err := src.Source(pf.Source())
			if err != nil {
				return err
			}
			funder, err := source.Resolve(cmd.Context())
			if err != nil {
				return err
			}
			ap, err := resolveAccountProvider(chain, manifestPath, templatePath)
			if err != nil {
				return err
			}
			hash, err := ap.Faucet(cmd.Context(), funder.Bytes(), to, amt, rpcURL)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), hash)
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "embedded chain id; ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&to, "to", "", "recipient address (0x-hex)")
	cmd.Flags().StringVar(&amount, "amount", "", "amount in wei (decimal)")
	_ = cmd.MarkFlagRequired("rpc")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("amount")
	src.Bind(cmd)
	pf.Bind(cmd)
	return cmd
}
