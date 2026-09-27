package chainsetup

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// TestFailureStatus_AsksTheFailingStagesClassifier: a failure raised under a
// step is classified by that step's classifier, so a composition that fails
// says which designed failure it was rather than only "CHAIN_FAILED".
func TestFailureStatus_AsksTheFailingStagesClassifier(t *testing.T) {
	for _, c := range []struct {
		step string
		kind error
		want lifecycle.Status
	}{
		{stepNew, errNewNoChain, lifecycle.ChainOpenWorkspaceFailNoChain},
		{stepPlace, errPlaceSetContended, lifecycle.ChainBuildNodeTableFailSetContended},
		{stepKeys, errKeySourceUnknown, lifecycle.ChainEnsureKeysFailUnknownSource},
		{stepGenesis, errGenesisExistingForeign, lifecycle.ChainBuildGenesisFailExistingForeign},
		{stepConfig, errConfigBadOverride, lifecycle.ChainBuildNodeConfigFailBadOverride},
		{stepBuild, errBuildBadOption, lifecycle.ChainBuildNodeCommandFailBadOption},
		{stepDeploy, errDeployInputMissing, lifecycle.ChainDeployNodesFailInputMissing},
		{stepInit, errInitDatadir, lifecycle.ChainInitNodesFailDatadir},
		{stepStart, errLaunchPortBusy, lifecycle.ChainLaunchNodesFailPortBusy},
		{"stop", errOpSomeStillUp, lifecycle.ChainOpStopNodesFailSomeStillUp},
		// An operation's precondition is one state whichever verb noticed it.
		{"swap node", errOpPrecondition, lifecycle.ChainOpFailPrecondition},
		{"restart", errOpNoSuchNode, lifecycle.ChainOpFailNoSuchNode},
	} {
		err := lifecycle.Mark(c.kind, errors.New("x"))
		if got := FailureStatus(c.step, err); got != c.want {
			t.Errorf("%s: %s, want %s", c.step, got, c.want)
		}
	}
	if got := FailureStatus(stepStart, errors.New("something the stage has no state for")); got != lifecycle.FailStageUnclassified {
		t.Errorf("an unmarked error was classified as %s", got)
	}
	if got := FailureStatus("machine", errors.New("unhandled")); got != lifecycle.FailStageUnclassified {
		t.Errorf("a machine failure was classified as %s", got)
	}
}
