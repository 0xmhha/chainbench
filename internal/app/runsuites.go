package app

import (
	"context"
	"fmt"
)

// Running several test definitions in one command.
//
// A definition file is one run. Several of them are that same run repeated, in
// the order given — not a new execution mode, so a single definition and a list
// of them follow one rule and there is nothing extra to reason about. The DSL
// stays the language of one definition and the engine stays its runtime; the
// sequencing is orchestration, and lives here.
//
// What differs between one and many is only what happens between definitions:
// the network is left running, so the next definition's own preflight decides
// whether to reuse it or compose a new one. Composing a chain is the expensive
// step, so consecutive definitions that want the same chain pay for it once.
// The order is the caller's and is never rearranged — a suite can depend on it.

// SuiteRunResult is one definition's outcome within a sequential run.
type SuiteRunResult struct {
	// Spec is the definition this result is for (its path, or its index when
	// the specs came inline).
	Spec string `json:"spec"`
	// Out is the run's report; its SessionRoot is that definition's session.
	Out RunSuiteOut `json:"out"`
	// Err is the run's error message, empty when it ran.
	Err string `json:"error,omitempty"`
}

// RunSuitesOut collects the sequential run's per-definition results.
type RunSuitesOut struct {
	// Runs are the per-definition results, in the order they ran.
	Runs []SuiteRunResult `json:"runs"`
}

// Totals adds up what the run produced across every definition: how many could
// not run at all, how many tests failed, and how many were blocked.
//
// The counting lives here rather than in a surface so the CLI's exit code and
// any other reader agree on what happened — the three are not interchangeable,
// and a surface that recounts them is a second opinion waiting to drift.
func (o RunSuitesOut) Totals() (setupErrors, failed, blocked int) {
	for _, r := range o.Runs {
		if r.Err != "" {
			setupErrors++
			continue
		}
		failed += r.Out.Summary.Summary.Fail
		blocked += r.Out.Summary.Summary.Blocked
	}
	return setupErrors, failed, blocked
}

// Failed reports whether anything went wrong, for a caller that needs only the
// verdict and not the breakdown.
func (o RunSuitesOut) Failed() bool {
	setupErrors, failed, blocked := o.Totals()
	return setupErrors > 0 || failed > 0 || blocked > 0
}

// RunSuites runs each definition in turn through the same RunSuite a single
// definition uses, keeping the network up between them so the next definition's
// preflight can reuse it. The last definition tears the network down unless the
// caller asked to keep it.
//
// A definition that fails does not stop the rest: the remaining ones still run
// and report, which is what a suite of independent definitions wants. The
// network is left as that definition left it, and the next definition's
// preflight decides what to do with it.
func RunSuites(ctx context.Context, d Deps, in RunSuiteIn) (RunSuitesOut, error) {
	specs, err := suiteSpecUnits(in)
	if err != nil {
		return RunSuitesOut{}, err
	}
	var out RunSuitesOut
	for i, unit := range specs {
		one := in
		one.SpecPaths, one.SpecContent = unit.paths, unit.content
		// Every definition but the last leaves the chain up for the next one to
		// judge; the last honours what the caller asked for.
		one.KeepUp = in.KeepUp || i < len(specs)-1

		res, runErr := RunSuite(ctx, d, one)
		entry := SuiteRunResult{Spec: unit.label, Out: res}
		if runErr != nil {
			entry.Err = runErr.Error()
		}
		out.Runs = append(out.Runs, entry)
	}
	return out, nil
}

// specUnit is one definition to run: either a file path or inline content.
type specUnit struct {
	label   string
	paths   []string
	content [][]byte
}

// suiteSpecUnits splits a request into one unit per definition, preserving the
// caller's order. Inline content (the MCP form) is split the same way, so both
// surfaces sequence identically.
func suiteSpecUnits(in RunSuiteIn) ([]specUnit, error) {
	if len(in.SpecContent) > 0 {
		units := make([]specUnit, 0, len(in.SpecContent))
		for i, c := range in.SpecContent {
			units = append(units, specUnit{label: fmt.Sprintf("spec %d", i+1), content: [][]byte{c}})
		}
		return units, nil
	}
	if len(in.SpecPaths) == 0 {
		return nil, fmt.Errorf("run: no test definition given")
	}
	units := make([]specUnit, 0, len(in.SpecPaths))
	for _, p := range in.SpecPaths {
		units = append(units, specUnit{label: p, paths: []string{p}})
	}
	return units, nil
}
