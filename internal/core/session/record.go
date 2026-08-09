package session

// StepResult records one executed step for steps.json: its inputs and the
// observed outcome (tx hash, receipt) so a run can be replayed and audited.
type StepResult struct {
	Index   int
	Type    string
	On      string
	Signer  string
	Nonce   uint64
	Gas     string
	Hash    string
	Receipt map[string]any
	// Error is why the step failed, empty when it succeeded. Without it a
	// failed run records that a step failed but not why, which is the one thing
	// the reader needs (design: never hide the cause).
	Error string
}

// AssertResult records one assertion outcome with its provenance for
// assert.json. Provenance carries the source-specific detail (rpc method/raw,
// func name/args, or log file/lines/offset) so a result is traceable.
type AssertResult struct {
	ID         string
	Source     string
	On         string
	Assert     string
	Expected   any
	Actual     any
	Pass       bool
	Provenance map[string]any
}

// PostResult records a post-action outcome. Post-action failures are recorded
// but do not change the test verdict.
type PostResult struct {
	Name   string
	OK     bool
	Detail string
}
