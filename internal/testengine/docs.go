// Package engine is the chain-agnostic orchestrator (H1, application layer):
// for each test it resolves config, computes the reuse fingerprint, builds or
// reuses an environment (place -> keyreg -> genesis -> provision -> supervise
// -> collect), runs the interpreter, records results, and tears down. Chain
// differences enter only through plugins, never through branching here.
package testengine
