package nodemonitor

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// Facts is one node's observed facts, gathered by the atomic observation
// modules and handed here to be judged. nodemonitor observes nothing itself; a
// zero field means "not observed / not constrained" and does not by itself
// fail a node. Fields:
//
//   - Wanted / WantChainID / WantPeers / WantParticipate come from the target
//     topology and genesis (0 or false = not constrained).
//   - PIDAlive is from process/inspect; RPCUp, ChainID, Height, Advancing,
//     Syncing, Peers from health; Forked and Participating from collector;
//     Failure from a classified launch/bring-up error (process.Classify).
type Facts struct {
	Node  int
	Label string

	Wanted          bool // this node belongs in the target topology (preflight)
	WantChainID     uint64
	WantPeers       int
	WantParticipate bool // a validator/producer expected to seal

	PIDAlive      bool
	RPCUp         bool
	ChainID       uint64
	Height        uint64
	Advancing     bool // block height rose within health's ready window
	Syncing       bool
	Peers         int
	Participating bool // sealing / participating in consensus (collector)
	Forked        bool // chain diverged (collector)

	Failure process.FailureMode // classified launch/bring-up failure, if any
}

// NodeReport is one node's verdict and the reasons behind it (the evidence).
type NodeReport struct {
	Node    int
	Label   string
	Verdict Verdict
	Reasons []string
}

// Classify maps one node's facts to a verdict. It is pure and deterministic —
// the whole READY/WAITABLE/RESTARTABLE/FATAL policy lives here. The checks run
// worst-first: a destructive-remedy state (FATAL) wins over a restartable one,
// which wins over a wait, which wins over READY.
func Classify(f Facts) NodeReport {
	r := NodeReport{Node: f.Node, Label: f.Label}
	reason := func(v Verdict, format string, a ...any) NodeReport {
		r.Verdict = v
		r.Reasons = append(r.Reasons, fmt.Sprintf(format, a...))
		return r
	}

	// A node the topology does not want is not this gate's concern — preflight
	// decides whether to tear it down. It never blocks readiness.
	if !f.Wanted {
		return reason(Ready, "not part of the target topology")
	}

	// FATAL — clearing it would need a destructive remedy, never auto-applied.
	if f.RPCUp && f.WantChainID != 0 && f.ChainID != f.WantChainID {
		return reason(Fatal, "chain id %d != wanted %d: wrong genesis", f.ChainID, f.WantChainID)
	}
	if f.Forked {
		return reason(Fatal, "chain diverged; needs rewind")
	}
	switch f.Failure {
	case process.EtcdStale:
		return reason(Fatal, "stale etcd datadir; needs data removal")
	case process.QuorumLost:
		return reason(Fatal, "validator set fell below quorum")
	}

	// RESTARTABLE — a fresh start may clear it.
	if !f.PIDAlive {
		return reason(Restartable, "process not alive")
	}
	switch f.Failure {
	case process.RPCUnready:
		return reason(Restartable, "rpc not ready (%s)", f.Failure)
	case process.EtcdJoinFailed:
		return reason(Restartable, "etcd join failed")
	case process.ForkNotCrossed:
		return reason(Restartable, "fork not crossed")
	}
	if !f.RPCUp {
		return reason(Restartable, "rpc not ready")
	}

	// WAITABLE — alive and reachable, just not there yet.
	if f.Syncing {
		r = reason(Waitable, "still syncing")
	}
	if !f.Advancing {
		r = reason(Waitable, "block height not advancing yet")
	}
	if f.WantPeers > 0 && f.Peers < f.WantPeers {
		r = reason(Waitable, "peers %d < wanted %d", f.Peers, f.WantPeers)
	}
	if f.WantParticipate && !f.Participating {
		r = reason(Waitable, "not yet participating in consensus")
	}
	if r.Verdict == Waitable {
		return r
	}

	return reason(Ready, "alive, rpc ready, advancing")
}
