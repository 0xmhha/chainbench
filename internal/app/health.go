package app

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// VerifyNetworkIn asks whether a network is producing blocks and what state
// each of its nodes is in.
type VerifyNetworkIn struct {
	// Nodes is the network to sample (from a data dir, --rpc endpoints, or a
	// composed workspace — resolving it is the surface's job).
	Nodes node.NodeSet
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
func VerifyNetwork(ctx context.Context, _ Deps, in VerifyNetworkIn) (VerifyNetworkOut, error) {
	rep, err := health.Run(ctx, in.Nodes, health.Options{
		ProgressDelay: in.ProgressDelay,
		ReadyTimeout:  in.ReadyTimeout,
	}, in.Bus)
	return VerifyNetworkOut{Report: rep}, err
}
