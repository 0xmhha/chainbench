package txcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewContract() *cobra.Command {
	c := &cobra.Command{
		Use:   "contract",
		Short: "Deploy and call contracts",
	}
	c.AddCommand(newDeployCmd(), newCallCmd())
	return c
}

func newDeployCmd() *cobra.Command {
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
			out, err := app.ContractDeploy(cmd.Context(), deps(cmd), app.ContractDeployIn{
				Chain:   app.ChainRef{Chain: chain, RPC: rpcURL},
				FromKey: fromKey, Bytecode: bytecode, Value: value,
			})
			if err != nil {
				return flagError(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "tx:       %s\ncontract: %s\n", out.Tx, out.Address)
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

func newCallCmd() *cobra.Command {
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
			res, err := app.ContractCall(cmd.Context(), deps(cmd), app.ContractCallIn{
				RPC: rpcURL, To: to, Data: data,
			})
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
	return surface.ReadOnly(cmd)
}
