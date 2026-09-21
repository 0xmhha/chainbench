package preset

import (
	"fmt"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// Key is one KEY preset: the identities a network runs as.
//
// It sits beside [Chain] rather than inside keyring because a preset is a
// DOCUMENT, and this package owns the documents. keyring owns the key model the
// document is made of — Entry, Network, Label — and uses this type rather than
// declaring it, which is what keeps one definition in one place.

// KeyPreset is a decoded ring: the identities it holds, and — for a file that
// still carries them — the network decisions recorded beside them.
//
// A ring that declares no validator set is the point: it is identities and
// nothing more, so what a network does with them is the network's to say.
type Key struct {
	// Nodes are the per-node identities. This is the keyring proper.
	Nodes []keyring.Entry
	// Network holds the decisions the file recorded, if any. It is empty for a
	// ring that declares only identities.
	Network keyring.Network
	// Password unlocks the keystores in this ring.
	Password string
}

// validate rejects a file that cannot describe a usable ring.

// Node returns the entry with the given 1-based index and whether it was found.
func (p Key) Node(index int) (keyring.Entry, bool) {
	for _, n := range p.Nodes {
		if n.Index == index {
			return n, true
		}
	}
	return keyring.Entry{}, false
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
func (p Key) NetworkFor(n int) keyring.Network {
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

// NetworkForNodes answers who validates when the validator set is a *specific*
// set of nodes rather than the first n — the producers a topology actually
// placed. For EN,BP,PN,BP the producers are node2 and node4, not the first two,
// so a count is not enough: the caller passes the node indices it resolved from
// the placement's roles, in order, and this selects each node's recorded
// identity by index.
//
// ExtraData is kept only when the selection is exactly the recorded validator
// set in order (the common all-producers case, unchanged); otherwise it is
// cleared so the genesis builder recomputes it for this set, the same rule
// NetworkFor follows when it narrows. The governance council is not narrowed — it
// is independent of which nodes validate — and is seeded from the validators
// only when the ring recorded none.
//
// An index with no identity in the ring is an error: it means the placement and
// the key set disagree, which must fail while composing rather than produce a
// genesis that names a validator the network cannot launch.
func (p Key) NetworkForNodes(indices []int) (keyring.Network, error) {
	out := p.Network
	vals := make([]string, 0, len(indices))
	bls := make([]string, 0, len(indices))
	for _, idx := range indices {
		e, ok := p.Node(idx)
		if !ok {
			return keyring.Network{}, fmt.Errorf("keyring: the key set has no identity for node%d", idx)
		}
		vals = append(vals, e.Address)
		if e.BLS != nil {
			bls = append(bls, e.BLS.PublicKey)
		}
	}
	out.Validators = vals
	out.BLSKeys = bls
	// The governance council is the producers: the system contracts require the
	// member set to match the validator set (govValidator rejects a genesis where
	// they differ in count), and every shipped preset records the two as the same
	// addresses. So the selected producers are also the members — otherwise a
	// three-producer network would seed a four-member council and fail to init.
	out.Members = append([]string(nil), vals...)
	// Keep the recorded extra-data only when the selection is exactly the
	// recorded validator set, in order. Then this is the common all-producers
	// case and nothing changes. A different set (a reordered or partial one, like
	// EN,BP,PN,BP) describes validators the recorded extra-data does not, so it is
	// cleared and the genesis builder recomputes it.
	if !slices.Equal(vals, p.Network.Validators) {
		out.ExtraData = ""
	}
	return out, nil
}

// truncate returns the first n of s as a new slice, tolerating a shorter s so a
// set with no derive.BLS keys narrows without a bounds check at every call site.
//
// It copies rather than reslices: the result outlives this call, and a shared
// array is how an append on the result silently rewrites the original.
func truncate(s []string, n int) []string {
	if n >= len(s) {
		return append([]string(nil), s...)
	}
	return append([]string(nil), s[:n]...)
}
