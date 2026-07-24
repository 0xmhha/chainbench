// Package testrun is the third pipeline phase (requirement #10): it runs the
// registered test cases (pkg/testkit) against a verified NodeSet, gating each
// case by chain compatibility and required capabilities, and produces a Report.
// It emits obs events and persists a RunRecord per case so the dashboard and
// `report` can read results back (docs/CHAINBENCH_GO_REDESIGN.md §3, §9).
package testrun

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// Options configures a test run.
type Options struct {
	// Names, if non-empty, restricts the run to cases with these names.
	Names []string
	// Categories, if non-empty, restricts the run to these categories.
	Categories []string
	// Factory builds RPC clients for cases; defaults to testkit.DefaultFactory.
	Factory testkit.ClientFactory
	// Bus receives per-case events (may be nil).
	Bus *obs.Bus
	// Store persists a RunRecord per case (may be nil).
	Store obs.Store
	// Now supplies timestamps for records; defaults to time.Now.
	Now func() time.Time
}

// Run executes the registered cases against ns and returns the aggregate
// Report. Cases not applicable to ns.Chain or lacking a required capability are
// skipped (recorded as StatusSkip). Per-case failures do not stop the run.
func Run(ctx context.Context, ns node.NodeSet, opts Options) (testkit.Report, error) {
	factory := opts.Factory
	if factory == nil {
		factory = testkit.DefaultFactory
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	var rep testkit.Report
	for _, c := range testkit.Cases() {
		if !nameSelected(c, opts) {
			continue
		}
		if reason := skipReason(c, ns); reason != "" {
			res := testkit.Result{Name: c.Name, Category: c.Category, Status: testkit.StatusSkip, Message: reason}
			rep.Results = append(rep.Results, res)
			record(opts, ns, res, now)
			emit(opts.Bus, ns, res)
			continue
		}

		res := testkit.RunCase(ctx, c, ns, factory)
		rep.Results = append(rep.Results, res)
		record(opts, ns, res, now)
		emit(opts.Bus, ns, res)
	}

	pass, fail, skip := rep.Counts()
	emit(opts.Bus, ns, testkit.Result{Name: "__summary__", Status: testkit.StatusPass,
		Message: fmt.Sprintf("pass=%d fail=%d skip=%d", pass, fail, skip)})
	return rep, nil
}

// nameSelected applies the Names/Categories filters.
func nameSelected(c testkit.Case, opts Options) bool {
	if len(opts.Names) > 0 && !contains(opts.Names, c.Name) {
		return false
	}
	if len(opts.Categories) > 0 && !contains(opts.Categories, c.Category) {
		return false
	}
	return true
}

// skipReason returns a non-empty reason if the case should be skipped for ns.
func skipReason(c testkit.Case, ns node.NodeSet) string {
	if !c.AppliesTo(ns.Chain) {
		return fmt.Sprintf("not compatible with chain %q (compat: %s)", ns.Chain, strings.Join(c.ChainCompat, ","))
	}
	for _, cap := range c.RequiresCaps {
		if !ns.HasCapability(cap) {
			return fmt.Sprintf("missing capability %q", cap)
		}
	}
	return ""
}

func record(opts Options, ns node.NodeSet, res testkit.Result, now func() time.Time) {
	if opts.Store == nil {
		return
	}
	status := obs.RunSucceeded
	switch res.Status {
	case testkit.StatusFail:
		status = obs.RunFailed
	case testkit.StatusSkip:
		status = obs.RunSucceeded // a skip is not a failure
	}
	ts := now()
	_ = opts.Store.SaveRun(obs.RunRecord{
		ID:        "test/" + res.Name,
		Phase:     obs.PhaseTest,
		Chain:     ns.Chain,
		Network:   ns.Network,
		Status:    status,
		StartedAt: ts,
		EndedAt:   ts.Add(res.Duration),
		Result:    map[string]any{"status": string(res.Status), "message": res.Message},
	})
}

func emit(bus *obs.Bus, ns node.NodeSet, res testkit.Result) {
	if bus == nil {
		return
	}
	kind := obs.KindResult
	if res.Status == testkit.StatusFail {
		kind = obs.KindError
	}
	bus.Publish(obs.Event{
		Phase: obs.PhaseTest, Kind: kind, Network: ns.Network,
		Message: res.Name,
		Fields:  map[string]any{"status": string(res.Status), "category": res.Category, "message": res.Message},
	})
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
