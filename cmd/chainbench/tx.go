package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func newTxCmd() *cobra.Command {
	tx := &cobra.Command{
		Use:   "tx",
		Short: "Send transactions and wait for receipts",
	}
	tx.AddCommand(newTxSendCmd(), newTxWaitCmd())
	return tx
}

func newTxSendCmd() *cobra.Command {
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
			key, err := decodeHex(fromKey)
			if err != nil {
				return fmt.Errorf("bad --from-key: %w", err)
			}
			payload, err := decodeHex(data)
			if err != nil {
				return fmt.Errorf("bad --data: %w", err)
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
			hash, err := w.Execute(cmd.Context(), to, payload, wei)
			if err != nil {
				return err
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

func newTxWaitCmd() *cobra.Command {
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
			rec, err := waitReceipt(cmd, rpcURL, hash, timeout)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			var r struct {
				Status      string `json:"status"`
				BlockNumber string `json:"blockNumber"`
				GasUsed     string `json:"gasUsed"`
				Contract    string `json:"contractAddress"`
			}
			_ = json.Unmarshal(rec, &r)
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
	return cmd
}

// waitReceipt polls for a transaction receipt until it appears or timeout.
func waitReceipt(cmd *cobra.Command, rpcURL, hash string, timeout time.Duration) (json.RawMessage, error) {
	c := rpc.Dial(rpcURL)
	deadline := time.Duration(0)
	for {
		rec, err := c.TxReceipt(cmd.Context(), hash)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			return rec, nil
		}
		if deadline >= timeout {
			return nil, fmt.Errorf("timed out after %s waiting for tx %s", timeout, hash)
		}
		time.Sleep(time.Second)
		deadline += time.Second
	}
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

// decodeHex decodes a 0x-prefixed (or bare) hex string; "" yields nil bytes.
func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// parseWei parses a decimal wei amount; "" is treated as zero.
func parseWei(s string) (*big.Int, error) {
	if s == "" {
		return big.NewInt(0), nil
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("bad --value %q (decimal wei expected)", s)
	}
	return v, nil
}
