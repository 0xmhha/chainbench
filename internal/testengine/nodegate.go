package testengine

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

// The run path gates a composed network through nodemonitor before running any
// test on it (E6): a network that is up but not yet producing, or missing a
// node, is waited on or restarted within limits rather than run against blind.
// The observation reuses health.Run and the restart reuses chainsetup.NetRestart
// — nodemonitor re-implements neither.

// healthObserver produces one round of nodemonitor.Facts for a composed network
// by reusing health.Run (rpc reachability, chain id, block advance, peers,
// sync). PIDAlive comes from the recorded pid; a crashed process reads as
// pid-recorded-but-rpc-down, which classifies RESTARTABLE all the same.
type healthObserver struct {
	nodes node.NodeSet
}

func (o healthObserver) Observe(ctx context.Context) ([]nodemonitor.Facts, error) {
	rep, err := health.Run(ctx, o.nodes, health.Options{}, nil)
	if err != nil {
		return nil, err
	}
	return factsFromReport(rep, o.nodes), nil
}

// factsFromReport maps a health report to per-node facts, taking PIDAlive from
// each node's recorded pid in ns (a node with no recorded pid is not alive).
// Pure, so the mapping is unit-tested without a live network.
func factsFromReport(rep health.Report, ns node.NodeSet) []nodemonitor.Facts {
	pid := make(map[int]int, len(ns.Nodes))
	for _, n := range ns.Nodes {
		pid[n.Index] = n.PID
	}
	facts := make([]nodemonitor.Facts, 0, len(rep.Nodes))
	for _, ni := range rep.Nodes {
		facts = append(facts, nodemonitor.Facts{
			Node:      ni.Index,
			Label:     "node" + strconv.Itoa(ni.Index),
			Wanted:    true,
			PIDAlive:  pid[ni.Index] > 0,
			RPCUp:     ni.OK,
			ChainID:   ni.ChainID,
			Height:    ni.BlockNumber,
			Advancing: rep.Producing,
			Syncing:   ni.Syncing,
			Peers:     clampCount(ni.PeerCount),
		})
	}
	return facts
}

// clampCount narrows a uint64 count to int without overflow (gosec G115); a peer
// count never approaches the limit, but the conversion is made safe regardless.
func clampCount(v uint64) int {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int(v)
}

// restartAdapter restarts one node through the existing workspace restart verb.
type restartAdapter struct {
	deps    chainsetup.Deps
	dataDir string
}

func (r restartAdapter) Restart(ctx context.Context, n int) error {
	_, err := chainsetup.NetRestart(ctx, r.deps, chainsetup.NetRestartIn{DataDir: r.dataDir, Node: n})
	return err
}

// ctxClock waits d unless the context ends first.
type ctxClock struct{}

func (ctxClock) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// stepSink records every verdict and recovery attempt into the run's setup
// steps, so a run that waited or restarted before testing says so (E6: 판정 증적).
type stepSink struct{ steps *[]string }

func (s stepSink) Verdict(round int, r nodemonitor.NodeReport) {
	*s.steps = append(*s.steps, fmt.Sprintf("nodemonitor r%d node%d %s: %s", round, r.Node, r.Verdict, joinReasons(r.Reasons)))
}

func (s stepSink) Recovery(n, attempt int, err error) {
	outcome := "ok"
	if err != nil {
		outcome = err.Error()
	}
	*s.steps = append(*s.steps, fmt.Sprintf("nodemonitor restart node%d attempt%d: %s", n, attempt, outcome))
}

func joinReasons(rs []string) string {
	if len(rs) == 0 {
		return "-"
	}
	out := rs[0]
	for _, r := range rs[1:] {
		out += "; " + r
	}
	return out
}

// gateReady runs the readiness gate over a composed workspace network, waiting
// on WAITABLE nodes and restarting RESTARTABLE ones within limits. It returns an
// error when the network cannot be made fit (a FATAL node, or a budget/cap
// reached) so the caller does not run tests against it. A nil or empty node set
// (an attach to bare URLs) is not gated here.
func gateReady(ctx context.Context, deps chainsetup.Deps, dataDir string, nodes *node.NodeSet, steps *[]string) error {
	if nodes == nil || len(nodes.Nodes) == 0 {
		return nil
	}
	res, err := nodemonitor.Gate(ctx,
		healthObserver{nodes: *nodes},
		restartAdapter{deps: deps, dataDir: dataDir},
		ctxClock{},
		stepSink{steps: steps},
		nodemonitor.Options{},
	)
	if err != nil {
		return fmt.Errorf("nodemonitor: %w", err)
	}
	if !res.OK {
		return fmt.Errorf("network not ready to test: %s", res.Terminate)
	}
	return nil
}
