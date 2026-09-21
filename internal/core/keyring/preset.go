package keyring

import (
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"strings"
)

// Entry is one node's material: the secret it is built from, and the public
// identity that follows from it.
//
// It replaces the three shapes this used to have — a read-side NodeKey without
// derive.BLS, a write-side Node with it, and a registry Key with yet another field set
// — none of which converted to another.
type Entry struct {
	// Label names this entry within a ring. A preset entry is labelled by its
	// index ("node1"); a ring built by a command labels its own.
	Label Label
	// Index is the 1-based node number within the ring, or 0 when the entry did
	// not come from a numbered set.
	Index int
	// Nodekey is the secret. It redacts itself when formatted; reaching the
	// hex takes an explicit Hex call.
	Nodekey derive.PrivateKey
	// derive.Identity is everything public that derives from Nodekey.
	derive.Identity
	// Account is the address this entry's keystore holds, when that is not the
	// one its nodekey derives. Empty means they are the same, or that no
	// keystore was read.
	//
	// The two are different things. A node's identity is its devp2p key; the
	// account it seals and stakes with is a keystore. They coincide in every
	// network this harness composes from a consistent ring, and they do not
	// have to — a wemix producer is a governance member, and the member is an
	// account rather than a peer. The key set is where that fact lives, so both
	// the genesis that funds the account and the launch that unlocks it read it
	// from the same place instead of each assuming.
	Account string
}

// SealingAccount is the address this entry seals and stakes with: its keystore's
// when that is a different account, and the one its nodekey derives otherwise.
//
// One rule, one owner. The genesis funds this address and the launch unlocks it,
// and those two answering differently is a producer that starts and cannot seal.
func (e Entry) SealingAccount() string {
	if e.Account != "" {
		return e.Account
	}
	return e.Address
}

// Network is what a *network* decides about a ring's identities: which of them
// validate, which seed the governance council, and who starts with a balance.
//
// None of it is a property of a key. Two networks can run from one ring with
// different validator sets, and a ring generated for one network is usable by
// another only because these answers are not baked into it.
//
// A preset file may record them, because presets predate the blueprint that
// owns them. [Preset.NetworkFor] is how a caller asks the question without
// caring whether the file had an answer.
type Network struct {
	// Validators are the validator addresses (0x-hex), in genesis order.
	Validators []string
	// BLSKeys are the validators' derive.BLS public keys (0x-hex), aligned with
	// Validators. Empty for a family that does not use derive.BLS.
	BLSKeys []string
	// ExtraData is the RLP-encoded validator extra-data (0x-hex), when the file
	// recorded one for exactly this set. It is derived from the validator set,
	// so it is only ever carried for the set it was computed from; the genesis
	// builder recomputes it whenever it is empty.
	ExtraData string
	// Members are the governance council addresses (0x-hex) that seed the
	// wbft-family system contracts. Empty for families with no system contracts.
	Members []string
	// Alloc is the raw genesis pre-funded accounts object (address -> account),
	// or nil when the network funds no accounts.
	Alloc json.RawMessage
}

// Verify reports whether an entry's recorded public fields match what its
// nodekey actually derives. A mismatch means the file and the key have come
// apart: a node would launch with one identity while the genesis registers
// another, which shows up as a chain that produces no blocks.
func (e Entry) Verify() error {
	// A public-only entry has no key to derive from, and deriving from the zero
	// key would produce a mismatch that reads like corrupted material. Say which
	// it is.
	if e.Nodekey == (derive.PrivateKey{}) {
		return fmt.Errorf("keyring: node %d: cannot verify a public-only entry — verification re-derives from the private key, which this read did not fetch", e.Index)
	}
	want, err := derive.Derive(e.Nodekey, derivationFor(e.Identity))
	if err != nil {
		return err
	}
	if !strings.EqualFold(want.Address, e.Address) {
		return fmt.Errorf("keyring: node %d: key derives address %s but the file records %s",
			e.Index, want.Address, e.Address)
	}
	if want.PublicKey != e.PublicKey {
		return fmt.Errorf("keyring: node %d: key derives a different devp2p public key", e.Index)
	}
	if e.BLS != nil && (want.BLS == nil || want.BLS.PublicKey != e.BLS.PublicKey || want.BLS.PoP != e.BLS.PoP) {
		return fmt.Errorf("keyring: node %d: key derives different derive.BLS material", e.Index)
	}
	return nil
}

// derivationFor asks for exactly as much as the identity claims to have, so
// verifying a poa entry does not compute derive.BLS material it never had.
func derivationFor(id derive.Identity) derive.Derivation {
	if id.BLS != nil {
		return derive.WithBLS
	}
	return derive.AccountOnly
}
