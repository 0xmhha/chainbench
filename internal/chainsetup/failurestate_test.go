package chainsetup

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// What each stage's failure means, as a state.
//
// These live with the stages rather than with the walk: the kinds are this
// package's, each classifier is written beside the code that raises the errors
// it names, and "which state is this error" is a question about the error. What
// the walk does with the answer is the verb package's test.

// TestEveryKindHasItsState drives every classifier over every kind it names,
// and then over one it does not.
func TestEveryKindHasItsState(t *testing.T) {
	for _, c := range []struct {
		name     string
		classify func(error) lifecycle.Status
		kind     error
		want     lifecycle.Status
	}{
		{"unknown key source", KeysFailure, errKeySourceUnknown, lifecycle.ChainEnsureKeysFailUnknownSource},
		{"too few identities", KeysFailure, errKeyCountShort, lifecycle.ChainEnsureKeysFailCountMismatch},
		{"a key that is not local", KeysFailure, errKeyRefNotLocal, lifecycle.ChainEnsureKeysFailKeyNotLocal},
		{"an unreadable key", KeysFailure, errKeyUnreadable, lifecycle.ChainEnsureKeysFailKeyUnreadable},

		{"a genesis that is not usable", GenesisFailure, errGenesisExistingInvalid, lifecycle.ChainBuildGenesisFailExistingInvalid},
		{"a genesis of another network", GenesisFailure, errGenesisExistingForeign, lifecycle.ChainBuildGenesisFailExistingForeign},
		{"a fork with nothing to resolve to", GenesisFailure, errGenesisForkUnresolved, lifecycle.ChainBuildGenesisFailForkUnresolved},
		{"a declaration nobody runs", GenesisFailure, errGenesisDeclUnused, lifecycle.ChainBuildGenesisFailDeclUnused},
		{"a target that cannot build it", GenesisFailure, errGenesisTargetUnable, lifecycle.ChainBuildGenesisFailTargetUnable},

		{"no binary", LaunchFailure, errLaunchNoBinary, lifecycle.ChainLaunchNodesFailNoBinary},
		{"a busy port", LaunchFailure, errLaunchPortBusy, lifecycle.ChainLaunchNodesFailPortBusy},
		{"an occupied machine", LaunchFailure, errLaunchOccupied, lifecycle.ChainLaunchNodesFailOccupied},
		{"a producer with no keystore", LaunchFailure, errLaunchNoKeystore, lifecycle.ChainLaunchNodesFailNoKeystore},
		{"a phase with nowhere to act", LaunchFailure, errLaunchPhaseEmpty, lifecycle.ChainLaunchNodesFailPhaseEmpty},

		{"a target that cannot initialize", InitFailure, errInitTargetUnable, lifecycle.ChainInitNodesFailTargetUnable},
		{"a genesis that cannot be read back", InitFailure, errInitGenesisUnreadable, lifecycle.ChainInitNodesFailGenesisUnreadable},
		{"a datadir that will not clear", InitFailure, errInitDatadir, lifecycle.ChainInitNodesFailDatadir},
		// The init stage asks the launch's question before it writes anything,
		// and the answer keeps the launch's name: what failed is the ports the
		// launch needs, whoever noticed.
		{"a busy port, noticed by init", InitFailure, errLaunchPortBusy, lifecycle.ChainLaunchNodesFailPortBusy},

		{"a bad override", ConfigFailure, errConfigBadOverride, lifecycle.ChainBuildNodeConfigFailBadOverride},
		{"a config that did not read back", ConfigFailure, errConfigReadback, lifecycle.ChainBuildNodeConfigFailReadback},
		{"a pinned input that cannot be read", ConfigFailure, errConfigPinUnreadable, lifecycle.ChainBuildNodeConfigFailPinUnreadable},

		{"a layout given twice", PlaceFailure, errPlaceTwoLayouts, lifecycle.ChainBuildNodeTableFailTwoLayouts},
		{"a contended server set", PlaceFailure, errPlaceSetContended, lifecycle.ChainBuildNodeTableFailSetContended},

		{"an input that is not there", DeployFailure, errDeployInputMissing, lifecycle.ChainDeployNodesFailInputMissing},
		{"an input somebody else wrote", DeployFailure, errDeployInputForeign, lifecycle.ChainDeployNodesFailInputForeign},

		{"no chain named", NewFailure, errNewNoChain, lifecycle.ChainOpenWorkspaceFailNoChain},
		{"a bad launch option", BuildFailure, errBuildBadOption, lifecycle.ChainBuildNodeCommandFailBadOption},
	} {
		if got := c.classify(ofKind(c.kind, errors.New("x"))); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	// A failure none of a stage's states covers is said to be one rather than
	// guessed at, and every classifier answers the same way.
	for name, classify := range map[string]func(error) lifecycle.Status{
		"keys": KeysFailure, "genesis": GenesisFailure, "start": LaunchFailure,
		"init": InitFailure, "config": ConfigFailure, "place": PlaceFailure,
		"deploy": DeployFailure, "new": NewFailure, "build": BuildFailure,
	} {
		if got := classify(errors.New("something else refused")); got != lifecycle.FailStageUnclassified {
			t.Errorf("%s classified an error it has no state for as %s", name, got)
		}
	}
}

// TestAKindKeepsTheMessage pins why the kinds ride alongside the error instead
// of in front of it: the sentence an operator reads is the one the step wrote.
func TestAKindKeepsTheMessage(t *testing.T) {
	inner := errors.New(`chainsetup: keys: node2: key reference "k.txt" is not a readable file`)
	m := ofKind(errKeyRefNotLocal, inner)
	if m.Error() != inner.Error() {
		t.Errorf("the message changed: %q", m.Error())
	}
	if !errors.Is(m, errKeyRefNotLocal) || !errors.Is(m, inner) {
		t.Error("a marked error lost either its kind or its cause")
	}
	if ofKind(errKeyRefNotLocal, nil) != nil {
		t.Error("a nil error came back marked")
	}
}

// TestTheRealRefusalsCarryTheirKind drives four refusal sites directly, because
// everything above this point hands the classifier an error built by hand.
//
// A kind attached in the test and not at the site is a classifier that passes
// its own tests and returns the debt state in production. These four are
// reachable without a composed workspace; the rest are covered by a live run.
func TestTheRealRefusalsCarryTheirKind(t *testing.T) {
	// A key named as something other than a local file.
	kerr := checkNodeKeyRef(2, "srv://build1/keys/node2.key")
	if kerr == nil {
		t.Fatal("a key on a server was accepted")
	}
	if got := KeysFailure(kerr); got != lifecycle.ChainEnsureKeysFailKeyNotLocal {
		t.Errorf("the real refusal classified as %s, want ChainEnsureKeysFailKeyNotLocal", got)
	}

	// A fork with no name. applyFork refuses before it reads any state, so a
	// zero workspace is enough to reach it.
	var w Workspace
	_, _, ferr := w.applyFork([]byte(`{}`), GenesisFork{})
	if ferr == nil {
		t.Fatal("a fork with no name was accepted")
	}
	if got := GenesisFailure(ferr); got != lifecycle.ChainBuildGenesisFailForkUnresolved {
		t.Errorf("the real refusal classified as %s, want ChainBuildGenesisFailForkUnresolved", got)
	}

	// An override that is not key=value, refused where it is set.
	var w2 Workspace
	oerr := w2.RecordConfigSet("all", []string{"nocolonhere"})
	if oerr == nil {
		t.Fatal("an override that is not key=value was accepted")
	}
	if got := ConfigFailure(oerr); got != lifecycle.ChainBuildNodeConfigFailBadOverride {
		t.Errorf("the real refusal classified as %s, want ChainBuildNodeConfigFailBadOverride", got)
	}

	// No binary at all, which checkBinary refuses before it touches a target.
	berr := checkBinary(context.Background(), nil, "")
	if berr == nil {
		t.Fatal("a launch with no binary was accepted")
	}
	if got := LaunchFailure(berr); got != lifecycle.ChainLaunchNodesFailNoBinary {
		t.Errorf("the real refusal classified as %s, want ChainLaunchNodesFailNoBinary", got)
	}

	// A blueprint and a topology at once.
	if perr := OneLayoutOnly(true, true); perr == nil {
		t.Fatal("a layout described twice was accepted")
	} else if got := PlaceFailure(perr); got != lifecycle.ChainBuildNodeTableFailTwoLayouts {
		t.Errorf("the real refusal classified as %s, want ChainBuildNodeTableFailTwoLayouts", got)
	}
}

// TestHaveReportsTheChainIdItWasComposedWith closes a gap that cost every
// run naming a chain id a full rebuild.
//
// preflight compares a recorded chain id against a wanted one, and Have never
// filled its side. So a workspace composed with --chain-id 9911 reported
// "chain id: have 0, want 9911" against the very request that built it, and
// the verdict was rebuild-all every time. It is read from the recorded request,
// the same place and for the same reason as the genesis digest — which already
// contains it, so the check was never what caught a real change.
func TestHaveReportsTheChainIdItWasComposedWith(t *testing.T) {
	req := NetUpIn{Chain: "stablenet", ChainID: 9911}
	w := &Workspace{state: State{Chain: "stablenet", Request: &req}}

	have := w.Have(context.Background())
	if have.ChainID != 9911 {
		t.Fatalf("Have reports chain id %d, want the 9911 it was composed with", have.ChainID)
	}
	// And a workspace with no recorded request still says nothing rather than
	// claiming a chain id it does not know.
	bare := &Workspace{state: State{Chain: "stablenet"}}
	if got := bare.Have(context.Background()).ChainID; got != 0 {
		t.Errorf("a workspace with no request reports chain id %d", got)
	}
}
