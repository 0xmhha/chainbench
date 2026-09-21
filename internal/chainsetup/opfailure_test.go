package chainsetup

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// What an operational verb's failure means, as a state.
//
// The measurement these come from is state-machine-04-operational-failures.md:
// 25 of 65 sites are shared between verbs, so most of what is checked here is
// that one condition keeps one name whoever raised it.

func TestOperationalKindsHaveTheirStates(t *testing.T) {
	for _, c := range []struct {
		name     string
		classify func(error) lifecycle.Status
		kind     error
		want     lifecycle.Status
	}{
		{"a precondition, from the needs table", OpPreconditionFailure, errOpPrecondition, lifecycle.ChainOpFailPrecondition},

		{"no such node", NodeOpFailure, errOpNoSuchNode, lifecycle.ChainOpFailNoSuchNode},
		{"a swap that asks for nothing", NodeOpFailure, errOpNothingToReplace, lifecycle.ChainOpReplaceNodeFailNothingAsked},
		{"a precondition, noticed acting on a node", NodeOpFailure, errOpPrecondition, lifecycle.ChainOpFailPrecondition},

		{"not every node came down", StopFailure, errOpSomeStillUp, lifecycle.ChainOpStopNodesFailSomeStillUp},
		{"no such node, noticed stopping", StopFailure, errOpNoSuchNode, lifecycle.ChainOpFailNoSuchNode},

		{"nothing to cross", CrossForkFailure, errCrossForkNoFork, lifecycle.ChainOpCrossForkFailNoFork},
		{"a head that would not arrive", CrossForkFailure, errCrossForkHeadUnreadable, lifecycle.ChainOpCrossForkFailHeadUnreadable},
		{"a chain already past the fork", CrossForkFailure, errCrossForkAlreadyPast, lifecycle.ChainOpCrossForkFailAlreadyPast},
		{"a restart nobody came back from", CrossForkFailure, errCrossForkNobodyCameBack, lifecycle.ChainOpCrossForkFailNobodyCameBack},
	} {
		if got := c.classify(ofKind(c.kind, errors.New("x"))); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// TestBorrowedStatesKeepTheirName is the decision the measurement asked for.
//
// Deciding which binary to launch, rendering a node's config again and
// re-initializing one node's datadir are the same work the composition does, so
// an operational verb that fails at one of them reports the composition's
// state. Six verbs with six names for one condition is what this avoids, and it
// is the call the init stage's busy port already made.
func TestBorrowedStatesKeepTheirName(t *testing.T) {
	for _, c := range []struct {
		name     string
		classify func(error) lifecycle.Status
		kind     error
		want     lifecycle.Status
	}{
		{"swap, no binary", NodeOpFailure, errLaunchNoBinary, lifecycle.ChainLaunchNodesFailNoBinary},
		{"swap, busy port", NodeOpFailure, errLaunchPortBusy, lifecycle.ChainLaunchNodesFailPortBusy},
		{"swap, target cannot init", NodeOpFailure, errInitTargetUnable, lifecycle.ChainInitNodesFailTargetUnable},
		{"swap, genesis unreadable", NodeOpFailure, errInitGenesisUnreadable, lifecycle.ChainInitNodesFailGenesisUnreadable},
		{"swap, bad config override", NodeOpFailure, errConfigBadOverride, lifecycle.ChainBuildNodeConfigFailBadOverride},
		{"cross-fork, no binary", CrossForkFailure, errLaunchNoBinary, lifecycle.ChainLaunchNodesFailNoBinary},
		{"cross-fork, config readback", CrossForkFailure, errConfigReadback, lifecycle.ChainBuildNodeConfigFailReadback},
	} {
		if got := c.classify(ofKind(c.kind, errors.New("x"))); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// TestAnOperationalFailureWithNoStateSaysSo: the default is the debt state, not
// the nearest guess. What reaches it is the driver's own refusals and the core
// packages', which belong to them.
func TestAnOperationalFailureWithNoStateSaysSo(t *testing.T) {
	for name, classify := range map[string]func(error) lifecycle.Status{
		"precondition": OpPreconditionFailure, "node": NodeOpFailure,
		"stop": StopFailure, "cross-fork": CrossForkFailure,
	} {
		if got := classify(errors.New("the driver said no")); got != lifecycle.FailStageUnclassified {
			t.Errorf("%s classified an error it has no state for as %s", name, got)
		}
	}
}

// TestTheRealOperationalRefusalsCarryTheirKind drives two sites directly,
// because everything above hands the classifier an error built by hand. A kind
// attached in the test and not at the site is a classifier that passes its own
// tests and returns the debt state in production.
func TestTheRealOperationalRefusalsCarryTheirKind(t *testing.T) {
	// A verb that declares no requirements at all.
	var w Workspace
	if err := w.allow("NoSuchVerbExists"); err == nil {
		t.Fatal("a verb with no declaration was allowed")
	} else if got := OpPreconditionFailure(err); got != lifecycle.ChainOpFailPrecondition {
		t.Errorf("the real refusal classified as %s, want ChainOpFailPrecondition", got)
	}

	// An index that is not in the node table.
	if _, err := w.nodeAt(7); err == nil {
		t.Fatal("an index with no node was accepted")
	} else if got := NodeOpFailure(err); got != lifecycle.ChainOpFailNoSuchNode {
		t.Errorf("the real refusal classified as %s, want ChainOpFailNoSuchNode", got)
	}
}
