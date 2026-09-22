package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/lifecycle"

	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
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
// The observation reuses health.Run and the restart reuses verb.NetRestart
// — nodemonitor re-implements neither.

// healthObserver produces one round of nodemonitor.Facts for a composed network
// by reusing health.Run (rpc reachability, chain id, block advance, peers,
// sync). PIDAlive comes from the recorded pid; a crashed process reads as
// pid-recorded-but-rpc-down, which classifies RESTARTABLE all the same.
type healthObserver struct {
	nodes node.NodeSet
	fork  forkGate
}

func (o healthObserver) Observe(ctx context.Context) ([]nodemonitor.Facts, error) {
	rep, err := health.Run(ctx, o.nodes, health.Options{}, nil)
	if err != nil {
		return nil, err
	}
	return factsFromReport(rep, o.nodes, o.fork), nil
}

// forkGate is what the readiness gate has to know about a network that stops on
// purpose: where it stops, and which nodes hand over there.
//
// Two things put a network in that shape. One is a hardfork: the pre-fork build
// refuses to seal the fork block and the successors take over. The other is a
// genesis the build cannot execute past — one naming a system-contract version
// it does not have — where the chain seals up to that block and stays. The gate
// treats them alike because from the outside they are alike; only the second
// has no successor, which is a preFork set with nothing in it.
//
// Such a network passes through two states no other network has, and in both of
// them the ordinary question — is every node keeping up? — has the wrong answer.
// Before the handover nothing advances, because the pre-fork build refuses to
// seal the fork block. After it the pre-fork nodes stop for good, because they
// cannot validate what the successors produce.
//
// The zero value is a network that crosses no fork, and every check below is
// then inert.
type forkGate struct {
	// at is the block the network stops one short of: the fork's activation
	// block, or the block a halting genesis cannot commit.
	at int64
	// preFork are the node indices running the build that hands over.
	preFork map[int]bool
	// restart says the whole network crosses together rather than handing
	// production from one set of nodes to another.
	//
	// Neither of the states below happens on a restart, and reading them there
	// is worse than useless. An ordinary fork does not halt the chain, so a
	// restarted network is never parked — and calling it parked let the gate
	// pass at height 0, before a single block was sealed. Nor does a restart
	// leave anyone behind: every node moves, so none of them is retired, and
	// treating them all as retired would report a network where nothing runs
	// as ready.
	restart bool
}

// parked reports whether every node stands at the block before the fork,
// waiting to be handed over. Every node, not the highest: one successor still
// syncing is a network that has not arrived, and it is exactly the node the
// handover would leave behind.
func (g forkGate) parked(rep health.Report) bool {
	if g.restart || g.at <= 0 || len(rep.Nodes) == 0 {
		return false
	}
	want := uint64(g.at - 1) //nolint:gosec // a declared block height, checked positive above
	for _, n := range rep.Nodes {
		if !n.OK || n.BlockNumber != want {
			return false
		}
	}
	return true
}

// retired reports whether one node has finished its part: it runs the pre-fork
// build and the network has produced the fork block, so it will not move again.
func (g forkGate) retired(index int, networkHead uint64) bool {
	if g.restart || g.at <= 0 || !g.preFork[index] {
		return false
	}
	return networkHead >= uint64(g.at) //nolint:gosec // a declared block height, checked positive above
}

// factsFromReport maps a health report to per-node facts, taking PIDAlive from
// each node's recorded pid in ns (a node with no recorded pid is not alive).
// Pure, so the mapping is unit-tested without a live network.
//
// It fills the Want* fields, which is what makes the classifier's conditions
// live. Before this it set only Wanted, so three of them never fired: the
// wrong-genesis check (WantChainID zero), the peering check (WantPeers zero) and
// the participation check. The conditions were written, tested in the classifier,
// and unreachable from the run path — the same shape as a comparison whose want
// side is never supplied.
//
// Two are still left unset, each for a reason worth writing down rather than
// filling badly.
//
// WantParticipate: Participating comes from the collector and this observer has
// no source for it, so requiring it would hold every network until the wait
// budget ran out.
//
// WantChainID: the classifier treats a mismatch as FATAL, so the wanted value
// has to be authoritative. Taking it from the observation itself is circular —
// if every node shares the wrong genesis it agrees with itself — and the
// composed value is not reachable here: the workspace records the chain's NAME,
// and the request's numeric id is zero whenever the chain's own manifest supplies
// it, which is the usual case. Filling it from the first node that answers would
// make a fatal verdict depend on iteration order. It stays unset until there is
// a real source, and the genesis a composition was built from is compared by
// preflight instead.
func factsFromReport(rep health.Report, ns node.NodeSet, fork forkGate) []nodemonitor.Facts {
	pid := make(map[int]int, len(ns.Nodes))
	for _, n := range ns.Nodes {
		pid[n.Index] = n.PID
	}
	// The network's head as this round saw it, so the classifier can tell a node
	// that has caught up from one that is merely not syncing. health.Report has
	// no such field: Producing is one fact about the whole network, which is
	// exactly why a node sitting at height 0 read as ready while the other three
	// produced.
	var head uint64
	for _, ni := range rep.Nodes {
		if ni.BlockNumber > head {
			head = ni.BlockNumber
		}
	}
	// A node in a multi-node network with no peer cannot receive a block or take
	// its turn, and nothing else in the facts says so: it is not syncing (it has
	// no one to sync from), and "advancing" is a fact about the network, which
	// the others make true. Measured: one validator of four sat at height 0
	// while the other three produced, every turn the round robin gave it was
	// lost to a round-change timeout, and it only joined after catching up ~30
	// blocks in one import.
	//
	// The floor is one peer, not n-1. A mesh gives every node n-1, but a proxied
	// topology deliberately gives a bp only its pn, so n-1 would refuse a
	// network that is exactly as connected as it declared. One peer is the line
	// between connected and alone, which is the failure this saw.
	wantPeers := 0
	if len(ns.Nodes) > 1 {
		wantPeers = 1
	}

	// A confirmed split is a fact about the whole set, so it lands on every
	// node's facts the way Advancing does. health already answers it — the
	// agreement check compares hashes at the highest block EVERY answering node
	// has, which is why a lagging node cannot be mistaken for a fork — and the
	// classifier already has the condition. Nothing carried the answer from one
	// to the other, so the FATAL "chain diverged" verdict could not fire from the
	// run path: a split network gated as ready, which is the same blind spot
	// `verify` had before the agreement check existed (it reported a forked
	// network healthy because the primary node's height was rising).
	//
	// Only a CHECKED disagreement counts. Unchecked means no comparison was
	// possible — every node at a different head during bring-up, say — and
	// reading that as a fork would terminate a healthy run.
	forked := rep.Agreement.Checked && !rep.Agreement.Agreed
	// A network composed to cross a hardfork stands one block short of it until
	// it is handed over: the pre-fork build refuses to seal the fork block and
	// the successors do not produce until they are told to. The height is not
	// advancing, and reporting only that would have the gate wait out its whole
	// budget and call a correct network unfit.
	//
	// So it is reported as what it is. Every node has to be there, not the
	// highest: one successor still syncing is a network that has not arrived,
	// and it is exactly the node the handover would leave behind.
	advancing := rep.Producing || fork.parked(rep)
	facts := make([]nodemonitor.Facts, 0, len(rep.Nodes))
	for _, ni := range rep.Nodes {
		facts = append(facts, nodemonitor.Facts{
			Node:       ni.Index,
			Label:      "node" + strconv.Itoa(ni.Index),
			Wanted:     true,
			Retired:    fork.retired(ni.Index, head),
			PIDAlive:   pid[ni.Index] > 0,
			RPCUp:      ni.OK,
			ChainID:    ni.ChainID,
			Height:     ni.BlockNumber,
			Advancing:  advancing,
			WantHeight: head,
			WantPeers:  wantPeers,
			Forked:     forked,
			Syncing:    ni.Syncing,
			Peers:      clampCount(ni.PeerCount),
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
	_, err := verb.NetRestart(ctx, r.deps, verb.NetRestartIn{DataDir: r.dataDir, Node: n})
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
func gateReady(ctx context.Context, deps chainsetup.Deps, dataDir string, nodes *node.NodeSet, steps *[]string, waitBudget time.Duration, fork forkGate) error {
	if nodes == nil || len(nodes.Nodes) == 0 {
		return nil
	}
	res, err := nodemonitor.Gate(ctx,
		healthObserver{nodes: *nodes, fork: fork},
		restartAdapter{deps: deps, dataDir: dataDir},
		ctxClock{},
		stepSink{steps: steps},
		// A zero budget takes nodemonitor's default; a caller raises it for a
		// large or slow bring-up (a 15-node poa network over docker) where late
		// endpoints take longer than the default to sync.
		nodemonitor.Options{MaxNodeMonitorTimeout: waitBudget},
	)
	if err != nil {
		return fmt.Errorf("nodemonitor: %w", err)
	}
	if !res.OK {
		return lifecycle.Mark(errPrepareNotReady, fmt.Errorf("network not ready to test: %s", res.Terminate))
	}
	return nil
}
