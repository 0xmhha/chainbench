package lifecycle

import (
	"context"
	"errors"
	"sort"
	"testing"
)

// These are the checks that make the table worth writing down. A transition
// table nobody verifies is a comment with commas in it: it can name a state
// that does not exist, leave one nothing reaches, or drift from the dependency
// order the composition has always had, and none of that shows up until
// something runs.

// TestEveryStateIsNamed holds the value table and the name table together.
//
// The names are not decoration. The machine's refusals print them, and a state
// that prints as a hex number is one a reader has to look up in the source to
// understand — at the moment they are reading a failure, which is the worst
// time to send them somewhere else.
func TestEveryStateIsNamed(t *testing.T) {
	for from, tos := range allowed {
		if _, ok := names[from]; !ok {
			t.Errorf("the table moves out of %#x, which nothing names", uint32(from))
		}
		for _, to := range tos {
			if _, ok := names[to]; !ok {
				t.Errorf("%s moves to %#x, which nothing names", from, uint32(to))
			}
		}
	}
}

// TestEveryNamedStateIsReachable is the other direction: a state nothing can
// reach is a state that will never happen, and one that is named and never
// happens reads like a case somebody forgot to wire.
//
// The entry states of the two areas and the common failures are the exceptions,
// because a run starts at one and can fall to another from anywhere.
func TestEveryNamedStateIsReachable(t *testing.T) {
	reached := map[Status]bool{
		ChainOpenWorkspace: true, AdoptChain: true,
	}
	for _, tos := range allowed {
		for _, to := range tos {
			reached[to] = true
		}
	}
	var orphan []string
	for s, n := range names {
		if reached[s] || s.Block() == areaCommon {
			continue
		}
		orphan = append(orphan, n)
	}
	sort.Strings(orphan)
	for _, n := range orphan {
		t.Errorf("%s is named and nothing reaches it", n)
	}
}

// TestFailuresAreTerminal pins the decision that no failure retries.
//
// It is written as a check rather than left to reading because the two failures
// that looked retryable were not, and the reasons were in two other files: the
// contended server set has already polled for ten seconds before this state is
// reached, and a busy port is held by a process somebody has to stop. If a
// later change adds a retry edge, it should have to come here and say so.
func TestFailuresAreTerminal(t *testing.T) {
	for from, tos := range allowed {
		if !from.IsFailure() {
			continue
		}
		t.Errorf("%s is a failure and moves to %v — failures end the walk", from, tos)
	}
}

// TestStagesKeepTheCompositionOrder is the check the dependency table becomes.
//
// chainsetup declares what each step reaches for directly (place needs new,
// config needs place and keys, deploy needs place, genesis and config, and so
// on). That table and the order the steps run in are different things, and
// today the two are reconciled at RUN time by require(), which is after the
// workspace is open and the earlier steps have already written.
//
// Here it is a comparison between two tables, so it happens at build time: walk
// the composition and check that every stage's dependencies are stages already
// passed.
func TestStagesKeepTheCompositionOrder(t *testing.T) {
	// what chainsetup's composeNeeds says, in this package's names
	needs := map[Status][]Status{
		ChainBuildNodeTable:   {ChainOpenWorkspace},
		ChainEnsureKeys:       {ChainOpenWorkspace},
		ChainBuildGenesis:     {ChainBuildNodeTable},
		ChainBuildNodeConfig:  {ChainBuildNodeTable, ChainEnsureKeys},
		ChainBuildNodeCommand: {ChainBuildNodeTable, ChainEnsureKeys},
		ChainDeployNodes:      {ChainBuildNodeTable, ChainBuildGenesis, ChainBuildNodeConfig},
		ChainInitNodes:        {ChainDeployNodes},
		ChainLaunchNodes:      {ChainInitNodes},
	}
	passed := map[Status]bool{}
	for _, s := range walkComposition(t) {
		for _, need := range needs[s] {
			if !passed[need] {
				t.Errorf("%s runs before %s, which it needs", s, need)
			}
		}
		passed[s] = true
	}
}

// walkComposition follows the table from the first stage to the last, taking
// the non-failure move each time, and returns the stages in the order a run
// meets them.
func walkComposition(t *testing.T) []Status {
	t.Helper()
	var order []Status
	blocks := map[Status]bool{}
	states := map[Status]bool{}
	at := ChainOpenWorkspace
	for range 100 {
		states[at] = true
		if b := at.Block(); !blocks[b] {
			blocks[b] = true
			order = append(order, b)
		}
		if at == ChainReady {
			return order
		}
		// Somewhere not yet visited — by STATE, because a stage's own detail
		// states are inside a block already counted.
		next := Status(0)
		for _, to := range allowed[at] {
			if !to.IsFailure() && !states[to] {
				next = to
				break
			}
		}
		if next == 0 {
			t.Fatalf("the walk stopped at %s with nowhere new to go", at)
		}
		at = next
	}
	t.Fatal("the walk did not reach ChainReady in a hundred moves")
	return nil
}

// TestNewRefusesAMissingHandler is why the loop has no nil check.
//
// A handler looked up per step has to be checked per step, and every check is a
// place where a stage nobody wrote becomes a stage quietly skipped. Refusing at
// construction moves that to one place, before anything has run.
func TestNewRefusesAMissingHandler(t *testing.T) {
	_, err := New(ChainOpenWorkspace, ChainReady, map[Status]Handler{
		ChainOpenWorkspace: func(context.Context, *Machine, Status) error { return nil },
	})
	if err == nil {
		t.Fatal("a machine was built with handlers for one stage out of ten")
	}
}

// TestRequestRefusesAMoveTheTableDoesNotHave is the rule that replaces the
// hand-written "run `chain place` first" checks: a stage cannot be entered from
// somewhere that has not been through what precedes it, and the refusal says so
// without any stage having to write the sentence.
func TestRequestRefusesAMoveTheTableDoesNotHave(t *testing.T) {
	m := must(t, ChainOpenWorkspace, ChainReady)
	if err := m.Request(ChainLaunchNodes); err == nil {
		t.Fatal("a machine at the first stage was allowed to jump to the last")
	}
	if m.At() != ChainOpenWorkspace {
		t.Errorf("a refused move still moved: now at %s", m.At())
	}
}

// TestCommonFailureIsReachableFromAnywhere keeps the workspace's own failures
// out of every stage's move list. What they report is not the stage failing.
func TestCommonFailureIsReachableFromAnywhere(t *testing.T) {
	m := must(t, ChainOpenWorkspace, ChainReady)
	if err := m.Request(FailRecordSave); err != nil {
		t.Fatalf("a common failure was refused: %v", err)
	}
	if m.At() != FailRecordSave {
		t.Errorf("at %s, want FailRecordSave", m.At())
	}
}

// TestRunStopsOnAStallingHandler catches the shape a skipped stage takes: a
// handler that returns nil and asks for nothing leaves the machine where it
// was, and a loop that trusted it would spin.
func TestRunStopsOnAStallingHandler(t *testing.T) {
	handlers := everyStage(func(context.Context, *Machine, Status) error { return nil })
	m, err := New(ChainOpenWorkspace, ChainReady, handlers)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Run(context.Background()); err == nil {
		t.Fatal("a handler that asked for nothing was taken as progress")
	}
}

// TestRunReachesReady walks the whole composition with handlers that do nothing
// but move, which is what the adapters in step 5 will be before they hold the
// real work.
func TestRunReachesReady(t *testing.T) {
	// A handler that moves and nothing else. It refuses to take a move back
	// into a state it has already been in, which is the job a real handler does
	// with its own knowledge — the launch stage stops going round when the
	// family runs out of phases.
	been := map[Status]bool{}
	handlers := everyStage(func(_ context.Context, m *Machine, at Status) error {
		been[at] = true
		for _, to := range allowed[at] {
			if !to.IsFailure() && !been[to] {
				return m.Request(to)
			}
		}
		return errors.New("nowhere left to go from " + at.String())
	})
	m, err := New(ChainOpenWorkspace, ChainReady, handlers)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Run(context.Background()); err != nil {
		t.Fatalf("the walk did not reach ChainReady: %v (stopped at %s)", err, m.At())
	}
}

// TestEntryLimitStopsALoop pins the one backwards move. A comparison may decide
// to compose again, and today that decision is taken once; a second arrival is
// a loop rather than progress.
func TestEntryLimitStopsALoop(t *testing.T) {
	m := must(t, ChainOpenWorkspace, ChainReady)
	if err := m.Request(ChainBuildNodeTable); err != nil {
		t.Fatal(err)
	}
	if err := m.Request(ChainEnsureKeys); err != nil {
		t.Fatal(err)
	}
	// back round to a block already entered
	m.at = CompareChainNetworkDiffers
	if err := m.Request(ChainOpenWorkspace); err == nil {
		t.Fatal("a block was entered a second time")
	}
	if m.At() != FailLoop {
		t.Errorf("at %s, want FailLoop", m.At())
	}
}

func everyStage(h Handler) map[Status]Handler {
	out := map[Status]Handler{}
	for s := range allowed {
		out[s.Block()] = h
	}
	return out
}

func must(t *testing.T, start, target Status) *Machine {
	t.Helper()
	m, err := New(start, target, everyStage(
		func(context.Context, *Machine, Status) error { return nil }))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestAreasAreFarApart holds the one number the whole vocabulary rests on.
//
// Every state's value is its area plus its block plus its slot, and nothing
// checks that arithmetic at compile time. Areas 0x1000 apart give each of them
// sixteen blocks of 0x100, so an area that grew past sixteen stages would start
// writing states that belong to the next one — and the collision would show up
// as a handler running for a stage it does not own, which is a bug that reads
// like a logic error rather than like a numbering one.
//
// The operational and test areas have no states yet. They are checked here
// because the spacing is the reason their bases were chosen now rather than
// when their first state is written.
func TestAreasAreFarApart(t *testing.T) {
	areas := map[string]Status{
		"chain": areaChain, "adopt": areaAdopt,
		"operational": areaChainOp, "test": areaTest,
	}
	const gap = 0x1000
	for an, a := range areas {
		if a%gap != 0 {
			t.Errorf("the %s area is %#x, which is not on a %#x boundary", an, uint32(a), gap)
		}
		for bn, b := range areas {
			if an >= bn {
				continue
			}
			if a == b {
				t.Errorf("the %s and %s areas are both %#x", an, bn, uint32(a))
			}
		}
	}
	// The common failures sit below every area so that adding an area cannot
	// walk over them.
	if areaCommon >= areaChain {
		t.Errorf("the common block is %#x, at or above the first area %#x", uint32(areaCommon), uint32(areaChain))
	}
	// Sixteen blocks per area, and the composition uses eleven of them.
	if blocks := (areaAdopt - areaChain) / blockSize; blocks != 0x10 {
		t.Errorf("an area holds %d blocks, want 16", blocks)
	}
}

// TestDetailedIsTheTableAndNotAGuess pins the distinction the walk rests on: a
// stage the table gives no way out of has to say which of its own states it
// went through, and one that may move on without naming a detail does not.
//
// Both kinds exist and the difference is not visible from "has detail states".
// Getting it wrong in either direction is silent: demand a report from a stage
// that has nothing to report and every composition stops; let one off and a
// stage that lost its report walks a path nobody took.
func TestDetailedIsTheTableAndNotAGuess(t *testing.T) {
	for _, c := range []struct {
		at   Status
		want bool
		why  string
	}{
		{ChainEnsureKeys, true, "its three sources are the only ways out"},
		{ChainBuildGenesis, true, "it must say where the genesis came from"},
		{ChainLaunchNodes, true, "it must walk at least one phase"},
		{ChainDeployNodes, false, "its two detail states say what the target was, and it may move on without either"},
		{ChainOpenWorkspace, false, "it has no states of its own"},
		{ChainBuildNodeConfig, false, "same"},
		{CompareChain, true, "the four verdicts are the only ways out"},
		{ReconcileChain, true, "kept and redone are the only ways out"},
	} {
		if got := Detailed(c.at); got != c.want {
			t.Errorf("Detailed(%s) = %v, want %v — %s", c.at, got, c.want, c.why)
		}
	}
}
