// Package collector gathers results and diagnostics out-of-process (DDD context
// C4): per-node live log tails plus a periodic cross-node chainstate snapshot.
// It never blocks a node process; log tails spool to disk under back-pressure
// while chainstate samples may be merged or dropped.
//
// Status: interface freeze only (T0.1). Implementation lands in T3.3.
package collector
