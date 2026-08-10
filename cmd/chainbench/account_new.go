package main

import (
	"github.com/spf13/cobra"
)

// newAccountNewCmd creates a new general (EOA) account — a keypair meant to be
// funded and transacted with — and optionally stores it (--out). It shares the
// source/store/password model with `keys` and `validator`; the account layer is
// where on-chain lifecycle (state, fund) lives.
func newAccountNewCmd() *cobra.Command {
	var jsonOut bool
	var sf storeFlags
	var pf passwordFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new account (EOA); optionally store it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGenerate(cmd, &sf, &pf, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the account as JSON")
	sf.bind(cmd)
	pf.bind(cmd)
	return cmd
}
