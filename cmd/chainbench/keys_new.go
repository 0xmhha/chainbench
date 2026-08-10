package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// newKeysNewCmd generates a fresh secp256k1 keypair and its address — the raw
// key material that an account or validator is then built from. It is the
// primitive layer: no keystore, no on-chain state, no consensus material.
func newKeysNewCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a secp256k1 keypair and its address",
		RunE: func(cmd *cobra.Command, _ []string) error {
			priv, address, err := accounts.GenerateKey()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			privHex := "0x" + hex.EncodeToString(priv)
			if jsonOut {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]string{"privateKey": privHex, "address": address})
			}
			fmt.Fprintf(out, "privateKey: %s\naddress:    %s\n", privHex, address)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the keypair as JSON")
	return cmd
}
