package testengine

import (
	"errors"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// What the ATTACH path's failures are, in the terms the test area's states use.
//
// A composing run no longer needs this: its machine's position IS where it
// failed, and classifying the error afterwards said the same thing a second
// way. Attaching still walks a sequence of calls rather than states, so until
// it moves this is how its failures are told apart.
//
// The messages stay what they are — they name the file, the field and what to
// do about it, which is what makes them worth reading. The kind rides alongside
// (lifecycle.Mark) so a caller can branch on which state a failure is without
// matching on prose.
//
// The five reading kinds are the measurement's, not an invention:
// design-v3/state-machine-05-test-failures.md counted 47 sites in this stage and
// found their messages fall into five, each pointing somewhere different to go
// fix it. A sixth would mean the measurement missed one, and the test that walks
// the states says so.

var (
	// errUnreadable: a spec file cannot be read, or the run was given nowhere
	// to work. Go fix the command line.
	errUnreadable = errors.New("a spec could not be read")
	// errMalformed: the document is not the grammar — a topology entry that is
	// a string where a mapping belongs, a key the composer does not know. Go
	// fix the document's shape.
	errMalformed = errors.New("the declaration is not the grammar")
	// errIncomplete: it is the grammar and does not say enough for what it
	// asks. A hardfork with no node table, an upgrade naming no fork. Go add
	// what it left out.
	errIncomplete = errors.New("the declaration does not say enough")
	// errUnknownName: it names something that is not there — a binary no
	// binaries entry declares, a chain the registry does not know, a fork the
	// successor's build has never heard of. Go fix the name, or add the thing.
	errUnknownName = errors.New("the declaration names something that is not there")
	// errContradicted: the invocation and the document disagree, and preferring
	// either silently runs the other's network. Go decide which one is meant.
	errContradicted = errors.New("the invocation and the declaration disagree")

	// errNoRoot: there is nowhere to write the run.
	//
	// The plan not being writable is NOT one of these. It is kept beside the
	// record as a convenience and its failure is recorded as a note, so a run
	// whose plan could not be written is still a run that can be judged. There
	// was a kind and a state for it until the ratchet below showed nothing
	// could ever mark one.
	errNoRoot = errors.New("the run has nowhere to record itself")

	// errUnreachable: there is a network and it does not answer — no node
	// became ready, or the workspace names one that is not there.
	errUnreachable = errors.New("the network could not be reached")

	// errPrepareFork: the declared fork was not crossed.
	errPrepareFork = errors.New("the declared fork was not crossed")
	// errPrepareHeight: the chain did not reach the height a run waits for.
	errPrepareHeight = errors.New("the chain did not reach the height the run waits for")
	// errPrepareAccount: a declared account could not be created or funded.
	errPrepareAccount = errors.New("a declared account could not be prepared")

	// errCasesCannotProceed: the running itself broke. A case reporting a
	// failure is a verdict and does not come here.
	errCasesCannotProceed = errors.New("the cases could not be run")
)

// runFailure is which state an error from a run is.
//
// The default is FailStageUnclassified, and every use of it is a debt: a stage
// still returning a sentence nobody can branch on. runFailureDebt below counts
// what is left, and the number only goes down.
func runFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errUnreadable):
		return lifecycle.TestReadDeclarationFailUnreadable
	case errors.Is(err, errMalformed):
		return lifecycle.TestReadDeclarationFailMalformed
	case errors.Is(err, errIncomplete):
		return lifecycle.TestReadDeclarationFailIncomplete
	case errors.Is(err, errUnknownName):
		return lifecycle.TestReadDeclarationFailUnknownName
	case errors.Is(err, errContradicted):
		return lifecycle.TestReadDeclarationFailContradicted

	case errors.Is(err, errNoRoot):
		return lifecycle.TestOpenSessionFailNoRoot

	case errors.Is(err, errUnreachable):
		return lifecycle.TestReachNetworkFailUnreachable

	case errors.Is(err, errPrepareFork):
		return lifecycle.TestPrepareFailFork
	case errors.Is(err, errPrepareHeight):
		return lifecycle.TestPrepareFailHeight
	case errors.Is(err, errPrepareAccount):
		return lifecycle.TestPrepareFailAccount

	case errors.Is(err, errCasesCannotProceed):
		return lifecycle.TestRunCasesFailCannotProceed

	}
	return lifecycle.FailStageUnclassified
}
