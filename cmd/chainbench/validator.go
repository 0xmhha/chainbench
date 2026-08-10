package main

import (
	"github.com/spf13/cobra"
)

// newValidatorCmd is the validator-identity group. A validator is an account
// plus consensus-specific material (BLS keys for wbft; governance/stake
// registration for poa), so it gets its own command rather than living under
// `account`. Subcommands live in the validator_*.go files and are composed here.
func newValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Inspect and manage validator identities (chain-aware consensus roles)",
	}
	cmd.AddCommand(newValidatorRosterCmd(), newValidatorSetCmd())
	return cmd
}
