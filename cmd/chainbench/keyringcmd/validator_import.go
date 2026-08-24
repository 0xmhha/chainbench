package keyringcmd

import (
	"github.com/spf13/cobra"
)

// newValidatorImportCmd imports an existing key as a validator identity for a
// chain — from a private key, mnemonic, or file — and attaches the chain's
// consensus material (BLS/PoP for wbft, derived in process).
func newValidatorImportCmd() *cobra.Command {
	var chain string
	var jsonOut bool
	var src SourceFlags
	var sf storeFlags
	var pf PasswordFlags
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a key as a validator identity for a chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source, err := src.Source(pf.Source())
			if err != nil {
				return err
			}
			return runValidator(cmd, chain, source, &sf, &pf, false, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the validator identity as JSON")
	src.Bind(cmd)
	sf.bind(cmd)
	pf.Bind(cmd)
	return cmd
}
