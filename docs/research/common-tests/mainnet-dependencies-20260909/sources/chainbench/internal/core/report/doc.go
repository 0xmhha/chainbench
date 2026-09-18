// Package report builds a run's root report by aggregating what the session
// already persisted: each test's verdict and the on-disk evidence that backs it.
//
// It is the single owner of the run-level report. Verdicts are produced by the
// test engine and persisted into the session (status.json); this package never
// re-runs or re-judges anything, it only reads the session artifact tree
// (through internal/core/session) and links each verdict to its evidence, then
// writes report.json at the session root. The engine calls Build after the
// session is saved; CLI and MCP read report.json back for display.
//
// The import direction is one way — report depends on session, session depends
// on neither — so the aggregator sits above the record layer without the record
// layer knowing it exists.
package report
