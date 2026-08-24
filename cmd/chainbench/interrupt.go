package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// interruptExitCode is the conventional status for a process killed by SIGINT
// (128 + 2). CI reads it to tell "the operator stopped this" apart from "the
// run failed", which are different facts about a test bench.
const interruptExitCode = 130

// interruptible returns a context cancelled by the first interrupt, and a stop
// function to release the handler.
//
// The point is not a tidy exit — it is that a chainbench run owns node
// processes. Without this, Ctrl-C killed the tool while the nodes it had
// started kept running and unrecorded: their ports stayed bound, the next run
// failed with "address already in use", and nothing said who had left them.
//
// Cancelling the context lets the step unwind through its normal error path,
// which is where the workspace is written (app.withWorkspace saves on failure
// too). So the first interrupt does not kill nodes — it makes them recoverable,
// which `net stop` then acts on. Killing them here instead would race the
// record and could leave the worse state: gone from the machine, still in the
// file.
//
// The second interrupt is deliberately not graceful. An operator pressing
// Ctrl-C twice is saying the first one did not work, and a tool that keeps
// waiting after that is a tool they will kill with -9 — which is how orphans
// are made.
func interruptible(w io.Writer) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		fmt.Fprintln(w, "\ninterrupted — finishing the step and recording what it started (interrupt again to force)")
		cancel()
		<-ch
		fmt.Fprintln(w, "forced — nodes this run started are still running and may not be recorded; `net stop --data-dir <dir>` clears what was, and the rest need finding by hand")
		os.Exit(interruptExitCode)
	}()
	return ctx, func() {
		signal.Stop(ch)
		cancel()
	}
}

// interrupted reports whether err is the interrupt travelling out as a context
// cancellation. A cancelled run is not a failed run: the distinction is what
// makes the exit code readable.
func interrupted(ctx context.Context, err error) bool {
	return err != nil && errors.Is(ctx.Err(), context.Canceled) && errors.Is(err, context.Canceled)
}
