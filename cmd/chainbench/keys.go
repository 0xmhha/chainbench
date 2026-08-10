package main

import (
	"github.com/spf13/cobra"
)

// newKeysCmd is the keys group — raw asymmetric key material (keypairs), the
// primitive layer under `account` (on-chain identity) and `validator`
// (consensus identity). Subcommands live in the keys_*.go files and are composed
// here. The preset/validator-set bundle generator moved to `validator set`.
func newKeysCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "keys",
		Short: "Generate and inspect raw key material (keypairs)",
	}
	c.AddCommand(newKeysNewCmd())
	return c
}
