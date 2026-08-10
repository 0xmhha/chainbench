package main

import (
	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/keymat"
)

// newKeysNewCmd generates a fresh secp256k1 keypair — the raw key material an
// account or validator is built from — and optionally stores it (--out).
func newKeysNewCmd() *cobra.Command {
	var jsonOut bool
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a secp256k1 keypair (optionally store it)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := keymat.RandomSource{}.Resolve(cmd.Context())
			if err != nil {
				return err
			}
			path, err := saveKey(&sf, &pf, a)
			if err != nil {
				return err
			}
			return printKey(cmd.OutOrStdout(), a, true, path, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the keypair as JSON")
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
