package testkit

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Status is the outcome of one case.
type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Result is one case's outcome.
type Result struct {
	Name     string        `json:"name"`
	Category string        `json:"category"`
	Status   Status        `json:"status"`
	Duration time.Duration `json:"duration"`
	Message  string        `json:"message,omitempty"`
}

// Report aggregates case results.
type Report struct {
	Results []Result `json:"results"`
	// Applicable is the number of selected cases compatible with the run's chain
	// (the coverage denominator). Cases gated out by chain incompatibility are
	// excluded; capability-gated skips are still applicable. Zero means the
	// producer did not populate it (coverage then reports 100).
	Applicable int `json:"applicable,omitempty"`
}

// Counts returns the number of passed, failed, and skipped results.
func (r Report) Counts() (pass, fail, skip int) {
	for _, res := range r.Results {
		switch res.Status {
		case StatusPass:
			pass++
		case StatusFail:
			fail++
		case StatusSkip:
			skip++
		}
	}
	return
}

// Failed reports whether any result failed.
func (r Report) Failed() bool {
	_, fail, _ := r.Counts()
	return fail > 0
}

// Coverage is the percentage of applicable cases that actually ran (passed or
// failed) — the signal that "0 failures" does not imply "well tested": a chain
// that gates most cases out has low coverage even with no failures. Returns 100
// when Applicable is unset (0), i.e. coverage is unknown.
func (r Report) Coverage() int {
	if r.Applicable <= 0 {
		return 100
	}
	pass, fail, _ := r.Counts()
	ran := pass + fail
	return ran * 100 / r.Applicable
}

// RunOpts carries per-run inputs a case may read but that must not live on the
// (serialized) NodeSet — currently the funded-account key for chain-agnostic
// write cases.
type RunOpts struct {
	FundedKey []byte
}

// RunCaseWith executes one case against ns with explicit run options and returns
// its Result. It recovers the Fatalf/Skip sentinels and any unexpected panic in
// the case body, so a buggy case fails rather than crashing the runner.
func RunCaseWith(ctx context.Context, c Case, ns node.NodeSet, f ClientFactory, opts RunOpts) Result {
	t := newT(ctx, ns, f, opts.FundedKey)
	start := time.Now()
	skipped := ""
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(failSentinel); ok {
					return // failure already recorded in t.msgs
				}
				if s, ok := r.(skipSentinel); ok {
					skipped = s.msg
					return
				}
				t.failed = true
				t.msgs = append(t.msgs, fmt.Sprintf("panic: %v", r))
			}
		}()
		c.Fn(t)
	}()
	res := Result{Name: c.Name, Category: c.Category, Duration: time.Since(start), Message: t.message()}
	switch {
	case skipped != "":
		res.Status = StatusSkip
		res.Message = skipped
	case t.failed:
		res.Status = StatusFail
	default:
		res.Status = StatusPass
	}
	return res
}
