package session

import (
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// Fingerprint is the full hex(sha256) of an environment's resolved declared
// config (binaries+genesis+config+topology+hardforks+placement). The env-id
// folder name is its first 12 hex chars; the full value is stored in env.json.
type Fingerprint string

// TestStatus is a test's terminal result, recorded in status.json.
type TestStatus string

const (
	// StatusPass means every assertion passed.
	StatusPass TestStatus = "pass"
	// StatusFail means a step expectation or an assertion failed.
	StatusFail TestStatus = "fail"
	// StatusBlocked means a pre-action failed, so the test did not run.
	StatusBlocked TestStatus = "blocked"
	// StatusSkip means the test did not apply to the target chain.
	StatusSkip TestStatus = "skip"
)

// Session is one test-run command: it owns the on-disk artifact tree and hands
// out shared environments and per-test records.
type Session interface {
	ID() string
	Root() string
	Keys() *store.Ring
	// Environment returns an existing environment with the given fingerprint,
	// or ok=false if none exists yet (drives reuse).
	Environment(fp Fingerprint) (Environment, bool)
	NewEnvironment(fp Fingerprint) (Environment, error)
	Test(seq int, id string) TestRecord
	Save() error
}

// Environment is a running chain instance shared across tests with the same
// fingerprint. It is the source of truth for node and endpoint resolution.
type Environment interface {
	ID() string
	Dir() string
	Fingerprint() Fingerprint
	// PopulateNodeTable fills the node table from a bring-up result before Save.
	PopulateNodeTable(ns node.NodeSet)
	Nodes() []node.Node
	// Resolve maps a selector ("bp1", "bp:any", "en:0") to a node.
	Resolve(selector string) (node.Node, error)
	ResolveEach(selectors []string) ([]node.Node, error)
	DataPath() string
	LogPath(nodeName string) string
	ChainstateDir() string
	Save() error
}

// TestRecord accumulates one test's artifacts (spec/steps/assert/status/post).
type TestRecord interface {
	Dir() string
	SetEnvRef(envID string)
	Spec(raw []byte)
	Step(i int, r StepResult)
	Assert(r AssertResult)
	Status(s TestStatus)
	// Reason records why a test ended the way it did, for the verdicts where
	// the status alone does not say: a blocked test failed to get an
	// environment, a skipped one did not apply. Without it an artifact records
	// that something went wrong and not what.
	Reason(why string)
	PostAction(r PostResult)
}
