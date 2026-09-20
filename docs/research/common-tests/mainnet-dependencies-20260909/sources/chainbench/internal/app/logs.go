package app

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

// Reading what a network wrote: the per-node logs a workspace collected.

// LogSearchIn is a log search: where to look and which lines to return.
//
// The options are embedded rather than repeated, so the fields a surface
// renders are declared once, in collector.SearchOpts.
type LogSearchIn struct {
	Dir string `cb:"workspace-dir,required" help:"workspace directory (searches <workspace-dir>/logs)"`
	collector.SearchOpts
}

type (
	// LogMatch is one matching line, with the node and position it came from.
	LogMatch = collector.Match
	// LogSearchFilter is which lines a search returns. Re-exported so a surface
	// says what it wants without reaching past app for the vocabulary
	// (architecture-v2 §2). Not LogFilter, which core/rpc already uses for an
	// eth_getLogs filter — a different thing entirely.
	LogSearchFilter = collector.SearchOpts
)

// LogSearch returns the log lines under a workspace that match.
func LogSearch(_ context.Context, _ Deps, in LogSearchIn) ([]LogMatch, error) {
	dir := in.Dir
	if dir == "" {
		return nil, fmt.Errorf("a workspace directory is required")
	}
	return collector.Search(dir, in.SearchOpts)
}

// LogTimeline returns a workspace's matching lines in time order, which is what
// answers "what happened, in what order" when a run went wrong.
func LogTimeline(_ context.Context, _ Deps, in LogSearchIn) ([]LogMatch, error) {
	dir := in.Dir
	if dir == "" {
		return nil, fmt.Errorf("a workspace directory is required")
	}
	return collector.Timeline(dir, in.SearchOpts)
}
