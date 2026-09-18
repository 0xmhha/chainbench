// Package blueprint is the declaration a network is composed from: one
// document that says everything about it, and nothing about how to build it.
//
// The problem it answers is that the composition's inputs are in four pieces —
// a topology file, a server set, a key preset and a family's own config — and
// none of them describes the whole network (network-blueprint-design.md §1.1).
// A reader who wants to know what will be launched has to open all four and
// then know how the steps combine them.
//
// This package parses and checks the document. It resolves nothing: a field
// left out stays left out, and filling it from the inventory, the key set, the
// plugin or the family is Resolve's job, one layer up (§3.4). Keeping the two
// apart is what lets a partial declaration round-trip — which is the property
// `net blueprint --from-preset` needs, since it writes a document a person then
// edits by hand.
//
// Every field is optional. {chain: wemix, nodes: [{role: bp} x4]} is a whole
// blueprint. What is NOT optional is that a field written down is honoured:
// nothing further along may overwrite an explicit value (§3.1, principle 2).
package blueprint
