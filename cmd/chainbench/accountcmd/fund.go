package accountcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/keyringcmd"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// newAccountFundCmd funds an account: it sends amount wei to a recipient from a
// funding account resolved through the shared key model (a private key,
// mnemonic, or a local/remote key file), using the chain's account provider.
// This is the account-layer sibling of `faucet`, but the funding key can come
// from any source, not just an inline hex key.
func newFundCmd() *cobra.Command {
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
			funder, err := src.Resolve(cmd.Context(), surface.Deps(cmd), pf.Source())
			if err != nil {
				return err
			}
			hash, err := app.FaucetFromKey(cmd.Context(), surface.Deps(cmd), app.FaucetKeyIn{
				Chain: app.ChainRef{
					Chain: chain, Manifest: manifestPath, Template: templatePath, RPC: rpcURL,
				},
				Key: funder, To: to, Amount: amount,
			})
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
