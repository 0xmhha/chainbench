package chainsetup

import (
	"context"
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
// Nine now, one per composition stage.
var unclassified = map[string]string{
	"new":     "NetNew reports a missing chain and a bad request the same way",
	"place":   "NetAllocate reports two layouts and a contended server set the same way",
	"keys":    "NetKeys reports an unknown source and an unreadable key the same way",
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
// detail to the result and marks a failure into the record.
//
// It is the closure netUpFrom already had. Passing it in rather than rebuilding
// it means the state-driven path and the list-driven one cannot come to record
// a composition differently.
type composeRun func(step string) error

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
		lifecycle.ChainEnsureKeys:      keysStage(in, run),
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
		if err := runStep(m, step, run); err != nil {
			return err
		}
		return m.Request(next)
	}
}

// keysStage is the key stage and its three sources.
//
// Which source was used is read from the request rather than from the verb,
// because the verb resolves it from the same two fields: a source names itself,
// and a silent source with a declaration next to it is the declaration. The
// handler asks only after the verb has succeeded, so a declaration that did not
// parse has already become a failure rather than a source.
func keysStage(in NetUpIn, run composeRun) lifecycle.Handler {
	return func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
		switch at {
		case lifecycle.ChainEnsureKeys:
			if err := runStep(m, "keys", run); err != nil {
				return err
			}
			return m.Request(keysSourceState(in))
		case lifecycle.ChainEnsureKeysFromPreset,
			lifecycle.ChainEnsureKeysGenerated,
			lifecycle.ChainEnsureKeysFromBlueprint:
			return m.Request(lifecycle.ChainBuildGenesis)
		}
		return unexpected("keys", at)
	}
}

// keysSourceState is which of the three key sources the request names.
func keysSourceState(in NetUpIn) lifecycle.Status {
	switch in.KeysSource {
	case "generate":
		return lifecycle.ChainEnsureKeysGenerated
	case "declared":
		return lifecycle.ChainEnsureKeysFromBlueprint
	case "":
		if in.BlueprintPath != "" {
			return lifecycle.ChainEnsureKeysFromBlueprint
		}
	}
	return lifecycle.ChainEnsureKeysFromPreset
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
			if err := runStep(m, "genesis", run); err != nil {
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
			if err := runStep(m, "start", run); err != nil {
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
// failure state that says which stage failed and that nobody has classified its
// failures yet.
func runStep(m *lifecycle.Machine, step string, run composeRun) error {
	if err := run(step); err != nil {
		if rerr := m.Request(lifecycle.FailStageUnclassified); rerr != nil {
			return rerr
		}
		return fmt.Errorf("%w (%s: %s)", err, step, unclassified[step])
	}
	return nil
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
