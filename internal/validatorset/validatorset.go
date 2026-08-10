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

	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Role names an account's function in a chain.
const (
	RoleValidator  = "validator"
	RoleGovernance = "governance-member"
	RoleNode       = "node"
)

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
	preset, err := keys.LoadPreset(keysDir)
	if err != nil {
		return Roster{}, err
	}

	family := p.Manifest().ConsensusFamily
	r := Roster{Chain: chainID, Family: family}

	switch family {
	case "wbft":
		for i, addr := range preset.Validators {
			detail := "no BLS"
			if i < len(preset.BLSKeys) && preset.BLSKeys[i] != "" {
				detail = "BLS present"
			}
			r.Accounts = append(r.Accounts, Account{Role: RoleValidator, Index: i + 1, Address: addr, Detail: detail})
		}
		for _, addr := range preset.Members {
			r.Accounts = append(r.Accounts, Account{Role: RoleGovernance, Address: addr, Detail: "system-contract council"})
		}
	case "poa":
		r.Note = "poa: validators are not in genesis — they are registered at the governance/etcd bootstrap; the key set only fixes node identities."
	default:
		r.Note = fmt.Sprintf("unknown consensus family %q; showing node identities only", family)
	}

	for _, n := range preset.Nodes {
		r.Accounts = append(r.Accounts, Account{Role: RoleNode, Index: n.Index, Address: n.Address, Detail: "devp2p identity"})
	}
	return r, nil
}
