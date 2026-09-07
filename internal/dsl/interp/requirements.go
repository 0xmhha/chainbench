package interp

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// NodeTable is what the interpreter and its vocabulary need of an environment:
// the nodes, how to address one, and how to write back a node whose pid
// changed.
//
// It is four methods of the eleven session.Environment carries. The other seven
// are storage — an id, a fingerprint, a data path, a log path, a chainstate
// directory — and the interpreter has no business with any of them. Declaring
// the requirement here rather than taking the whole interface is the same move
// keyring/operation makes with its Opener: a package says what it needs, and
// whoever has it supplies it.
//
// session.Environment satisfies this without knowing it exists, so nothing
// changes for the engine. What changes is for everyone else: a test can drive
// the interpreter with four small methods instead of standing up a real session
// on disk, which is what interp's and testhelper's tests were doing because
// implementing eleven was the more expensive option.
type NodeTable interface {
	// Nodes is the table, in node order.
	Nodes() []node.Node
	// Resolve maps a selector ("bp1", "en:any", "node3") to a node.
	Resolve(selector string) (node.Node, error)
	// ResolveEach maps several at once.
	ResolveEach(selectors []string) ([]node.Node, error)
	// UpdateNode replaces the entry for n.Index. A node stopped or relaunched
	// mid-run keeps one record of its pid: this table, not a private copy.
	UpdateNode(n node.Node)
}

// Recorder is what the interpreter writes its verdict into: four methods of the
// twelve a session.TestRecord carries.
//
// The result types stay session's on purpose. Mirroring StepResult and its
// siblings here would leave two structs that have to agree, and a field added
// to one and forgotten in the other drops evidence from the artifact silently —
// which is the failure this codebase has spent the year removing (three port
// representations, two ring writers, a migration record that went stale). A
// narrower interface costs nothing and buys the same legibility; a duplicated
// vocabulary costs a recurring hazard and buys an import.
type Recorder interface {
	// Step records one step's provenance, by index.
	Step(i int, r session.StepResult)
	// Assert records one assertion's outcome.
	Assert(r session.AssertResult)
	// PostAction records one post-action's outcome.
	PostAction(r session.PostResult)
	// Status records the test's terminal verdict.
	Status(s session.TestStatus)
}
