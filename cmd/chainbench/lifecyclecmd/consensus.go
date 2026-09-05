package lifecyclecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
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
			p, err := app.ResolveChain(chain, manifestPath, templatePath)
			if err != nil {
				return err
			}
			method := p.Manifest().Consensus.ValidatorsMethod
			vals, err := registry.Validators(cmd.Context(), rpc.Dial(rpcURL), method)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "validators (%s via %s): %d\n", p.Manifest().ID, method, len(vals))
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
	return cmd
}
