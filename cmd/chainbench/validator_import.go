package main

import (
	"github.com/spf13/cobra"
)

// newValidatorImportCmd imports an existing key as a validator identity for a
// chain — from a private key, mnemonic, or file — and attaches the chain's
// consensus material (BLS/PoP for wbft via --bootnode).
func newValidatorImportCmd() *cobra.Command {
	var chain, bootnode string
	var jsonOut bool
	var src sourceFlags
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a key as a validator identity for a chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source, err := src.source(pf.source())
			if err != nil {
				return err
			}
			return runValidator(cmd, chain, bootnode, source, &sf, &pf, false, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().StringVar(&bootnode, "bootnode", "", "go-wbft bootnode tool (required for wbft BLS derivation)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the validator identity as JSON")
	src.bind(cmd)
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
