package collector

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// Collector tails node logs and samples chainstate for one environment. For
// attach mode (rpc-url only) it degrades to RPC-only, without log tails.
type Collector interface {
	// Start begins per-node log tails and the chainstate sampler.
	Start(ctx context.Context, env session.Environment) error
	// WaitLog blocks until pattern appears in the node's log or timeout, and
	// returns the matched line's location for assertion provenance.
	WaitLog(ctx context.Context, nodeName, pattern string, timeout time.Duration) (LogMatch, error)
	// Snapshot returns the latest cross-node chainstate.
	Snapshot() Chainstate
	// Stop tears down the tails and sampler.
	Stop() error
}

// LogMatch locates a matched log line for assertion provenance. Lines is the
// inclusive [start, end] line range within File.
type LogMatch struct {
	File       string
	Lines      [2]int
	ByteOffset int64
	Text       string
}

// Chainstate is a cross-node snapshot used by verification and the dashboard.
type Chainstate struct {
	// Heights maps node name to its latest block height.
	Heights map[string]uint64
	// BPParticipation maps a producer to its signed-block count in the window.
	BPParticipation map[string]int
	// Peers maps node name to its peer count.
	Peers map[string]int
	// Forked is true when nodes report divergent block hashes at a height.
	Forked bool
}
