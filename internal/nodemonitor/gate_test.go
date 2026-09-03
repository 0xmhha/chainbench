package nodemonitor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

// scriptObserver returns a scripted set of facts per round; the last entry
// repeats once the script is exhausted.
type scriptObserver struct {
	rounds [][]nodemonitor.Facts
	seen   int
	err    error
}

func (o *scriptObserver) Observe(context.Context) ([]nodemonitor.Facts, error) {
	if o.err != nil {
		return nil, o.err
	}
	i := o.seen
	if i >= len(o.rounds) {
		i = len(o.rounds) - 1
	}
	o.seen++
	return o.rounds[i], nil
}

// countRestarter records restarts and can fail on demand.
type countRestarter struct {
	calls []int
	fail  error
}

func (r *countRestarter) Restart(_ context.Context, node int) error {
	r.calls = append(r.calls, node)
	return r.fail
}

// fakeClock counts sleeps without waiting.
type fakeClock struct{ slept int }

func (c *fakeClock) Sleep(context.Context, time.Duration) error { c.slept++; return nil }

// capSink captures evidence.
type capSink struct {
	verdicts   []nodemonitor.NodeReport
	recoveries int
}

func (s *capSink) Verdict(_ int, r nodemonitor.NodeReport) { s.verdicts = append(s.verdicts, r) }
func (s *capSink) Recovery(int, int, error)                { s.recoveries++ }

func node(n int, mut func(*nodemonitor.Facts)) nodemonitor.Facts {
	f := nodemonitor.Facts{
		Node: n, Wanted: true, WantChainID: 1, ChainID: 1,
		PIDAlive: true, RPCUp: true, Advancing: true,
	}
	if mut != nil {
		mut(&f)
	}
	return f
}

func TestGate_AllReadyImmediately(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{{node(1, nil), node(2, nil)}}}
	res, err := nodemonitor.Gate(context.Background(), obs, &countRestarter{}, &fakeClock{}, nil, nodemonitor.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Terminate != "" {
		t.Fatalf("res = %+v, want OK", res)
	}
}

func TestGate_WaitsThenReady(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{
		{node(1, func(f *nodemonitor.Facts) { f.Advancing = false })}, // WAITABLE
		{node(1, func(f *nodemonitor.Facts) { f.Advancing = false })}, // WAITABLE
		{node(1, nil)}, // READY
	}}
	clock := &fakeClock{}
	res, err := nodemonitor.Gate(context.Background(), obs, &countRestarter{}, clock, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: 30 * time.Second, WaitInterval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("res = %+v, want OK after waiting", res)
	}
	if clock.slept != 2 {
		t.Errorf("slept %d times, want 2", clock.slept)
	}
}

func TestGate_WaitBudgetExhausted(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{
		{node(1, func(f *nodemonitor.Facts) { f.Advancing = false })}, // always WAITABLE
	}}
	res, err := nodemonitor.Gate(context.Background(), obs, &countRestarter{}, &fakeClock{}, nil,
		nodemonitor.Options{MaxNodeMonitorTimeout: 2 * time.Second, WaitInterval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.Terminate == "" {
		t.Fatalf("res = %+v, want terminate on budget exhaustion", res)
	}
}

func TestGate_RestartsThenReady(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{
		{node(1, func(f *nodemonitor.Facts) { f.PIDAlive = false })}, // RESTARTABLE
		{node(1, nil)}, // READY after restart
	}}
	rst := &countRestarter{}
	sink := &capSink{}
	res, err := nodemonitor.Gate(context.Background(), obs, rst, &fakeClock{}, sink, nodemonitor.Options{MaxRestarts: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("res = %+v, want OK after restart", res)
	}
	if len(rst.calls) != 1 || rst.calls[0] != 1 {
		t.Errorf("restarts = %v, want [1]", rst.calls)
	}
	if sink.recoveries != 1 {
		t.Errorf("recoveries recorded = %d, want 1", sink.recoveries)
	}
}

func TestGate_RestartCapExhausted(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{
		{node(1, func(f *nodemonitor.Facts) { f.PIDAlive = false })}, // always RESTARTABLE
	}}
	rst := &countRestarter{}
	res, err := nodemonitor.Gate(context.Background(), obs, rst, &fakeClock{}, nil, nodemonitor.Options{MaxRestarts: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.Terminate == "" {
		t.Fatalf("res = %+v, want terminate on restart cap", res)
	}
	if len(rst.calls) != 2 {
		t.Errorf("restarts = %v, want exactly 2 before giving up", rst.calls)
	}
}

func TestGate_FatalTerminatesWithoutRestart(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{
		{node(1, func(f *nodemonitor.Facts) { f.Forked = true })}, // FATAL
	}}
	rst := &countRestarter{}
	res, err := nodemonitor.Gate(context.Background(), obs, rst, &fakeClock{}, nil, nodemonitor.Options{MaxRestarts: 3})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.Terminate == "" {
		t.Fatalf("res = %+v, want terminate on FATAL", res)
	}
	if len(rst.calls) != 0 {
		t.Errorf("FATAL must not restart, got %v", rst.calls)
	}
}

func TestGate_RestartErrorTerminates(t *testing.T) {
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{
		{node(1, func(f *nodemonitor.Facts) { f.PIDAlive = false })},
	}}
	rst := &countRestarter{fail: errors.New("driver down")}
	res, err := nodemonitor.Gate(context.Background(), obs, rst, &fakeClock{}, nil, nodemonitor.Options{MaxRestarts: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.Terminate == "" {
		t.Fatalf("res = %+v, want terminate when restart fails", res)
	}
}

func TestGate_ObserveErrorIsError(t *testing.T) {
	obs := &scriptObserver{err: errors.New("rpc dial")}
	_, err := nodemonitor.Gate(context.Background(), obs, &countRestarter{}, &fakeClock{}, nil, nodemonitor.Options{})
	if err == nil {
		t.Fatal("want an error when observation fails")
	}
}

func TestGate_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	obs := &scriptObserver{rounds: [][]nodemonitor.Facts{{node(1, nil)}}}
	if _, err := nodemonitor.Gate(ctx, obs, &countRestarter{}, &fakeClock{}, nil, nodemonitor.Options{}); err == nil {
		t.Fatal("want ctx error when canceled")
	}
}
