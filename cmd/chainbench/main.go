// Command chainbench is the user-facing CLI (requirement #15) that drives the
// three-phase pipeline: setup (plan/launch a local network), verify (confirm
// block production over RPC), test (run cases), plus faucet and chain listing.
// It is built on cobra and imports all chain plugins for registration.
package main

import (
	"fmt"
	"os"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	_ "github.com/0xmhha/chainbench/tests/all"
)

func main() {
	// The context is cancellable so an interrupt reaches the run rather than
	// only the shell: a command that started nodes gets the chance to stop
	// them. See interrupt.go.
	ctx, stop := interruptible(os.Stderr)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		if interrupted(ctx, err) {
			fmt.Fprintln(os.Stderr, "interrupted")
			os.Exit(interruptExitCode)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(exitCode(err))
	}
}
