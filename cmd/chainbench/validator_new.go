package main

import (
	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/keymat"
)

// newValidatorNewCmd generates a new validator identity for a chain: a fresh
// account plus the chain's consensus material (BLS/PoP for wbft via --bootnode;
// none for poa, whose validators register at bootstrap). Optionally stores the
// key.
func newValidatorNewCmd() *cobra.Command {
	var chain, bootnode string
	var jsonOut bool
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a new validator identity for a chain (chain-aware consensus material)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidator(cmd, chain, bootnode, keymat.RandomSource{}, &sf, &pf, true, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().StringVar(&bootnode, "bootnode", "", "go-wbft bootnode tool (required for wbft BLS derivation)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the validator identity as JSON")
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
