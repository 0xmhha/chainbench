// Package arch turns the architecture rules into tests.
//
// The rules were written down first and enforced by attention, which does not
// scale past the point where a refactor touches fifty packages. Worse, the
// checking has twice been done by an ad-hoc script carrying its own copy of the
// placement table; both times the copy disagreed with the document and the
// answer was wrong — once inventing a violation, once silently skipping a
// package it did not know.
//
// So the tests here read the documents rather than restating them:
//
//   - layers_test.go parses the module placement out of
//     docs/dev/architecture/layers.md §3 and checks the real import graph
//     against it. A package the table does not mention fails, because a check
//     that skips what it does not know is not a check.
//   - state_test.go parses the state-ownership verdicts out of the same
//     document and checks which packages actually write files.
//// Not every rule has a document to read. Some are properties of the code that no
// prose states, and those carry their own list with a reason per entry — a
// reviewer reads the reason, not a diff:
//
//   - target_test.go: a composition step may ask where its target is and may not
//     run a different job because of the answer.
//   - producer_fills_test.go: when one function is the sole producer of a struct
//     another function decides on, every field the decider reads is filled or
//     explained. Three shipped defects were that shape — a comparison, a peering
//     check and a fork check that were all written, all tested, and all inert
//     because nothing supplied the value.
////   - worklist_test.go holds the tracker's canonical open-work list (§0): its size
//     is pinned, and every entry has to point at a section that exists AND that
//     actually contains the row it names. The tracker had been counted with two
//     marker vocabularies and a status pass reported "nothing open" three times
//     while eleven items sat in the vocabulary it did not count.
//
//   - requirements_test.go reads the product requirements' traceability table out
//     of docs/dev/chainbench-system-direction.md §12 and holds two things a
//     machine can hold: every requirement names a test, and every named test
//     exists. It does not judge whether the test proves the requirement — nothing
//     can, and a check that pretended to would be the vacuous kind this package
//     exists to remove.
//

// There is no production code here. The package exists so the rules have a
// home that `go test ./...` runs.
package arch
