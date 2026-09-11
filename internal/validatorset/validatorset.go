// Package validatorset presents a chain's consensus identities — its validator
// set and related roles — from a key set. The roles differ by consensus family:
// a wbft-family chain (stablenet, wbft) carries its validators (with baked BLS
// keys) and — for anzeon system contracts — a governance council in genesis; a
// poa-family chain (wemix) has no validators in genesis (they are registered at
// the governance/etcd bootstrap), so its key set only fixes node identities.
// This is the shared core behind the `validator` CLI subcommand and its MCP
// mirror. Plain (EOA) account concerns live under the `account` surface instead.
package validatorset

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// What an account does in a chain. Spelled Account rather than Role because a
// node's role and an account's function are different questions about
// different things, and sharing the word Role made them look like one (A7).
// The account role vocabulary is registry's: a family produces the names and a
// reader matches on them, so declaring them again here would be one concept with
// two homes ([[layers]] §5b, pinned by arch.TestNamesDoNotCollide).

// Account is one account with its chain role.
type Account struct {
	Role    string `json:"role"`
	Index   int    `json:"index,omitempty"`
	Address string `json:"address"`
	Detail  string `json:"detail,omitempty"`
}

// Roster is the chain-aware account view of a key set.
type Roster struct {
	Chain    string    `json:"chain"`
	Family   string    `json:"family"`
	Accounts []Account `json:"accounts"`
	Note     string    `json:"note,omitempty"`
}

// Load resolves the accounts a chain needs from the preset at keysDir, grouped
// by role per the chain's consensus family. It errors if the chain is not
// registered or the preset cannot be read.
func Load(chainID, keysDir string) (Roster, error) {
	p, err := registry.Get(chainID)
	if err != nil {
		return Roster{}, err
	}
	if keysDir == "" {
		keysDir = "keys/preset"
	}
	preset, err := store.LoadPreset(keysDir)
	if err != nil {
		return Roster{}, err
	}

	family := p.Manifest().ConsensusFamily
	r := Roster{Chain: chainID, Family: family}

	// Ask the family what it takes from a ring, rather than switching on its id.
	// This used to be a `switch family { case "wbft": … case "poa": … }` here,
	// which put each family's knowledge in two places: adding one meant editing
	// this switch as well as the family, and a family the switch did not know
	// was reported as "unknown" when the truth was "has not said".
	//
	// A ring that declares no validator set means the network chooses, so the
	// roster shows what a full-size network would use.
	net := preset.NetworkFor(0)
	if reader, ok := p.Family().(registry.RingAccountReader); ok {
		accounts, note := reader.RingAccounts(net.Validators, net.BLSKeys, net.Members)
		for _, a := range accounts {
			r.Accounts = append(r.Accounts, Account{
				Role: a.Role, Index: a.Index, Address: a.Address, Detail: a.Detail,
			})
		}
		r.Note = note
	} else {
		r.Note = fmt.Sprintf("consensus family %q does not say which accounts it takes from a key set; showing node identities only", family)
	}

	for _, n := range preset.Nodes {
		r.Accounts = append(r.Accounts, Account{Role: registry.AccountNode, Index: n.Index, Address: n.Address, Detail: "devp2p identity"})
	}
	return r, nil
}
