package main

import (
	"github.com/spf13/cobra"
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
			return runGenerate(cmd, &sf, &pf, viewKeys, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the keypair as JSON")
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
