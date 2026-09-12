// Package txcmd owns submitting work to a chain and waiting for its outcome:
// transactions (send, wait) and contracts (deploy, call). Its subject is the
// transaction; a command whose subject is the account it moves value to belongs
// in accountcmd.
package txcmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewTx() *cobra.Command {
	tx := &cobra.Command{
		Use:   "tx",
		Short: "Send transactions and wait for receipts",
	}
	tx.AddCommand(newSendCmd(), newWaitCmd())
	return tx
}

func newSendCmd() *cobra.Command {
	var (
		chain   string
		rpcURL  string
		fromKey string
		to      string
		data    string
		value   string
	)
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Sign and send a transaction to an address (optionally with calldata)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" || fromKey == "" || to == "" {
				return fmt.Errorf("--rpc, --from-key and --to are required")
			}
			hash, err := app.TxSend(cmd.Context(), surface.Deps(cmd), app.TxSendIn{
				Chain:   app.ChainRef{Chain: chain, RPC: rpcURL},
				FromKey: fromKey, To: to, Data: data, Value: value,
			})
			if err != nil {
				return flagError(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), hash)
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "stablenet", "chain id")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&fromKey, "from-key", "", "sender private key (hex)")
	cmd.Flags().StringVar(&to, "to", "", "recipient/contract address (0x-hex)")
	cmd.Flags().StringVar(&data, "data", "", "calldata (0x-hex); empty for a plain transfer")
	cmd.Flags().StringVar(&value, "value", "0", "value in wei (decimal)")
	return cmd
}

func newWaitCmd() *cobra.Command {
	var (
		rpcURL  string
		hash    string
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for a transaction receipt and print it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpcURL == "" || hash == "" {
				return fmt.Errorf("--rpc and --hash are required")
			}
			r, err := app.TxWait(cmd.Context(), surface.Deps(cmd), app.TxWaitIn{
				RPC: rpcURL, Hash: hash, Timeout: timeout,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "status:  %s\nblock:   %s\ngasUsed: %s\n", txStatus(r.Status), r.BlockNumber, r.GasUsed)
			if r.Contract != "" {
				fmt.Fprintf(out, "contract: %s\n", r.Contract)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "node RPC URL")
	cmd.Flags().StringVar(&hash, "hash", "", "transaction hash (0x-hex)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "how long to wait for inclusion")
	return surface.ReadOnly(cmd)
}

func txStatus(s string) string {
	switch s {
	case "0x1":
		return "success (0x1)"
	case "0x0":
		return "failed (0x0)"
	default:
		return s
	}
}

// flagError names the flag an operator typed for a value app rejected.
//
// app takes a transaction as a whole and reports which part was wrong; the
// operator typed flags. Translating here keeps the use case free of the CLI's
// spelling while still telling the person which word to fix.
func flagError(err error) error {
	if err == nil {
		return nil
	}
	for _, sub := range []struct{ from, to string }{
		{"bad sender key", "bad --from-key"},
		{"bad deployer key", "bad --from-key"},
		{"bad calldata", "bad --data"},
		{"bad bytecode", "bad --bytecode"},
		{"bad amount", "bad --value"},
	} {
		if strings.Contains(err.Error(), sub.from) {
			return fmt.Errorf("%s", strings.Replace(err.Error(), sub.from, sub.to, 1))
		}
	}
	return err
}
