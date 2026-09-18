// Package accounts is chainbench's boundary over the accounts SDK
// (github.com/0xmhha/accounts). AccountProvider is chainbench-owned so the rest
// of chainbench does not couple to the SDK's concrete types; the default
// implementation is backed by the SDK's per-chain protocol profiles and its
// account/wallet packages (decision D3 — interface boundary + SDK default
// impl). This provider is the replacement for network/internal/signer +
// abiutil; those are retired when the tx handlers are absorbed (G2–G3).
package accounts

import (
	"context"
	"math/big"

	"github.com/0xmhha/accounts/protocol"
)

// AccountProvider is the chainbench-facing account/tx capability for one chain.
type AccountProvider interface {
	// Protocol returns the underlying accounts SDK protocol profile.
	Protocol() protocol.Protocol
	// SupportsTxType reports whether the chain accepts the given EIP-2718 tx
	// type byte (e.g. 0x16 fee delegation).
	SupportsTxType(t byte) bool
	// HasAccountExtra reports whether the chain's account model carries an
	// extended account-state record beyond the standard nonce/balance (the SDK's
	// Extra). It is a generic capability predicate; the record's concrete fields
	// are chain-specific (e.g. stablenet's blacklisted/authorized bits) and are
	// not part of this contract.
	HasAccountExtra() bool

	// AddressForKey derives the 0x-prefixed hex address for a private key,
	// offline (no RPC).
	AddressForKey(privKey []byte) (string, error)
	// OpenWallet builds a Wallet for privKey against a node's RPC endpoint.
	OpenWallet(ctx context.Context, privKey []byte, rpcURL string) (Wallet, error)
	// Faucet funds toHex with amountWei from the genesis-allocated key
	// fromKey via the node at rpcURL, returning the tx hash (requirement #3).
	Faucet(ctx context.Context, fromKey []byte, toHex string, amountWei *big.Int, rpcURL string) (string, error)
}

// defaultProvider adapts an accounts SDK protocol.Protocol to AccountProvider.
type defaultProvider struct {
	proto protocol.Protocol
}

// New returns the default AccountProvider backed by the given SDK protocol.
func New(p protocol.Protocol) AccountProvider {
	return defaultProvider{proto: p}
}

// ForChain returns the default AccountProvider for a chain id known to the
// accounts SDK ("stablenet"|"wbft"|"wemix").
func ForChain(chainID string) (AccountProvider, error) {
	p, err := protocol.ByName(chainID)
	if err != nil {
		return nil, err
	}
	return New(p), nil
}

func (d defaultProvider) Protocol() protocol.Protocol { return d.proto }
func (d defaultProvider) SupportsTxType(t byte) bool  { return d.proto.SupportsTxType(t) }
func (d defaultProvider) HasAccountExtra() bool       { return d.proto.HasExtra() }

func (d defaultProvider) AddressForKey(privKey []byte) (string, error) {
	return addressForKey(privKey)
}

func (d defaultProvider) OpenWallet(ctx context.Context, privKey []byte, rpcURL string) (Wallet, error) {
	return openWallet(ctx, privKey, rpcURL)
}

func (d defaultProvider) Faucet(ctx context.Context, fromKey []byte, toHex string, amountWei *big.Int, rpcURL string) (string, error) {
	w, err := openWallet(ctx, fromKey, rpcURL)
	if err != nil {
		return "", err
	}
	return w.SendCoin(ctx, toHex, amountWei)
}
