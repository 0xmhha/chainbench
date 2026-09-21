package chainsetup

import (
	"context"
	"fmt"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// The composition driven by its state rather than by a list.
//
// What this replaces is a walk over the constant UpStepNames with two
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

// composeStage is everything the machine needs to know about one stage of the
// composition. One table rather than several: a stage's name, its state, what
// follows it, how it fails and what it still guesses are all facts about the
// same thing, and a stage added to one list and forgotten in another is the
// failure mode a single table removes.
type composeStage struct {
	// step is the name the record has always carried. It does not change: a
	// handler runs the same verb, and the verb marks itself.
	step string
	// at is the state this stage is entered in, and next is the stage after it.
	at   lifecycle.Status
	next lifecycle.Status
	// assumed is the path through this stage's own states taken when the step
	// does not report one.
	//
	// It is debt. A stage that fills it is claiming a route on the step's
	// behalf, and the claim is only as good as the sentence next to it. The
	// list shrinks as steps start reporting; see stagesStillAssuming.
	assumed []lifecycle.Status
	// classify turns this stage's error into the state that failure is. A stage
	// without one has failures that are still a sentence nobody can branch on,
	// and owed says why.
	classify func(error) lifecycle.Status
	// owed is why this stage cannot yet say more than "it failed". It is set
	// exactly when classify is not; see stagesStillOwing.
	owed string
}

// composition is the nine stages, in the order they run.
//
// The two counted debts are visible by reading down the columns: four stages
// still assume a path, seven still report one kind of failure. Both numbers
// only go down, and TestEveryStageIsPlaced holds them.
var composition = []composeStage{
	{
		step: "new", at: lifecycle.ChainOpenWorkspace, next: lifecycle.ChainBuildNodeTable,
		classify: NewFailure,
	},
	{
		step: "place", at: lifecycle.ChainBuildNodeTable, next: lifecycle.ChainEnsureKeys,
		classify: PlaceFailure,
	},
	{
		// The first stage whose work reports its own state. It used to be read
		// off the request, and that was wrong in a case the request cannot
		// show: a node table naming per-node keys is the source whatever the
		// request said, so a composition with an inline topology was recorded
		// as having used the preset.
		step: "keys", at: lifecycle.ChainEnsureKeys, next: lifecycle.ChainBuildGenesis,
		classify: KeysFailure,
	},
	{
		// The block with the most failures, because it is the only stage that
		// handles two chains: a network crossing a fork reads the handing
		// chain's genesis to build the receiving chain's.
		step: "genesis", at: lifecycle.ChainBuildGenesis, next: lifecycle.ChainBuildNodeConfig,
		classify: GenesisFailure,
	},
	{
		step: "config", at: lifecycle.ChainBuildNodeConfig, next: lifecycle.ChainBuildNodeCommand,
		classify: ConfigFailure,
	},
	{
		step: "build", at: lifecycle.ChainBuildNodeCommand, next: lifecycle.ChainDeployNodes,
		classify: BuildFailure,
	},
	{
		step: "deploy", at: lifecycle.ChainDeployNodes, next: lifecycle.ChainInitNodes,
		classify: DeployFailure,
	},
	{
		step: "init", at: lifecycle.ChainInitNodes, next: lifecycle.ChainLaunchNodes,
		classify: InitFailure,
	},
	{
		// The block with the most detail states, because it is the one stage
		// whose shape the family decides. The launch reports a phase's worth of
		// states per phase, so a launch that dies in the third join says so.
		step: "start", at: lifecycle.ChainLaunchNodes, next: lifecycle.ChainVerify,
		classify: LaunchFailure,
	},
}

// The two debts, counted. Lowering a number is how a stage moving in gets said
// out loud; neither ever goes up.
const (
	// stagesStillAssuming walk a path they claim on the step's behalf. None do:
	// a stage with states of its own reports the ones it went through, and one
	// with none goes straight on.
	stagesStillAssuming = 0
	// stagesStillOwing report every failure as one sentence. None do. The list
	// is kept because the shape is what a stage added later starts in, and a
	// count of zero is the thing a reader should see stay at zero.
	stagesStillOwing = 0
)

// composeRun is one step of the composition: it runs the verb, appends the
// detail to the result and marks a failure into the record. It reports the
// states the step went through, or nothing when the step does not yet say.
//
// It is the closure netUpFrom already had. Passing it in rather than rebuilding
// it means the state-driven path and the list-driven one cannot come to record
// a composition differently.
type composeRun func(step string) ([]lifecycle.Status, error)

// composeStages is the composition's stages for this run.
//
// Reconciling against a running network is a state between the key set and the
// genesis — the first stage that writes to the target — so a run that does it
// moves there from the keys instead of straight on. Judging later meant a
// refusal that had already overwritten the running network's genesis.
func composeStages(reconciling bool) []composeStage {
	if !reconciling {
		return composition
	}
	out := slices.Clone(composition)
	for i := range out {
		if out[i].step == "keys" {
			out[i].next = lifecycle.ReconcileChain
		}
	}
	return out
}

// upHandlers is one handler per composition stage, for a run that composes
// straight through.
func upHandlers(run composeRun) map[lifecycle.Status]lifecycle.Handler {
	return handlersFor(composition, run)
}

// handlersFor is one handler per stage in the given order.
//
// The reconciliation always gets a handler, even for a run that does not do it.
// The table lets the keys stage move there, so the state is reachable, and a
// machine refuses to start when a reachable stage has none — registering one
// that says what happened is what keeps the loop free of nil checks.
func handlersFor(stages []composeStage, run composeRun) map[lifecycle.Status]lifecycle.Handler {
	out := make(map[lifecycle.Status]lifecycle.Handler, len(stages)+1)
	out[lifecycle.ReconcileChain] = notReconciling
	for _, s := range stages {
		stage := s // captured by the closure
		out[stage.at] = func(_ context.Context, m *lifecycle.Machine, at lifecycle.Status) error {
			// A stage is entered at one state. Its own detail states are
			// reached inside this call, so being called at one of them means a
			// move nobody wrote — a table and a handler that disagree, which is
			// worth saying rather than guessing at.
			if at != stage.at {
				return fmt.Errorf("chainsetup: the %s stage was asked for %s, which nothing sets", stage.step, at)
			}
			passed, err := stage.run(m, run)
			if err != nil {
				return err
			}
			// The path is walked here rather than one handler call per state:
			// the step already did all of it, and these states are what it went
			// through, not further work to do. Each move still goes through the
			// table, so a path the table does not allow is refused.
			for _, s := range passed {
				if err := m.Request(s); err != nil {
					return err
				}
			}
			return m.Request(stage.next)
		}
	}
	return out
}

// notReconciling is the reconciliation's handler for a run that does not
// reconcile. Reaching it means the table and the stage list disagree.
func notReconciling(_ context.Context, _ *lifecycle.Machine, at lifecycle.Status) error {
	return fmt.Errorf("chainsetup: %s was reached by a run that does not reconcile against a running network", at)
}

// run does this stage's work and reports the path through it.
//
// A stage that neither reports a path nor assumes one is a stage with no states
// of its own, and it goes straight to the next. A stage that has states and
// reports nothing is a report that was lost, which is refused rather than
// filled in.
func (s composeStage) run(m *lifecycle.Machine, run composeRun) ([]lifecycle.Status, error) {
	passed, err := run(s.step)
	if err != nil {
		return nil, s.failed(m, passed, err)
	}
	if len(passed) > 0 {
		return passed, nil
	}
	if len(s.assumed) > 0 {
		return s.assumed, nil
	}
	if lifecycle.Detailed(s.at) {
		// A stage with states of its own has to say which it went through.
		// Silence is a report that was lost, and filling one in is what these
		// stages stopped doing. A stage with none says nothing and goes on —
		// which is a different thing from classifying its failures, and
		// confusing the two made every stage that does one demand the other.
		return nil, fmt.Errorf("chainsetup: the %s step did not say which states it went through", s.step)
	}
	return nil, nil
}

// failed puts the machine in the state this error is and returns what the
// caller should report.
//
// passed is how far the step got before it failed, and it is walked first. That
// is what makes the table's shape real: the genesis stage's fork failure can
// only happen once a genesis exists, and the table says so by listing it under
// the built states rather than under the entry. A step that reports nothing
// failed before it reached any state of its own, and the failure is reached
// from the entry.
//
// A stage that classifies lands on the state its error is. A stage that does
// not lands on the debt state, and the reason it cannot say more is appended to
// the error so the two are read together rather than one being looked up.
func (s composeStage) failed(m *lifecycle.Machine, passed []lifecycle.Status, err error) error {
	for _, p := range passed {
		if rerr := m.Request(p); rerr != nil {
			return rerr
		}
	}
	at := lifecycle.FailStageUnclassified
	if s.classify != nil {
		at = s.classify(err)
	}
	if rerr := m.Request(at); rerr != nil {
		return rerr
	}
	// Reaching the debt state has to say why, either way. A stage with no
	// classifier says what it cannot tell apart; a stage that has one and fell
	// to its default says that this particular failure has no state — which is
	// the case a reader would otherwise take for a stage nobody has moved in.
	if at == lifecycle.FailStageUnclassified {
		switch {
		case s.owed != "":
			return fmt.Errorf("%w (%s: %s)", err, s.step, s.owed)
		case s.classify != nil:
			return fmt.Errorf("%w (%s: this failure has no state of its own)", err, s.step)
		}
	}
	return err
}

// startFor is the state a composition resumes at. An empty step is the whole
// composition, which starts where it always did.
func startFor(from string) (lifecycle.Status, error) {
	if from == "" {
		return lifecycle.ChainOpenWorkspace, nil
	}
	for _, s := range composition {
		if s.step == from {
			return s.at, nil
		}
	}
	return 0, fmt.Errorf("chainsetup: no stage is named %q", from)
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
