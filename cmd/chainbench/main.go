// Command chainbench is the user-facing CLI (requirement #15): it composes a
// network (the chain command / net verbs), verifies block production, and runs
// DSL test specs against it (the run command), plus faucet and chain listing.
// It is built on cobra and imports all chain plugins for registration.
package main

import (
	"fmt"
	"os"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
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
