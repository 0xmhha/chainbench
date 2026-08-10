package main

import (
	"github.com/spf13/cobra"
)

// newAccountCmd is the account group — general (EOA) blockchain accounts you
// transact with. Validator identities live under `validator`; raw keypairs
// under `keys`. Subcommands live in the account_*.go files and are composed here.
func newAccountCmd() *cobra.Command {
	acct := &cobra.Command{
		Use:   "account",
		Short: "Inspect and manage accounts (EOA); validator identities live under `validator`",
	}
	acct.AddCommand(newAccountNewCmd(), newAccountImportCmd(), newAccountFundCmd(), newAccountListCmd(), newAccountStateCmd())
	return acct
}
