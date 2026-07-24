// Package verify is the second pipeline phase (requirement #9): given a NodeSet
// (from setup or attach), it confirms over RPC that the chain is producing
// blocks and gathers per-node info (chain id, height, peers, sync state). It is
// the gate before the test phase and the entry point for attached networks
// (docs/CHAINBENCH_GO_REDESIGN.md §3).
package verify

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
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
	// Sleep waits for d; injectable so tests don't spend real time. Defaults
	// to time.Sleep.
	Sleep func(d time.Duration)
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
	Network   string     `json:"network"`
	Producing bool       `json:"producing"`
	Nodes     []NodeInfo `json:"nodes"`
}

// Run verifies every node in ns. Producing is true when the primary node's
// block height strictly increases across the two samples. The returned error is
// non-nil only for a wholly empty node set; per-node failures are recorded in
// NodeInfo.OK/Err. bus may be nil.
func Run(ctx context.Context, ns node.NodeSet, opts Options, bus *obs.Bus) (Report, error) {
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

	for _, n := range ns.Nodes {
		info := NodeInfo{Index: n.Index, RPCURL: n.RPCURL}
		p := dial(n.RPCURL)
		if err := fill(ctx, p, &info); err != nil {
			info.OK = false
			info.Err = err.Error()
			emit(bus, obs.Event{Phase: obs.PhaseVerify, Kind: obs.KindError, Network: ns.Network,
				Node: n.Index, Message: "node info failed", Fields: map[string]any{"error": err.Error()}})
		} else {
			info.OK = true
			emit(bus, obs.Event{Phase: obs.PhaseVerify, Kind: obs.KindProgress, Network: ns.Network,
				Node: n.Index, Message: "node info", Fields: map[string]any{
					"chain_id": info.ChainID, "block": info.BlockNumber, "peers": info.PeerCount}})
		}
		rep.Nodes = append(rep.Nodes, info)
	}

	rep.Producing = detectProducing(ctx, ns, dial, sleep, delay)
	emit(bus, obs.Event{Phase: obs.PhaseVerify, Kind: obs.KindResult, Network: ns.Network,
		Message: "verify complete", Fields: map[string]any{"producing": rep.Producing, "nodes": len(rep.Nodes)}})
	return rep, nil
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

// detectProducing samples the primary node's height twice, delay apart, and
// reports whether it strictly increased.
func detectProducing(ctx context.Context, ns node.NodeSet, dial func(string) Prober, sleep func(time.Duration), delay time.Duration) bool {
	primary, ok := ns.Primary()
	if !ok {
		return false
	}
	p := dial(primary.RPCURL)
	first, err := p.BlockNumber(ctx)
	if err != nil {
		return false
	}
	sleep(delay)
	second, err := p.BlockNumber(ctx)
	if err != nil {
		return false
	}
	return second > first
}

func emit(bus *obs.Bus, e obs.Event) {
	if bus != nil {
		bus.Publish(e)
	}
}
