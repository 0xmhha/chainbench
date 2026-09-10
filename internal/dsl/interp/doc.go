// Package interp executes a parsed test definition against a running network.
//
// internal/dsl says what a definition MEANS; this package says what happens
// when it runs. It walks the unified statement sequence in order, looks each
// statement's name up in a [Registry] — the vocabulary internal/testhelper
// installs — and records every step and assertion into the session.
//
// The registry is the seam. Nothing is seeded here, so the grammar does not
// depend on the verbs: a caller registers the built-in vocabulary, or its own.
// That is what lets the interpreter be tested without a chain, and what keeps
// a new built-in from being a change to the interpreter.
//
// [Bindings] carry values between statements. A step saves under a name and a
// later one reads it back with $ref. An unbound reference is an error, never a
// silently empty string — a typo has to be loud, because an assertion that
// compares against "" can pass for the wrong reason.
//
// The interpreter asks a session for what it needs rather than taking the whole
// thing: the node table, the node control it drives faults through, and the
// recorder. A narrower ask is what makes a fault scenario testable against a
// fake.
package interp
