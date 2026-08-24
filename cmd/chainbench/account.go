package main

import (
	"github.com/spf13/cobra"
)

// newAccountCmd is the account group — the ON-CHAIN side of an account: its
// balance and nonce (state), and funding it from the faucet. Key material —
// creating, importing, listing, exporting keys — lives under `keyring`, one
// group for every kind of key; the generation/import verbs this group used to
// carry were the same operations under another name and are gone with the
// deprecated `keys` group.
func newAccountCmd() *cobra.Command {
	acct := &cobra.Command{
		Use:   "account",
		Short: "Inspect and fund accounts on a chain; key material lives under `keyring`",
	}
	acct.AddCommand(newAccountFundCmd(), newAccountStateCmd())
	return acct
}
