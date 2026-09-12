// Package wait is the one cancellable pause in the repository.
//
// Every polling loop needs to wait between attempts, and there are two ways to
// write it. time.Sleep is the shorter one and it is the wrong one: a context
// cannot interrupt it, so a caller that has given up keeps waiting for the rest
// of the budget. The select form is correct and was hand-written at eleven call
// sites, which is eleven chances to write the short one by mistake — and it was
// written by mistake once, in the handoff's endpoint wait, where a cancelled run
// kept dialling a dead endpoint for thirty seconds.
//
// One named function makes the correct form the short one.
package wait

import (
	"context"
	"time"
)

// Sleep pauses for d and returns nil, or returns the context's error as soon as
// the caller gives up — whichever happens first. It is time.Sleep with a context,
// which is what a polling loop always wanted.
//
// A non-positive d returns immediately, and still reports an already-cancelled
// context: a zero interval is a caller that wants to retry at once, not one that
// wants to ignore cancellation. That is the only thing the up-front ctx.Err()
// check does that the select below does not — with a positive d the select already
// reports a cancelled context without waiting.
func Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
