package app

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

// Reading what a network wrote: the per-node logs a workspace collected.

type (
	// LogSearchIn selects which lines to return.
	LogSearchIn = collector.SearchOpts
	// LogMatch is one matching line, with the node and position it came from.
	LogMatch = collector.Match
)

// LogSearch returns the log lines under a workspace that match.
func LogSearch(_ Deps, dir string, in LogSearchIn) ([]LogMatch, error) {
	if dir == "" {
		return nil, fmt.Errorf("a workspace directory is required")
	}
	return collector.Search(dir, in)
}

// LogTimeline returns a workspace's matching lines in time order, which is what
// answers "what happened, in what order" when a run went wrong.
func LogTimeline(_ Deps, dir string, in LogSearchIn) ([]LogMatch, error) {
	if dir == "" {
		return nil, fmt.Errorf("a workspace directory is required")
	}
	return collector.Timeline(dir, in)
}
