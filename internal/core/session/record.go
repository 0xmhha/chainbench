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

// ArtifactRef names one input a test actually used, so a reviewer can trace a
// verdict back to the exact genesis, config, command, or deployment it ran
// against. Ref is either an env-relative path (the test shared the environment's
// artifact unchanged) or a "sha256:<hex>" content address (the test used its own
// override, snapshotted by content). It never carries key material — a key is
// referenced by its public identity or file path, never its secret bytes.
type ArtifactRef struct {
	// Kind is what the artifact is: "genesis", "config", "commands",
	// "deployment", or "keys".
	Kind string `json:"kind"`
	// Ref is an env-relative path or a "sha256:<hex>" content address.
	Ref string `json:"ref"`
	// Node is the node the artifact belongs to (e.g. "node2"); empty for a
	// network-wide artifact like the genesis.
	Node string `json:"node,omitempty"`
}

// TestArtifacts is the manifest of inputs one test used, written to
// artifacts.json in the test record. It is the seam later stages fill: E2
// records reused/transferred material, E3 the config and command, E8 the
// deployment. E0A fixes the schema so those stages do not each invent one.
type TestArtifacts struct {
	Refs []ArtifactRef `json:"refs"`
}
