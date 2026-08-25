package keyring

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Entry is one node's material: the secret it is built from, and the public
// identity that follows from it.
//
// It replaces the three shapes this used to have — a read-side NodeKey without
// BLS, a write-side Node with it, and a registry Key with yet another field set
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
	Nodekey PrivateKey
	// Identity is everything public that derives from Nodekey.
	Identity
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
	// BLSKeys are the validators' BLS public keys (0x-hex), aligned with
	// Validators. Empty for a family that does not use BLS.
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

// Preset is a decoded ring: the identities it holds, and — for a file that
// still carries them — the network decisions recorded beside them.
//
// A ring that declares no validator set is the point: it is identities and
// nothing more, so what a network does with them is the network's to say.
type Preset struct {
	// Nodes are the per-node identities. This is the keyring proper.
	Nodes []Entry
	// Network holds the decisions the file recorded, if any. It is empty for a
	// ring that declares only identities.
	Network Network
	// Password unlocks the keystores in this ring.
	Password string
}

// validate rejects a file that cannot describe a usable ring.

// Node returns the entry with the given 1-based index and whether it was found.
func (p Preset) Node(index int) (Entry, bool) {
	for _, n := range p.Nodes {
		if n.Index == index {
			return n, true
		}
	}
	return Entry{}, false
}

// NetworkFor answers who validates in a network of n validators.
//
// When the ring's file recorded a validator set, the first n of it are taken.
// When it recorded none — a ring that is identities and nothing else — the
// first n identities are used. Either way the caller asks the same question and
// does not have to know which kind of ring it was handed.
//
// n<=0 or n beyond what is available means "all of them".
//
// ExtraData survives only when the whole recorded set is used. It encodes the
// validator set, so a narrowed one would describe validators the network never
// starts; the genesis builder recomputes it from the set it is given.
//
// The governance council is not narrowed: it is independent of how many
// validators are active.
func (p Preset) NetworkFor(n int) Network {
	if len(p.Network.Validators) > 0 {
		out := p.Network
		if n > 0 && n < len(out.Validators) {
			// Copied, not resliced. A truncated slice shares its array, so a
			// caller appending to the result would write over the preset's own
			// validator list.
			out.Validators = truncate(out.Validators, n)
			out.BLSKeys = truncate(out.BLSKeys, n)
			out.ExtraData = ""
		}
		return out
	}

	// A ring with no declared set: the network's validators are its first n
	// identities, in ring order.
	out := p.Network
	limit := len(p.Nodes)
	if n > 0 && n < limit {
		limit = n
	}
	for _, e := range p.Nodes[:limit] {
		out.Validators = append(out.Validators, e.Address)
		if e.BLS != nil {
			out.BLSKeys = append(out.BLSKeys, e.BLS.PublicKey)
		}
	}
	if len(out.Members) == 0 {
		// A governance council with no members cannot pass anything, so a
		// network that needs one and was told nothing seats its validators —
		// which is what every existing preset records anyway.
		out.Members = append([]string(nil), out.Validators...)
	}
	return out
}

// truncate returns the first n of s as a new slice, tolerating a shorter s so a
// set with no BLS keys narrows without a bounds check at every call site.
//
// It copies rather than reslices: the result outlives this call, and a shared
// array is how an append on the result silently rewrites the original.
func truncate(s []string, n int) []string {
	if n >= len(s) {
		return append([]string(nil), s...)
	}
	return append([]string(nil), s[:n]...)
}

// Verify reports whether an entry's recorded public fields match what its
// nodekey actually derives. A mismatch means the file and the key have come
// apart: a node would launch with one identity while the genesis registers
// another, which shows up as a chain that produces no blocks.
func (e Entry) Verify() error {
	want, err := Derive(e.Nodekey, derivationFor(e.Identity))
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
		return fmt.Errorf("keyring: node %d: key derives different BLS material", e.Index)
	}
	return nil
}

// derivationFor asks for exactly as much as the identity claims to have, so
// verifying a poa entry does not compute BLS material it never had.
func derivationFor(id Identity) Derivation {
	if id.BLS != nil {
		return WithBLS
	}
	return AccountOnly
}
