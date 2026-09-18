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
//
// There is no production code here. The package exists so the rules have a
// home that `go test ./...` runs.
package arch
