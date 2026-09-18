// Package nodemonitor decides whether a network is fit to run a test on, and
// owns the limited recovery that a not-yet-fit network is allowed. It combines
// the facts the atomic observation modules already produce — process/inspect
// (pid alive), health (rpc, block advance, peers, sync, chain id), collector
// (fork/divergence, consensus participation), and preflight (does this node
// belong) — into one per-node verdict:
//
//	READY       the node is fit; start the test
//	WAITABLE    alive and reachable but not there yet; wait up to MaxNodeMonitorTimeout
//	RESTARTABLE a restart may clear it; restart and re-check, capped by MaxRestarts
//	FATAL       a destructive remedy would be needed; terminate, never auto-applied
//
// It observes nothing itself and reimplements none of the observation work: the
// Observer seam hands it facts gathered by the existing modules, and the
// Restarter seam performs a restart through the existing restart verb. A
// destructive remedy — deleting data, force-rewinding, swapping genesis — is
// never applied automatically; the states that would need one are FATAL.
//
// The gate is meant to run before an environment is reused, after a fresh
// compose, after a restart / binary-config change / partition heal, and before
// each test. Every verdict and every recovery attempt is recorded through the
// EvidenceSink.
package nodemonitor
