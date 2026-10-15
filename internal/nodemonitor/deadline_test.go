package nodemonitor_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

type observeFunc func(context.Context) ([]nodemonitor.Facts, error)

func (f observeFunc) Observe(ctx context.Context) ([]nodemonitor.Facts, error) { return f(ctx) }

type restartFunc func(context.Context, int) error

func (f restartFunc) Restart(ctx context.Context, n int) error { return f(ctx, n) }

func TestGateDeadlineIncludesObservation(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	started := time.Now()
	obs := observeFunc(func(ctx context.Context) ([]nodemonitor.Facts, error) {
		deadline, ok := ctx.Deadline()
		if !ok || deadline.Sub(started) > 100*time.Millisecond {
			t.Error("observer did not receive the readiness deadline")
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	res, err := nodemonitor.Gate(parent, obs, &countRestarter{}, &fakeClock{}, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: 20 * time.Millisecond})
	if err != nil || res.OK || !strings.Contains(res.Terminate, "deadline") {
		t.Fatalf("readiness timeout must be a terminating result, not a parent cancellation: %+v %v", res, err)
	}
}

func TestGateDeadlineIncludesRestart(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{{node(1, func(f *nodemonitor.Facts) { f.PIDAlive = false })}}}
	rst := restartFunc(func(ctx context.Context, _ int) error { <-ctx.Done(); return ctx.Err() })
	res, err := nodemonitor.Gate(parent, obs, rst, &fakeClock{}, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: 20 * time.Millisecond})
	if err != nil || res.OK || !strings.Contains(res.Terminate, "deadline") || strings.Contains(res.Terminate, "restart failed") {
		t.Fatalf("restart must share the readiness budget: %+v %v", res, err)
	}
}

func TestGateRejectsReadyObservationAfterDeadline(t *testing.T) {
	obs := observeFunc(func(context.Context) ([]nodemonitor.Facts, error) {
		time.Sleep(30 * time.Millisecond)
		return []nodemonitor.Facts{node(1, nil)}, nil
	})
	res, err := nodemonitor.Gate(context.Background(), obs, &countRestarter{}, &fakeClock{}, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: 5 * time.Millisecond})
	if err != nil || res.OK || !strings.Contains(res.Terminate, "deadline") {
		t.Fatalf("late ready response was accepted: %+v %v", res, err)
	}
}

func TestGateParentDeadlineRemainsAnError(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	obs := observeFunc(func(ctx context.Context) ([]nodemonitor.Facts, error) { <-ctx.Done(); return nil, ctx.Err() })
	res, err := nodemonitor.Gate(parent, obs, &countRestarter{}, &fakeClock{}, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: time.Second})
	if !errors.Is(err, context.DeadlineExceeded) || res.Terminate != "" || res.OK {
		t.Fatalf("parent deadline became a product timeout: %+v %v", res, err)
	}
}

type sleepFunc func(context.Context, time.Duration) error

func (f sleepFunc) Sleep(ctx context.Context, d time.Duration) error { return f(ctx, d) }

func TestGateDeadlineIncludesWait(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{{node(1, func(f *nodemonitor.Facts) { f.Advancing = false })}}}
	clock := sleepFunc(func(ctx context.Context, _ time.Duration) error { <-ctx.Done(); return ctx.Err() })
	res, err := nodemonitor.Gate(parent, obs, &countRestarter{}, clock, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: 20 * time.Millisecond, WaitInterval: time.Millisecond})
	if err != nil || res.OK || !strings.Contains(res.Terminate, "deadline") || len(res.Reports) != 1 || res.Reports[0].Verdict != nodemonitor.Waitable {
		t.Fatalf("wait must share the readiness budget and retain the last verdict: %+v %v", res, err)
	}
}
