package wait_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/wait"
)

// TestSleep_WaitsTheWholeIntervalWhenNobodyCancels is the ordinary case, and it is
// worth pinning because the obvious wrong implementation — returning as soon as the
// context is merely alive — would turn every polling loop into a busy loop.
func TestSleep_WaitsTheWholeIntervalWhenNobodyCancels(t *testing.T) {
	start := time.Now()
	if err := wait.Sleep(context.Background(), 80*time.Millisecond); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if elapsed := time.Since(start); elapsed < 70*time.Millisecond {
		t.Errorf("returned after %s; it did not wait", elapsed)
	}
}

// TestSleep_ReturnsAsSoonAsTheCallerGivesUp is the whole reason this package
// exists. The interval is far longer than the test's patience, so an
// implementation that cannot be interrupted fails by timing out rather than by
// passing slowly.
func TestSleep_ReturnsAsSoonAsTheCallerGivesUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- wait.Sleep(ctx, time.Hour) }()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("returned after %s; cancellation was not prompt", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the pause ignored cancellation")
	}
}

// TestSleep_ReportsADeadlineThatHasAlreadyPassed keeps the timeout case distinct
// from cancellation, because a caller that logs the error needs to know which
// happened.
func TestSleep_ReportsADeadlineThatHasAlreadyPassed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // let the deadline pass

	err := wait.Sleep(ctx, time.Hour)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

// TestSleep_AnAlreadyCancelledContextDoesNotWaitFirst pins the behaviour a polling
// loop depends on: cancelling between attempts must not cost one more interval —
// with a second-long poll and a cancelled run, that would be a second of delay per
// remaining node.
//
// It holds with or without the up-front ctx.Err() check, because the select
// reports a cancelled context without waiting. Measured while mutating: removing
// that check fails only the zero-interval case, which is what it is really for.
func TestSleep_AnAlreadyCancelledContextDoesNotWaitFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := wait.Sleep(ctx, 2*time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("waited %s before reporting an already-cancelled context", elapsed)
	}
}

// TestSleep_ZeroIntervalStillHonoursCancellation is the boundary a "return early
// if d <= 0" shortcut gets wrong: a caller that wants to retry at once is not a
// caller that wants to ignore cancellation.
func TestSleep_ZeroIntervalStillHonoursCancellation(t *testing.T) {
	if err := wait.Sleep(context.Background(), 0); err != nil {
		t.Errorf("a zero interval with a live context should return nil, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := wait.Sleep(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("a zero interval with a cancelled context should report it, got %v", err)
	}
	if err := wait.Sleep(ctx, -time.Second); !errors.Is(err, context.Canceled) {
		t.Errorf("a negative interval with a cancelled context should report it, got %v", err)
	}
}
