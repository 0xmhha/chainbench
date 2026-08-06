// Package engine is the chain-agnostic orchestrator (H1, application layer):
// for each test it resolves config, computes the reuse fingerprint, builds or
// reuses an environment (place -> keyreg -> genesis -> provision -> supervise
// -> collect), runs the interpreter, records results, and tears down. Chain
// differences enter only through plugins, never through branching here.
//
// Status: interface freeze only (T0.1). The composed implementation is the
// walking skeleton in T4.1.
package engine
