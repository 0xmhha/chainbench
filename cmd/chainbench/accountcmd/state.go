package accountcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newAccountStateCmd reports an account's on-chain state over RPC.
func newStateCmd() *cobra.Command {
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
			out, err := app.AccountState(cmd.Context(), deps(cmd), app.AccountStateIn{
				RPC: rpcURL, Address: addr,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "address:  %s\nbalance:  %s wei\nnonce:    %d\ncontract: %v\n",
				out.Address, out.Balance, out.Nonce, out.Contract)
			return nil
		},
	}
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&addr, "address", "", "account address (0x-hex)")
	return cmd
}

// deps is what every account verb hands the app layer: side notes to stderr.
func deps(cmd *cobra.Command) app.Deps {
	errOut := cmd.ErrOrStderr()
	return app.Deps{Env: os.Getenv, Logf: func(format string, args ...any) {
		fmt.Fprintf(errOut, format+"\n", args...)
	}}
}
