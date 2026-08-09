// Package collector gathers results and diagnostics out-of-process (DDD context
// C4): per-node live log tails plus a periodic cross-node chainstate snapshot.
// It never blocks a node process; a failing probe is skipped and a live tail
// only advances past complete lines.
//
// The collector samples each node's height, peers, and head block over RPC,
// tallies bp participation from head producers over a bounded window, and flags
// forks/reorgs when a known height reports a divergent hash. With Deps.OnLine
// set it also tails each node's log live, the seam that mirrors logs to the
// dashboard. Remote (SSH) tailing is a follow-up seam.
package collector
