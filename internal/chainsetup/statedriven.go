package chainsetup

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// The composition driven by its state rather than by a list.
//
// What this replaces is a walk over the constant upStepNames with two
// comparisons wedged into the middle of it: one asking on every iteration
// whether the step just reached is one the requested stage stops before, and
// one asking whether this is the step after which a running network gets
// reconciled. Both are string comparisons on a step's name, which is where a
// rule goes when it has nowhere of its own to live. Here the first is the
// machine's target and the second is a state.
//
// The stages still do their work through the same verbs, and the verbs still
// record themselves, so what a composition writes and what its record says do
// not change. Only what decides which stage runs next.

// unclassified is why a stage still reports lifecycle.FailStageUnclassified
// when it fails, one entry per stage that does.
//
// It is a debt list and it only shrinks, which is the rule the other debt lists
// in this repository keep. A stage leaves when its work moves out of the verb
// and into its handler, because that is when the handler can tell its failures
// apart: today every one of these verbs fails with an error built by
// fmt.Errorf, and no handler can branch on a sentence.
//
// Eight now. The keys stage left when its four declared failures became kinds
// a caller can branch on, which is what a stage leaving looks like.
var unclassified = map[string]string{
	"new":     "NetNew reports a missing chain and a bad request the same way",
	"place":   "NetAllocate reports two layouts and a contended server set the same way",
	"genesis": "NetGenesis reports a foreign existing genesis and an unresolved fork the same way",
	"config":  "NetConfig reports a bad override and a failed readback the same way",
	"build":   "NetLaunchOpts reports every bad option the same way",
	"deploy":  "NetProvision reports a missing input and a foreign one the same way",
	"init":    "NetInit reports an unreachable target and an unreadable genesis the same way",
	"start":   "NetStart reports no binary, a busy port and an occupied datadir the same way",
}

// stageOf is the lifecycle stage each of the composition's steps is.
//
// The names on the left are the ones the record has always carried and they do
// not change: a handler runs the same verb, and the verb marks itself.
var stageOf = map[string]lifecycle.Status{
	"new":     lifecycle.ChainOpenWorkspace,
	"place":   lifecycle.ChainBuildNodeTable,
	"keys":    lifecycle.ChainEnsureKeys,
	"genesis": lifecycle.ChainBuildGenesis,
	"config":  lifecycle.ChainBuildNodeConfig,
	"build":   lifecycle.ChainBuildNodeCommand,
	"deploy":  lifecycle.ChainDeployNodes,
	"init":    lifecycle.ChainInitNodes,
	"start":   lifecycle.ChainLaunchNodes,
}

// composeRun is one step of the composition: it runs the verb, appends the
// detail to the result and marks a failure into the record. It reports the
// state the step ended in, or zero when the step does not yet say.
//
// It is the closure netUpFrom already had. Passing it in rather than rebuilding
// it means the state-driven path and the list-driven one cannot come to record
// a composition differently.
type composeRun func(step string) (lifecycle.Status, error)

// upHandlers is one handler per composition stage.
//
// Each holds its whole stage: the verb it runs and every state inside it. The
// stages with detail states are the reason these are written out rather than
// generated from a table — a stage that has more than one way to go decides
// which one it took, and that decision is the stage's own knowledge.
func upHandlers(in NetUpIn, run composeRun) map[lifecycle.Status]lifecycle.Handler {
	return map[lifecycle.Status]lifecycle.Handler{
		lifecycle.ChainOpenWorkspace:   plainStage("new", lifecycle.ChainBuildNodeTable, run),
		lifecycle.ChainBuildNodeTable:  plainStage("place", lifecycle.ChainEnsureKeys, run),
		lifecycle.ChainEnsureKeys:      keysStage(run),
		lifecycle.ChainBuildGenesis:    genesisStage(in, run),
		lifecycle.ChainBuildNodeConfig: plainStage("config", lifecycle.ChainBuildNodeCommand, run),
		lifecycle.ChainBuildNodeCommand: plainStage("build",
			lifecycle.ChainDeployNodes, run),
		lifecycle.ChainDeployNodes: plainStage("deploy", lifecycle.ChainInitNodes, run),
		lifecycle.ChainInitNodes:   plainStage("init", lifecycle.ChainLaunchNodes, run),
		lifecycle.ChainLaunchNodes: launchStage(run),
	}
}

// plainStage is a stage with one way through: run the verb, move on.
func plainStage(step string, next lifecycle.Status, run composeRun) lifecycle.Handler {
	return func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		if at != stageOf[step] {
			return unexpected(step, at)
		}
		if _, err := runStep(m, step, run); err != nil {
			return err
		}
		return m.Request(next)
	}
}

// keysStage is the key stage: the source the identities came from, and the four
// ways it refuses.
//
// This is the first stage whose work reports its own state. It used to be read
// off the request, and that was wrong in a case the request cannot show: a node
// table that names per-node keys is the source whatever the request said, so a
// composition with an inline topology was recorded as having used the preset.
// The step says which source it took, and this passes it on.
func keysStage(run composeRun) lifecycle.Handler {
	return func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		switch at {
		case lifecycle.ChainEnsureKeys:
			from, err := runStep(m, "keys", run)
			if err != nil {
				return err
			}
			if from == 0 {
				return fmt.Errorf("chainsetup: the keys step did not say which source it used")
			}
			return m.Request(from)
		case lifecycle.ChainEnsureKeysFromPreset,
			lifecycle.ChainEnsureKeysGenerated,
			lifecycle.ChainEnsureKeysFromBlueprint:
			return m.Request(lifecycle.ChainBuildGenesis)
		}
		return unexpected("keys", at)
	}
}

// keysFailure is which of the key stage's failures this error is.
//
// The default is the debt state and not a guess. Two things still reach it: the
// preconditions the transition table makes unreachable in a composition but not
// in a bare `chain keys`, and whatever the key store itself refuses when it
// writes the set. Neither has a state, and naming one of the four would say
// something the error does not.
func keysFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errKeySourceUnknown):
		return lifecycle.ChainEnsureKeysFailUnknownSource
	case errors.Is(err, errKeyCountShort):
		return lifecycle.ChainEnsureKeysFailCountMismatch
	case errors.Is(err, errKeyRefNotLocal):
		return lifecycle.ChainEnsureKeysFailKeyNotLocal
	case errors.Is(err, errKeyUnreadable):
		return lifecycle.ChainEnsureKeysFailKeyUnreadable
	}
	return lifecycle.FailStageUnclassified
}

// genesisStage is the genesis stage: where the genesis came from, then what was
// applied on top of it.
//
// The four states are not four routes. The first pair says whether a genesis
// was built or taken verbatim, and the second pair says what else the request
// asked for — a scheduled fork, a separate genesis per binary — so a request
// that asks for neither goes straight on.
func genesisStage(in NetUpIn, run composeRun) lifecycle.Handler {
	return func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		switch at {
		case lifecycle.ChainBuildGenesis:
			if _, err := runStep(m, "genesis", run); err != nil {
				return err
			}
			if in.GenesisExisting != "" {
				return m.Request(lifecycle.ChainBuildGenesisFromExisting)
			}
			return m.Request(lifecycle.ChainBuildGenesisFromTemplate)
		case lifecycle.ChainBuildGenesisFromTemplate, lifecycle.ChainBuildGenesisFromExisting:
			if in.GenesisFork != nil {
				return m.Request(lifecycle.ChainBuildGenesisForkApplied)
			}
			fallthrough
		case lifecycle.ChainBuildGenesisForkApplied:
			if len(in.GenesisPerBinary) > 0 {
				return m.Request(lifecycle.ChainBuildGenesisVariantsWritten)
			}
			fallthrough
		case lifecycle.ChainBuildGenesisVariantsWritten:
			return m.Request(lifecycle.ChainBuildNodeConfig)
		}
		return unexpected("genesis", at)
	}
}

// launchStage is the launch and the pass it makes.
//
// The launch verb runs every phase the family declares inside one call, so this
// handler makes one pass: it launches, then reports the pass done. The
// per-phase walk the states describe arrives when the phase loop moves out of
// Workspace.Start and into this handler, and until then this stage claims one
// pass because one is what it made.
func launchStage(run composeRun) lifecycle.Handler {
	return func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		switch at {
		case lifecycle.ChainLaunchNodes:
			if _, err := runStep(m, "start", run); err != nil {
				return err
			}
			return m.Request(lifecycle.ChainLaunchNodesPhaseLaunching)
		case lifecycle.ChainLaunchNodesPhaseLaunching:
			return m.Request(lifecycle.ChainLaunchNodesPhaseDone)
		case lifecycle.ChainLaunchNodesPhaseDone:
			return m.Request(lifecycle.ChainVerify)
		}
		return unexpected("start", at)
	}
}

// runStep runs one step's verb and, when it fails, puts the machine in the
// state that failure is.
//
// A stage that classifies its failures has an entry in failureOf and lands on
// the state its error is; a stage that does not lands on the debt state, and
// the reason it cannot say more is appended to the error so the two are read
// together rather than one being looked up.
func runStep(m *lifecycle.Machine, step string, run composeRun) (lifecycle.Status, error) {
	reached, err := run(step)
	if err == nil {
		return reached, nil
	}
	at := lifecycle.FailStageUnclassified
	if classify, ok := failureOf[step]; ok {
		at = classify(err)
	}
	if rerr := m.Request(at); rerr != nil {
		return 0, rerr
	}
	if at == lifecycle.FailStageUnclassified {
		if why, owed := unclassified[step]; owed {
			return 0, fmt.Errorf("%w (%s: %s)", err, step, why)
		}
	}
	return 0, err
}

// failureOf is, per stage, how its error becomes a state. A stage without an
// entry here is one whose failures are still a sentence, which is what
// unclassified lists.
var failureOf = map[string]func(error) lifecycle.Status{
	"keys": keysFailure,
}

// unexpected is what a handler says when it is asked for a state of its own
// stage that nothing sets. It is worth saying rather than guessing at: a stage
// reached by a move nobody wrote is a table and a handler that disagree.
func unexpected(step string, at lifecycle.Status) error {
	return fmt.Errorf("chainsetup: the %s stage was asked for %s, which nothing sets", step, at)
}

// startFor is the state a composition resumes at. An empty step is the whole
// composition, which starts where it always did.
func startFor(from string) (lifecycle.Status, error) {
	if from == "" {
		return lifecycle.ChainOpenWorkspace, nil
	}
	s, ok := stageOf[from]
	if !ok {
		return 0, fmt.Errorf("chainsetup: no stage is named %q", from)
	}
	return s, nil
}

// targetFor is how far a composition runs, as the state it stops on.
//
// The machine stops when it REACHES the target, before running the handler that
// owns it, so a stage stops the walk by naming the stage after it. This is what
// the stage comparison inside the old loop did on every iteration.
func targetFor(stage UpStage) (lifecycle.Status, error) {
	switch stage {
	case UpDeploy:
		return lifecycle.ChainInitNodes, nil
	case UpStart, "":
		return lifecycle.ChainVerify, nil
	}
	return 0, fmt.Errorf("chainsetup: unknown stage %q (want %s or %s)", stage, UpDeploy, UpStart)
}
