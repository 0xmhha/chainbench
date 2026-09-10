// Package testhelper is the DSL's built-in vocabulary: the actions, assertions
// and readers a test definition can name, and the code behind each one.
//
// It is where the grammar meets a live chain. [Register] installs the whole
// vocabulary into an interp.Registry, and the interpreter then looks up each
// statement by the name the definition used. Nothing here parses — the grammar
// is internal/dsl and the execution order is internal/dsl/interp; this package
// supplies the verbs.
//
// The vocabulary is grouped by what it touches, one file each:
//
//	builtins.go   tx, waitBlock, waitFor, read, newAccount, and the block and
//	              call assertions every chain shares
//	assets.go     faucet, deployContract, registerContract
//	fault.go      stopNode, startNode, restartNode, swapNode, partition,
//	              healPartition, readNodeLog — the per-node control a fault
//	              scenario drives
//	derived.go    the websocket pair (wsOpen, wsSubscribe, wsCollected)
//	account.go    resolving an account label to an address and a signer
//	read.go       the readers a $ref binding draws its value from
//	metric.go     the metric assertion, scraping a node's metrics endpoint
//	logs.go       log search
//	blockprobe.go, txprobe.go, load.go
//
// One rule runs through all of them: a verb that cannot do its job says so
// rather than quietly passing. The metric assertion fails outright on a node
// with no metrics port instead of skipping, because "metrics silently off" is
// the defect it exists to catch, and an unbound $ref is an error rather than an
// empty string, because reading "" would make an assertion pass for the wrong
// reason (see interp.Bindings). Deciding that a whole definition does not apply
// to a chain is a different job and belongs to testengine's capability gate.
//
// The registered names are the contract the authoring guide publishes; it is
// generated from these registration sites, so adding a verb here is what makes
// it appear there (docs/guide/dsl-authoring.md).
package testhelper
