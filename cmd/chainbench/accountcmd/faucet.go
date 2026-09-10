package accountcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

func NewFaucet() *cobra.Command {
	var (
		chain        string
		manifestPath string
		templatePath string
		rpcURL       string
		fromKey      string
		to           string
		amount       string
	)
	cmd := &cobra.Command{
		Use:   "faucet",
		Short: "Send funds from a genesis-allocated key to an account (requirement #3)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			hash, err := app.Faucet(cmd.Context(), surface.Deps(cmd), app.FaucetIn{
				Chain: app.ChainRef{
					Chain: chain, Manifest: manifestPath, Template: templatePath, RPC: rpcURL,
				},
				FromKey: fromKey, To: to, Amount: amount,
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
	cmd.Flags().StringVar(&fromKey, "from-key", "", "funding private key (hex)")
	cmd.Flags().StringVar(&to, "to", "", "recipient address (0x-hex)")
	cmd.Flags().StringVar(&amount, "amount", "", "amount in wei (decimal)")
	_ = cmd.MarkFlagRequired("rpc")
	_ = cmd.MarkFlagRequired("from-key")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("amount")
	return cmd
}
