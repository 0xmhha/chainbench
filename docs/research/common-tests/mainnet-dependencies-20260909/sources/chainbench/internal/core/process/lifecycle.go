package process

import (
	"context"
	"fmt"
	"sync"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Node-process control over an already-launched NodeSet. It lives here — next to
// Driver and NodeSpec, the contract it drives — so the surfaces that tear a
// running network down (a handoff's teardown, live-test fixtures) do not depend
// on the legacy pipeline package that used to own it.
//
// It operates on the PIDs a NodeSet records, so it only reaches nodes chainbench
// launched; an attached node (PID 0) is not ours to stop.

// StopNodeSet stops every launched node (PID > 0) in ns through d. It is
// best-effort: a node that fails to stop is collected in errs and the rest still
// stop, so one dead PID does not block tearing down the network. It returns how
// many were stopped and the per-node errors.
//
// The nodes are stopped concurrently, because each stop now waits for its node
// to close its database (see stop.go). Done in turn, a network whose nodes all
// ignored the signal would take the grace period once per node — six nodes at
// fifteen seconds is a minute and a half of teardown. Asking them all at once
// and waiting together costs the grace once.
func StopNodeSet(ctx context.Context, d Driver, ns node.NodeSet) (int, []error) {
	type result struct {
		index int
		pid   int
		err   error
	}
	var wg sync.WaitGroup
	results := make(chan result, len(ns.Nodes))
	for _, n := range ns.Nodes {
		if n.PID <= 0 {
			continue
		}
		wg.Add(1)
		go func(n node.Node) {
			defer wg.Done()
			results <- result{n.Index, n.PID, d.Stop(ctx, Handle{Index: n.Index, PID: n.PID})}
		}(n)
	}
	wg.Wait()
	close(results)

	// Collected in node order rather than completion order: which node failed
	// to stop is what a reader needs, and a list that reorders itself between
	// runs is harder to compare.
	byIndex := map[int]result{}
	for r := range results {
		byIndex[r.index] = r
	}
	stopped := 0
	var errs []error
	for _, n := range ns.Nodes {
		r, ok := byIndex[n.Index]
		if !ok {
			continue
		}
		if r.err != nil {
			errs = append(errs, fmt.Errorf("node%d (pid %d): %w", r.index, r.pid, r.err))
			continue
		}
		stopped++
	}
	return stopped, errs
}
