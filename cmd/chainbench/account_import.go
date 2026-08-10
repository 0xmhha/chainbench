package main

import (
	"github.com/spf13/cobra"
)

// newAccountImportCmd imports an existing account (EOA) from a private key, a
// BIP-39 mnemonic (with a configurable HD coin type), or a key file, and
// optionally re-stores it. Same source/store/password model as `keys import`.
func newAccountImportCmd() *cobra.Command {
	var jsonOut bool
	var src sourceFlags
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an account from a private key, mnemonic, or file; optionally store it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runImport(cmd, &src, &sf, &pf, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the result as JSON")
	src.bind(cmd)
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
