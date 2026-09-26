package chainsetup

import (
	"strings"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// FailureStatus is which failure state a stage's error is, asked of the
// classifier written beside the stage that raised it.
//
// Each stage marks its errors with the kinds it knows (lifecycle.Mark) and has
// a classifier that turns a marked error into a state. Nothing asked them: a
// failed composition reported CHAIN_FAILED and the message, and which of the
// designed failure states it was — a busy port, an unknown key source, a
// genesis of another network — was left for the reader to work out from prose.
//
// The step names are the ones a failure is raised under: the nine compose
// steps, the fork crossing, and the operations, which report their verb.
func FailureStatus(step string, err error) lifecycle.Status {
	if err == nil {
		return 0
	}
	var s lifecycle.Status
	switch step {
	case stepNew:
		s = NewFailure(err)
	case stepPlace:
		s = PlaceFailure(err)
	case stepKeys:
		s = KeysFailure(err)
	case stepGenesis:
		s = GenesisFailure(err)
	case stepConfig:
		s = ConfigFailure(err)
	case stepBuild:
		s = BuildFailure(err)
	case stepDeploy:
		s = DeployFailure(err)
	case stepInit:
		s = InitFailure(err)
	case stepStart:
		s = LaunchFailure(err)
	case crossForkStep:
		s = CrossForkFailure(err)
	case "stop":
		s = StopFailure(err)
	default:
		// The operations on one node (restart, swap node, rm, …) share one
		// classifier; anything it does not know is left to the precondition
		// check every operation makes.
		if strings.Contains(step, " ") || step == "restart" || step == "rm" || step == "hardfork" {
			s = NodeOpFailure(err)
		} else {
			s = lifecycle.FailStageUnclassified
		}
	}
	if s == lifecycle.FailStageUnclassified {
		if p := OpPreconditionFailure(err); p != lifecycle.FailStageUnclassified {
			return p
		}
	}
	return s
}
