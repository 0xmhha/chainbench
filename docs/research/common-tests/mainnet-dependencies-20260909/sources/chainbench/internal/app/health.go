package app

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// VerifyNetworkIn asks whether a network is producing blocks and what state
// each of its nodes is in.
type VerifyNetworkIn struct {
	// Nodes is the network to sample, when the caller already holds it.
	Nodes node.NodeSet
	// DataDir names a composed workspace to read the network from, and RPCURLs
	// name a running one directly (Chain labels it). Either is resolved here.
	//
	// Resolving used to be "the surface's job", and both surfaces did it — so
	// which endpoints a workspace means, and what an attached set is called,
	// were settled twice.
	DataDir string
	Chain   string
	RPCURLs []string
	// ProgressDelay is the wait between the two block-height samples; zero
	// uses the package default.
	ProgressDelay time.Duration
	// ReadyTimeout bounds how long to wait for the height to start advancing;
	// zero takes a single two-sample reading.
	ReadyTimeout time.Duration
	// Bus receives orchestration events; nil disables emission.
	Bus *collector.Bus
}

// VerifyNetworkOut is the health report.
type VerifyNetworkOut struct {
	Report health.Report
}

// VerifyNetwork samples every node and reports the producing verdict — the
// function behind `chainbench verify` and its MCP mirror, so both surfaces
// return the same verdict from the same code.
func VerifyNetwork(ctx context.Context, d Deps, in VerifyNetworkIn) (VerifyNetworkOut, error) {
	if len(in.Nodes.Nodes) == 0 {
		ns, err := ResolveNodes(ctx, d, in.DataDir, in.Chain, in.RPCURLs)
		if err != nil {
			return VerifyNetworkOut{}, err
		}
		in.Nodes = ns
	}
	rep, err := health.Run(ctx, in.Nodes, health.Options{
		ProgressDelay: in.ProgressDelay,
		ReadyTimeout:  in.ReadyTimeout,
	}, in.Bus)
	return VerifyNetworkOut{Report: rep}, err
}

// ResolveNodes names the network a verb should act on: the endpoints given
// directly, or the ones a composed workspace records.
func ResolveNodes(ctx context.Context, d Deps, dataDir, chain string, rpcURLs []string) (NodeSet, error) {
	if len(rpcURLs) > 0 {
		eps := make([]node.RPCEndpoint, len(rpcURLs))
		for i, u := range rpcURLs {
			eps[i] = node.RPCEndpoint{RPCURL: u}
		}
		return node.AttachedSet(chain, "attached", eps)
	}
	if dataDir != "" {
		res, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: dataDir})
		return res.Nodes, err
	}
	return NodeSet{}, fmt.Errorf("a network is needed: give an endpoint or a workspace")
}
