package testkit

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/node"
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

// RunCase executes one case against ns and returns its Result. It recovers both
// the Fatalf sentinel and any unexpected panic in the case body, so a buggy
// case fails rather than crashing the runner.
func RunCase(ctx context.Context, c Case, ns node.NodeSet, f ClientFactory) Result {
	t := newT(ctx, ns, f)
	start := time.Now()
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(failSentinel); ok {
					return // failure already recorded in t.msgs
				}
				t.failed = true
				t.msgs = append(t.msgs, fmt.Sprintf("panic: %v", r))
			}
		}()
		c.Fn(t)
	}()
	res := Result{Name: c.Name, Category: c.Category, Duration: time.Since(start), Message: t.message()}
	if t.failed {
		res.Status = StatusFail
	} else {
		res.Status = StatusPass
	}
	return res
}
