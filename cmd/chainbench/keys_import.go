package main

import (
	"github.com/spf13/cobra"
)

// newKeysImportCmd imports an existing key — from a known private key, a BIP-39
// mnemonic (with a configurable HD coin type), or a key file (raw hex or
// keystore) — and optionally re-stores it in the chosen format (--out).
func newKeysImportCmd() *cobra.Command {
	var jsonOut bool
	var src sourceFlags
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a key from a private key, mnemonic, or file (optionally store it)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runImport(cmd, &src, &sf, &pf, viewKeys, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the result as JSON")
	src.bind(cmd)
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
