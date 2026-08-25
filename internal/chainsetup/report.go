package chainsetup

import (
	"fmt"
	"time"
)

// Outcome is how one executed step ended.
type Outcome string

const (
	// OK means the step did what it says.
	OK Outcome = "OK"
	// Failed means the step ran and did not succeed.
	Failed Outcome = "FAIL"
	// Skipped means the step was not reached (an earlier one failed, or the run
	// stopped before it).
	Skipped Outcome = "SKIP"
	// NotImplemented means the framework has no code for this step. It is kept
	// distinct from Failed: "we never built this" and "we built it and it broke"
	// lead to different work.
	NotImplemented Outcome = "TODO"
)

// Result records one executed step.
type Result struct {
	Step     CaseStep
	Outcome  Outcome
	Detail   string
	Duration time.Duration
}

// Reporter receives each step's result as it completes, so a long bring-up
// reports progress rather than going quiet.
type Reporter func(Result)

// Run is the outcome of executing a case.
type Run struct {
	Case    Case
	Results []Result
	// Nodes are the launched node RPC endpoints, when the run got that far.
	Nodes []string
	// DataDir is where the network's files live.
	DataDir string
}

// Failed reports whether any step failed or was missing.
func (r Run) Failed() bool {
	for _, res := range r.Results {
		if res.Outcome == Failed || res.Outcome == NotImplemented {
			return true
		}
	}
	return false
}

// FirstProblem returns the first step that failed or was not implemented.
func (r Run) FirstProblem() (Result, bool) {
	for _, res := range r.Results {
		if res.Outcome == Failed || res.Outcome == NotImplemented {
			return res, true
		}
	}
	return Result{}, false
}

// stepRunner executes one step and returns a detail line describing what it
// observed. Returning an error fails the step and stops the run.
type stepRunner func() (detail string, err error)

// tracker executes steps in order, reporting each, and stops at the first
// failure. A step with no runner is reported as not implemented.
type tracker struct {
	report  Reporter
	results []Result
	stopped bool
	// stopAfter, when non-empty, ends the run once that step completes.
	stopAfter string
}

func newTracker(report Reporter, stopAfter string) *tracker {
	if report == nil {
		report = func(Result) {}
	}
	return &tracker{report: report, stopAfter: stopAfter}
}

// do runs one step unless the tracker has already stopped.
func (t *tracker) do(step CaseStep, run stepRunner) {
	if t.stopped {
		t.add(Result{Step: step, Outcome: Skipped})
		return
	}
	if !step.Implemented || run == nil {
		t.add(Result{Step: step, Outcome: NotImplemented,
			Detail: "no implementation: this step is modelled but not built"})
		t.stopped = true
		return
	}
	start := time.Now()
	detail, err := run()
	elapsed := time.Since(start)
	if err != nil {
		t.add(Result{Step: step, Outcome: Failed, Detail: err.Error(), Duration: elapsed})
		t.stopped = true
		return
	}
	t.add(Result{Step: step, Outcome: OK, Detail: detail, Duration: elapsed})
	if t.stopAfter != "" && step.ID == t.stopAfter {
		t.stopped = true
	}
}

func (t *tracker) add(r Result) {
	t.results = append(t.results, r)
	t.report(r)
}

// halted reports whether the run has stopped (failed, or reached --stop-after).
func (t *tracker) halted() bool { return t.stopped }

// validateStopAfter checks that a --stop-after value names a step of the case.
func validateStopAfter(c Case, stopAfter string) error {
	if stopAfter == "" {
		return nil
	}
	if c.StepIndex(stopAfter) < 0 {
		return fmt.Errorf("chainsetup: case %q has no step %q", c.ID, stopAfter)
	}
	return nil
}
