// Package keymat is the shared key-material core behind the keys/account/
// validator commands (and, later, their MCP tools). It composes three
// orthogonal concerns so each can vary independently: a Source (where a key
// comes from — generate, private key, mnemonic, a local or remote file), a
// Store (how it is saved — a raw hex file or an encrypted keystore), and a
// PasswordSource (how the password is supplied — static, a file, or prompt-once
// then reuse). Every strategy is backed by the github.com/0xmhha/accounts SDK
// (account, hdwallet, keystore), so no external crypto dependency is added.
package keymat

import "fmt"

// DefaultCoinType is Ethereum's SLIP-44 coin type. BIP-39/BIP-44 uses it by
// convention for Ethereum-style chains, but a chain may register its own, so
// keymat exposes it as a configurable knob rather than hard-coding 60.
const DefaultCoinType uint32 = 60

// HDPath is a BIP-44 derivation path m/44'/coin'/account'/change/index. Only
// mnemonic (BIP-39) sources use it. CoinType is first-class because the correct
// derived key depends on it: a chain that is not Ethereum should set its own
// coin type so the addresses match that chain's wallets.
type HDPath struct {
	CoinType uint32
	Account  uint32
	Change   uint32
	Index    uint32
}

// DefaultHDPath returns m/44'/60'/0'/0/0 — the Ethereum default. Override
// CoinType (and the rest) for other chains.
func DefaultHDPath() HDPath { return HDPath{CoinType: DefaultCoinType} }

// String renders the path in the standard notation the hdwallet consumes.
func (p HDPath) String() string {
	return fmt.Sprintf("m/44'/%d'/%d'/%d/%d", p.CoinType, p.Account, p.Change, p.Index)
}
