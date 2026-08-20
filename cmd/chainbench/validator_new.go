package main

import (
	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// newValidatorNewCmd generates a new validator identity for a chain: a fresh
// account plus the chain's consensus material (BLS/PoP for wbft, derived in process;
// none for poa, whose validators register at bootstrap). Optionally stores the
// key.
func newValidatorNewCmd() *cobra.Command {
	var chain string
	var jsonOut bool
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a new validator identity for a chain (chain-aware consensus material)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidator(cmd, chain, keyring.RandomSource{}, &sf, &pf, true, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the validator identity as JSON")
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
