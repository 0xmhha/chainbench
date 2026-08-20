package main

import (
	"github.com/spf13/cobra"
)

// newAccountCmd is the account group — general (EOA) blockchain accounts you
// transact with. Subcommands live in the account_*.go files.
//
// Its key-handling half (new, import, list) is superseded by `keyring`: an
// account key and a node key are the same kind of secret, and splitting them by
// intended use meant three ways to make one. The on-chain half (fund, state)
// stays, because that is about a chain and not about a key.
func newAccountCmd() *cobra.Command {
	acct := &cobra.Command{
		Use:   "account",
		Short: "Inspect and manage accounts (EOA); key material lives under `keyring`",
	}
	acct.AddCommand(newAccountNewCmd(), newAccountImportCmd(), newAccountFundCmd(), newAccountListCmd(), newAccountStateCmd())
	return acct
}
