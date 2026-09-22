// Package origin is one vocabulary for where a value came from.
//
// Several places in a composition record this, and each had invented its own
// words: a resolved network said blueprint/inventory/keyset/chain/default, a
// compose plan said declaration/command/harness. The two overlapped without
// agreeing — "harness" and "default" are the same rung under two names — and
// neither could say that a chain-preset or the server set chose a value, which
// are two of the seven things that do.
//
// What this is NOT: a mechanism. Nothing here decides which value wins. The
// rungs are ordered so that the order can be READ, and so a reader meeting
// "placement" in one artifact and in another knows it means the same rung, but
// the merging itself happens upstream — in the DSL's merge of a case onto a
// chain-preset, in the composition's choice between a flag and a declaration,
// in the allocator's assignment of hosts and ports.
//
// nodeconfig.Layer is deliberately not folded in. It answers a different
// question — who ASKED for a knob the binary does not have, so a refusal can
// name them — and its own documentation says why it is not this list: the
// declaration tiers arrive at the argv assembler already merged, so the
// assembler cannot tell them apart and does not pretend to.
package origin

// Origin names one rung a value can come from.
//
// It is a string rather than an integer because it is written into artifacts a
// person reads (a plan, a resolved network) and a number there would need a
// table to decode. [Origin.Rank] is where the order lives.
type Origin string

// The rungs, weakest first. This is the priority line the design fixed: a value
// from a later rung is the one a run uses, and a value from an earlier one is
// what it would have used otherwise.
//
// Two of them never compete. A chain's own facts and a key set's material have
// no rival supplier — nothing else knows a nodekey — but "where did this come
// from" still has to have an answer for them, which is why they are rungs and
// not a separate idea.
const (
	// FromDefault is a built-in this code chose because nothing else named the
	// value. A reader has no file to open, which is exactly why a plan says so
	// out loud.
	FromDefault Origin = "default"
	// FromChain is what the chain-manifest knows: the chain id, the flag dialect,
	// the forks the build has.
	FromChain Origin = "chain"
	// FromKeySet is what a key preset supplied: a nodekey, a sealing account.
	FromKeySet Origin = "keyset"
	// FromBlueprint is a value a network blueprint document states. It sits at the
	// declaration rung and keeps its own name because a reader who has to go
	// change the value opens that file and not another.
	FromBlueprint Origin = "blueprint"
	// FromDeclaration is a value the chain-preset and the case state, after the
	// case's overrides have been merged onto the preset. The two are one rung
	// here because the merge happens before anything records provenance — which
	// of them said it is answered by reading the case.
	FromDeclaration Origin = "declaration"
	// FromPlacement is what the allocator decided from the server set: which host a
	// node runs on, which port slot it holds, where its data root is.
	FromPlacement Origin = "placement"
	// FromCommand is what the invocation overrode, from the CLI or from MCP. It
	// wins over every document.
	FromCommand Origin = "command"
)

// ladder is the priority line, weakest first. Rank reads it; a test holds it to
// the design document, so the line exists once as data rather than in prose in
// several places.
var ladder = []Origin{FromDefault, FromChain, FromKeySet, FromBlueprint, FromDeclaration, FromPlacement, FromCommand}

// Rank is where o sits on the line: higher wins. An unknown origin ranks below
// everything, so a value carrying a word this package does not know never
// silently outranks one that does.
func (o Origin) Rank() int {
	for i, r := range ladder {
		if r == o {
			return i + 1
		}
	}
	return 0
}

// Known reports whether o is one of the rungs.
func (o Origin) Known() bool { return o.Rank() > 0 }

// Ladder is the line, weakest first. It is a copy, so a caller reading the
// order cannot reorder it for everyone else.
func Ladder() []Origin { return append([]Origin(nil), ladder...) }
