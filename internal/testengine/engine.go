package testengine

import "context"

// Engine runs a suite of tests, composing the middle components behind one
// uniform flow (parse -> resolve -> fingerprint -> reuse-or-build -> run ->
// record -> teardown). It holds no chain-specific logic.
type Engine interface {
	// Run executes the given specs (raw JSON definitions) serially and returns
	// the session root directory holding the artifacts.
	Run(ctx context.Context, specs [][]byte) (sessionRoot string, err error)
}
