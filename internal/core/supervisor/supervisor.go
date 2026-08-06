package supervisor

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
)

// Supervisor owns node bring-up: setup produces the plan and concurrent
// provision/launch primitives, and the supervisor orchestrates launch, the
// health gate, backoff recovery, and teardown.
type Supervisor interface {
	// BringUp launches the plan's nodes behind a health gate and returns the
	// resulting node set and a classified diagnosis.
	BringUp(ctx context.Context, plan setup.Plan, opts Options) (node.NodeSet, Diagnosis, error)
	// Teardown stops the nodes (SIGTERM then SIGKILL), which also stops each
	// node's embedded etcd. It removes the datadir only when opts requests it.
	Teardown(ctx context.Context, ns node.NodeSet, opts TeardownOpts) error
}

// Options tunes bring-up: health gating and fork-aware binary swaps.
type Options struct {
	// LeaderGate polls the producer's etcd until a leader is ready.
	LeaderGate bool
	// AlignJoinGap aligns node start times to the etcd join slot; the gap is
	// derived from the cluster size, not passed in.
	AlignJoinGap bool
	MaxAttempts  int
	Backoff      func(attempt int) time.Duration
	// ForkSwaps schedules same-chain (type-2) binary swaps before a fork block.
	ForkSwaps []ForkSwap
}

// TeardownOpts controls teardown. RemoveDataDir is a separate concern from
// stopping the process, used for re-setup and disk management.
type TeardownOpts struct {
	RemoveDataDir bool
	Grace         time.Duration
}

// ForkSwap schedules a binary swap on a node before a fork block (type-2).
type ForkSwap struct {
	Node     string
	Fork     string
	ToBinary string
}

// Diagnosis is the classified outcome of a bring-up attempt. On failure, Mode
// and Detail carry the real cause; ProducerLog preserves a tail for the session.
type Diagnosis struct {
	OK          bool
	Mode        FailureMode
	Detail      string
	ProducerLog string
}

// FailureMode classifies a bring-up failure. No "flaky" label is used.
type FailureMode int

const (
	// EtcdJoinFailed means a node could not join the etcd cluster.
	EtcdJoinFailed FailureMode = iota
	// EtcdStale means a stale datadir blocked cluster formation on restart.
	EtcdStale
	// ForkNotCrossed means the target fork block was never reached.
	ForkNotCrossed
	// QuorumLost means the validator set fell below quorum.
	QuorumLost
	// RPCUnready means a node's RPC never became ready.
	RPCUnready
)

// String returns the failure-mode label.
func (m FailureMode) String() string {
	switch m {
	case EtcdJoinFailed:
		return "EtcdJoinFailed"
	case EtcdStale:
		return "EtcdStale"
	case ForkNotCrossed:
		return "ForkNotCrossed"
	case QuorumLost:
		return "QuorumLost"
	case RPCUnready:
		return "RPCUnready"
	default:
		return "Unknown"
	}
}
