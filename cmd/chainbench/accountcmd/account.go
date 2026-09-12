// Package accountcmd owns the commands that act on an ACCOUNT rather than on a
// node or a chain: reading its state, and funding it. `faucet` is here and not
// under `tx` because its subject is the recipient, not the transaction.
package accountcmd

import (
	"github.com/spf13/cobra"
)

// newAccountCmd is the account group — the ON-CHAIN side of an account: its
// balance and nonce (state), and funding it from the faucet. Key material —
// creating, importing, listing, exporting keys — lives under `keyring`, one
// group for every kind of key; the generation/import verbs this group used to
// carry were the same operations under another name and are gone with the
// deprecated `keys` group.
func New() *cobra.Command {
	acct := &cobra.Command{
		Use:   "account",
		Short: "Inspect and fund accounts on a chain; key material lives under `keyring`",
	}
	acct.AddCommand(newFundCmd(), newStateCmd())
	return acct
}
