package lifecyclecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewConsensus() *cobra.Command {
	var (
		chain        string
		manifestPath string
		templatePath string
		rpcURL       string
	)
	cmd := &cobra.Command{
		Use:   "consensus",
		Short: "Query consensus state (validator set) using the chain's RPC namespace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" {
				return fmt.Errorf("--rpc url is required")
			}
			res, err := app.Validators(cmd.Context(), deps(cmd), chain, manifestPath, templatePath, rpcURL)
			if err != nil {
				return err
			}
			vals := res.Validators
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "validators (%s via %s): %d\n", res.Chain, res.Method, len(vals))
			for i, v := range vals {
				fmt.Fprintf(out, "  %d. %s\n", i+1, v)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "embedded chain id; ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	return surface.ReadOnly(cmd)
}
