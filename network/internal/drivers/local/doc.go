// Package local implements a subprocess runner that executes the chainbench
// CLI (chainbench.sh) on the local machine. It is a pure process runner —
// stdout/stderr are streamed to the structured logger, and RunResult exposes
// the captured buffers plus exit code for the caller's inspection.
//
// This package does not emit bus events. Higher layers (command handlers)
// translate run outcomes into semantic events like node.stopped.
package local
