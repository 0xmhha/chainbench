package upgrade_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
)

// deadEndpoint returns a URL nothing answers on, by taking a port and closing it.
// A closed port refuses immediately, so the wait spends its whole time in the
// sleep between attempts — which is exactly where cancellation has to land.
func deadEndpoint(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return "http://" + addr
}

// TestWaitEndpointsReady_ReturnsWhenTheContextIsCancelled is the defect this
// covers. The wait used time.Sleep, the one form of waiting a context cannot
// interrupt, so a cancelled run kept dialling a dead endpoint for the rest of its
// budget — thirty seconds on the handoff path — and reported the cancellation that
// much later.
//
// The budget here is deliberately far longer than the test's patience: if
// cancellation is not honoured, this fails by timing out rather than by passing
// slowly.
func TestWaitEndpointsReady_ReturnsWhenTheContextIsCancelled(t *testing.T) {
	ep := deadEndpoint(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- upgrade.WaitEndpointsReady(ctx, []string{ep}, 5*time.Minute) }()

	// Long enough that the wait is inside its sleep, short enough that a passing
	// run is quick.
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled — the caller's cancellation has to be what comes back", err)
		}
		if elapsed := time.Since(start); elapsed > 10*time.Second {
			t.Errorf("returned after %s; cancellation was not honoured promptly", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the wait ignored cancellation and is still polling")
	}
}

// TestWaitEndpointsReady_StillTimesOutOnItsOwn keeps the cancellation path from
// swallowing the deadline: with no cancellation, an endpoint that never answers
// still has to produce the timeout error that names it.
func TestWaitEndpointsReady_StillTimesOutOnItsOwn(t *testing.T) {
	ep := deadEndpoint(t)
	err := upgrade.WaitEndpointsReady(context.Background(), []string{ep}, 300*time.Millisecond)
	if err == nil {
		t.Fatal("a dead endpoint must fail the wait")
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the timeout, not a cancellation", err)
	}
	if !contains(err.Error(), ep) {
		t.Errorf("the error does not name the endpoint: %v", err)
	}
}

// TestWaitEndpointsReady_SkipsEmptyEndpoints pins the one branch above the loop:
// a blank entry is not an endpoint that never answers, it is nothing to wait for.
func TestWaitEndpointsReady_SkipsEmptyEndpoints(t *testing.T) {
	if err := upgrade.WaitEndpointsReady(context.Background(), []string{"", ""}, time.Millisecond); err != nil {
		t.Errorf("blank endpoints should be skipped, got %v", err)
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
