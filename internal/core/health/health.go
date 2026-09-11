// Package health answers "is this network producing, and what state is each
// node in" (requirement #9): given a NodeSet, it samples every node over RPC
// and returns a report — chain id, height, peers, sync state, plus the
// producing verdict from the primary node's height advancing.
//
// This is the INSPECTION read, used by `chainbench verify` and its MCP mirror.
// It is deliberately not the same code as engine's launch gate
// (NewBlockAdvanceGate), which polls one node to a target height as a
// launcher precondition: one produces a report about every node, the other
// blocks a bring-up on one node, and merging them would bend both.
//
// It was core/pipeline/verify; the move out of the pipeline tree is part of
// the legacy-pipeline retirement — the code is sound, only its home
// was legacy.
package health

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Prober is the RPC surface verify needs from a node. *rpc.Client satisfies it;
// tests inject a fake.
type Prober interface {
	ChainID(ctx context.Context) (uint64, error)
	BlockNumber(ctx context.Context) (uint64, error)
	PeerCount(ctx context.Context) (uint64, error)
	Syncing(ctx context.Context) (bool, error)
}

// Options configures a verify run.
type Options struct {
	// Dial builds a Prober for a node's RPC URL. Defaults to rpc.Dial.
	Dial func(url string) Prober
	// ProgressDelay is the wait between the two block-height samples used to
	// detect production. Defaults to 2s.
	ProgressDelay time.Duration
	// ReadyTimeout bounds how long detection waits for the height to start
	// advancing. When >0, verify re-samples every ProgressDelay until the
	// height rises above its baseline or this window elapses — a just-launched
	// wbft network needs ~20-30s for static-node peering to form the quorum
	// mesh before it produces its first block. When <=0 (the default), verify
	// takes a single two-sample reading (legacy behavior).
	ReadyTimeout time.Duration
	// Sleep waits for d; injectable so tests don't spend real time. Defaults
	// to time.Sleep.
	Sleep func(d time.Duration)
}

// BlockHasher reads the hash of a block at a height. It is a separate interface
// from Prober for the same reason Sealer is: a Prober that cannot answer it is
// not broken, and the check then reports that it could not tell rather than
// guessing. *rpc.Client satisfies it.
type BlockHasher interface {
	BlockHashAt(ctx context.Context, n uint64) (string, error)
}

// Agreement is whether the nodes are on the same chain.
//
// It is a struct rather than a bool because "not checked" and "agreed" are
// different answers and collapsing them is the failure this exists to prevent.
// A single node, or a Prober that cannot read a hash, yields Checked=false with
// a reason — never Agreed=true by default.
type Agreement struct {
	// Checked is whether the comparison could be made at all.
	Checked bool `json:"checked"`
	// Agreed is whether every node reported the same hash at Height. It is
	// meaningless unless Checked.
	Agreed bool `json:"agreed"`
	// Height is the block compared: the highest block every answering node has,
	// because nodes at different heads legitimately hold different head hashes
	// and comparing those would call a lagging node a fork.
	Height uint64 `json:"height,omitempty"`
	// Detail names the disagreement, or says why no comparison was made.
	Detail string `json:"detail,omitempty"`
}

// NodeInfo is the verified state of one node.
type NodeInfo struct {
	Index       int    `json:"index"`
	RPCURL      string `json:"rpc_url"`
	ChainID     uint64 `json:"chain_id"`
	BlockNumber uint64 `json:"block_number"`
	PeerCount   uint64 `json:"peer_count"`
	Syncing     bool   `json:"syncing"`
	OK          bool   `json:"ok"`
	Err         string `json:"err,omitempty"`
}

// Report is the outcome of a verify run.
type Report struct {
	Network   string `json:"network"`
	Producing bool   `json:"producing"`
	// Agreement is whether the nodes are on one chain. Producing alone does not
	// say: it reads the PRIMARY node's height rising, which is equally true of a
	// network that has split, where every partition keeps producing on its own
	// fork. A verify that answered "healthy" for that was answering a question
	// nobody asked.
	Agreement Agreement  `json:"agreement"`
	Nodes     []NodeInfo `json:"nodes"`
}

// Run verifies every node in ns. Producing is true when the primary node's
// block height strictly increases across the two samples. The returned error is
// non-nil only for a wholly empty node set; per-node failures are recorded in
// NodeInfo.OK/Err. bus may be nil.
func Run(ctx context.Context, ns node.NodeSet, opts Options, bus *collector.Bus) (Report, error) {
	dial := opts.Dial
	if dial == nil {
		dial = func(url string) Prober { return rpc.Dial(url) }
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	delay := opts.ProgressDelay
	if delay == 0 {
		delay = 2 * time.Second
	}

	rep := Report{Network: ns.Network}

	// Determine production first: with a readiness window this polls through
	// the RPC-not-up-yet and peering-not-converged phases, so the per-node
	// snapshot below reflects the settled network rather than the moment right
	// after launch.
	rep.Producing = detectProducing(ctx, ns, dial, sleep, delay, opts.ReadyTimeout)

	for _, n := range ns.Nodes {
		info := NodeInfo{Index: n.Index, RPCURL: n.RPCURL}
		p := dial(n.RPCURL)
		if err := fill(ctx, p, &info); err != nil {
			info.OK = false
			info.Err = err.Error()
			emit(bus, collector.Event{Phase: collector.PhaseVerify, Kind: collector.KindError, Network: ns.Network,
				Node: n.Index, Message: "node info failed", Fields: map[string]any{"error": err.Error()}})
		} else {
			info.OK = true
			emit(bus, collector.Event{Phase: collector.PhaseVerify, Kind: collector.KindProgress, Network: ns.Network,
				Node: n.Index, Message: "node info", Fields: map[string]any{
					"chain_id": info.ChainID, "block": info.BlockNumber, "peers": info.PeerCount}})
		}
		rep.Nodes = append(rep.Nodes, info)
	}

	rep.Agreement = checkAgreement(ctx, ns, rep.Nodes, dial)
	if rep.Agreement.Checked && !rep.Agreement.Agreed {
		emit(bus, collector.Event{Phase: collector.PhaseVerify, Kind: collector.KindError, Network: ns.Network,
			Message: "nodes disagree", Fields: map[string]any{"detail": rep.Agreement.Detail, "height": rep.Agreement.Height}})
	}

	emit(bus, collector.Event{Phase: collector.PhaseVerify, Kind: collector.KindResult, Network: ns.Network,
		Message: "verify complete", Fields: map[string]any{
			"producing": rep.Producing, "nodes": len(rep.Nodes),
			"agreed": rep.Agreement.Agreed, "agreement_checked": rep.Agreement.Checked}})
	return rep, nil
}

// checkAgreement asks every answering node for its hash at the highest block ALL
// of them have, and reports whether they match.
//
// The height matters. Nodes at different heads legitimately hold different head
// hashes, so comparing heads would report a node that is one block behind as a
// fork. The common height is the lowest head among the answering nodes: the last
// block every one of them has an opinion about.
func checkAgreement(ctx context.Context, ns node.NodeSet, infos []NodeInfo, dial func(string) Prober) Agreement {
	type answer struct {
		index int
		url   string
	}
	var live []answer
	common := ^uint64(0)
	for i, info := range infos {
		if !info.OK {
			continue
		}
		live = append(live, answer{index: info.Index, url: ns.Nodes[i].RPCURL})
		if info.BlockNumber < common {
			common = info.BlockNumber
		}
	}
	if len(live) < 2 {
		return Agreement{Detail: fmt.Sprintf("only %d node(s) answered; agreement needs at least two", len(live))}
	}
	if common == 0 {
		return Agreement{Detail: "the lowest node is still at genesis; nothing produced to compare yet"}
	}

	byHash := map[string][]int{}
	for _, a := range live {
		h, ok := dial(a.url).(BlockHasher)
		if !ok {
			return Agreement{Detail: "this prober cannot read a block hash, so agreement was not checked"}
		}
		hash, err := h.BlockHashAt(ctx, common)
		if err != nil || hash == "" {
			return Agreement{Detail: fmt.Sprintf("node%d could not report its hash at block %d, so agreement was not checked", a.index, common)}
		}
		byHash[hash] = append(byHash[hash], a.index)
	}
	if len(byHash) == 1 {
		return Agreement{Checked: true, Agreed: true, Height: common,
			Detail: fmt.Sprintf("all %d node(s) report the same hash at block %d", len(live), common)}
	}

	parts := make([]string, 0, len(byHash))
	for hash, nodes := range byHash {
		sort.Ints(nodes)
		parts = append(parts, fmt.Sprintf("%s: %v", short(hash), nodes))
	}
	sort.Strings(parts)
	return Agreement{Checked: true, Agreed: false, Height: common,
		Detail: fmt.Sprintf("block %d has %d different hashes — %s", common, len(byHash), strings.Join(parts, "; "))}
}

// short trims a hash for a message that has to name several.
func short(hash string) string {
	if len(hash) > 12 {
		return hash[:12] + "…"
	}
	return hash
}

func fill(ctx context.Context, p Prober, info *NodeInfo) error {
	var err error
	if info.ChainID, err = p.ChainID(ctx); err != nil {
		return err
	}
	if info.BlockNumber, err = p.BlockNumber(ctx); err != nil {
		return err
	}
	if info.PeerCount, err = p.PeerCount(ctx); err != nil {
		return err
	}
	if info.Syncing, err = p.Syncing(ctx); err != nil {
		return err
	}
	return nil
}

// detectProducing reports whether the primary node's height advances. With
// timeout<=0 it takes a single two-sample reading (delay apart). With
// timeout>0 it captures a baseline then re-samples every delay until the
// height rises above the baseline or the cumulative wait reaches timeout —
// giving a freshly launched network time to peer and start sealing.
func detectProducing(ctx context.Context, ns node.NodeSet, dial func(string) Prober, sleep func(time.Duration), delay, timeout time.Duration) bool {
	primary, ok := ns.Primary()
	if !ok {
		return false
	}
	p := dial(primary.RPCURL)

	if timeout <= 0 {
		baseline, err := p.BlockNumber(ctx)
		if err != nil {
			return false
		}
		sleep(delay)
		second, err := p.BlockNumber(ctx)
		if err != nil {
			return false
		}
		return second > baseline
	}

	// Poll through transient errors (RPC still starting) until the height
	// advances beyond the first successful reading, or the window elapses.
	var baseline uint64
	haveBaseline := false
	for waited := time.Duration(0); waited < timeout; waited += delay {
		if ctx.Err() != nil {
			return false
		}
		if cur, err := p.BlockNumber(ctx); err == nil {
			if !haveBaseline {
				baseline, haveBaseline = cur, true
			} else if cur > baseline {
				return true
			}
		}
		sleep(delay)
	}
	return false
}

func emit(bus *collector.Bus, e collector.Event) {
	if bus != nil {
		bus.Publish(e)
	}
}
