package collector_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

// flakyReader fails failuresBefore times, then returns payload.
type flakyReader struct {
	calls          int
	failuresBefore int
	gotOffset      int64
	payload        []byte
	err            error
}

func (r *flakyReader) ReadFrom(_ context.Context, _ string, offset int64) ([]byte, error) {
	r.calls++
	r.gotOffset = offset
	if r.err != nil {
		return nil, r.err
	}
	if r.calls <= r.failuresBefore {
		return nil, errors.New("connection dropped")
	}
	return r.payload, nil
}

func noSleep(context.Context, time.Duration) error { return nil }

// TestReconnecting_RecoversWithinAttempts: two drops then success, from the same
// offset, within the attempt budget.
func TestReconnecting_RecoversWithinAttempts(t *testing.T) {
	inner := &flakyReader{failuresBefore: 2, payload: []byte("line\n")}
	r := collector.ReconnectingLogReader{Reader: inner, Attempts: 3, Sleep: noSleep}
	got, err := r.ReadFrom(context.Background(), "/n.log", 42)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if string(got) != "line\n" {
		t.Errorf("payload = %q, want line", got)
	}
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3 (2 drops + 1 success)", inner.calls)
	}
	if inner.gotOffset != 42 {
		t.Errorf("offset = %d, want 42 preserved across reconnects", inner.gotOffset)
	}
}

// TestReconnecting_ExhaustsAttempts: failures through every attempt return the
// last error, so the tail loop retries next tick from the unchanged offset.
func TestReconnecting_ExhaustsAttempts(t *testing.T) {
	inner := &flakyReader{failuresBefore: 99, payload: nil}
	r := collector.ReconnectingLogReader{Reader: inner, Attempts: 3, Sleep: noSleep}
	if _, err := r.ReadFrom(context.Background(), "/n.log", 0); err == nil {
		t.Fatal("want an error when every attempt fails")
	}
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3", inner.calls)
	}
}

// TestReconnecting_ContextCancel: a cancelled context stops retrying at once.
func TestReconnecting_ContextCancel(t *testing.T) {
	inner := &flakyReader{err: errors.New("dropped")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := collector.ReconnectingLogReader{Reader: inner, Attempts: 5, Sleep: noSleep}
	if _, err := r.ReadFrom(ctx, "/n.log", 0); err == nil {
		t.Fatal("want ctx error")
	}
	if inner.calls != 1 {
		t.Errorf("calls = %d, want 1 (cancel stops retry)", inner.calls)
	}
}
