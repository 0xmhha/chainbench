package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Target probes are deliberately controllable; this is concurrency evidence,
// not a substitute for live remote deployment or test fault scope acceptance.
type probingJobEngine struct {
	*blockingJobEngine
	blocked atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func newProbingJobEngine() *probingJobEngine {
	return &probingJobEngine{blockingJobEngine: newBlockingJobEngine(), entered: make(chan struct{}, 16), release: make(chan struct{})}
}
func (e *probingJobEngine) Prepare(ctx context.Context, a DeploymentActor, in WebPlanInput) (WebPreparedJob, error) {
	if e.blocked.Load() && in.WorkspaceID == "slow" {
		e.entered <- struct{}{}
		select {
		case <-e.release:
		case <-ctx.Done():
			return WebPreparedJob{}, ctx.Err()
		}
	}
	p, err := e.blockingJobEngine.Prepare(ctx, a, in)
	p.Claims[0].HostIdentity = "host:" + in.WorkspaceID
	return p, err
}

type acceptanceResult struct {
	job WebJob
	err error
}

func TestWebJobSlowProbeDoesNotBlockReadsCancellationOrIndependentStart(t *testing.T) {
	e := newProbingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, a)
	_, owner := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "owner", Operation: "chain.deploy"}, "owner")
	<-e.started
	slow, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "slow", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	fast, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "fast", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	e.blocked.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan acceptanceResult, 1)
	go func() { j, err := s.Start(ctx, a, slow.ID, "slow"); result <- acceptanceResult{j, err} }()
	defer func() {
		cancel()
		close(e.release)
		select {
		case <-result:
		case <-time.After(3 * time.Second):
			t.Error("slow acceptance did not finish")
		}
	}()
	select {
	case <-e.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("slow target was not probed")
	}
	read := make(chan error, 1)
	go func() { _, err := s.Get(owner.ID); read <- err }()
	select {
	case err := <-read:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("target probe holds the store lock and blocks reads")
	}
	cancelled := make(chan error, 1)
	go func() { _, err := s.Cancel(a, owner.ID); cancelled <- err }()
	select {
	case err := <-cancelled:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("target probe blocks cancellation of unrelated work")
	}
	started := make(chan acceptanceResult, 1)
	go func() {
		j, err := s.Start(context.Background(), a, fast.ID, "fast")
		started <- acceptanceResult{j, err}
	}()
	select {
	case got := <-started:
		if got.err != nil {
			t.Fatal(got.err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("independent target start waits for slow probe")
	}
}

func TestWebJobConcurrentIdenticalAcceptanceExecutesOnce(t *testing.T) {
	e := newProbingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, a)
	plan, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "slow", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	e.blocked.Store(true)
	results := make(chan acceptanceResult, 2)
	for i := 0; i < 2; i++ {
		go func() {
			j, err := s.Start(context.Background(), a, plan.ID, "same-key")
			results <- acceptanceResult{j, err}
		}()
	}
	remaining, released := 2, false
	defer func() {
		if !released {
			close(e.release)
		}
		for remaining > 0 {
			select {
			case <-results:
				remaining--
			case <-time.After(3 * time.Second):
				t.Error("acceptance did not finish")
				return
			}
		}
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-e.entered:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("concurrent target validation serialized behind the store lock")
		}
	}
	close(e.release)
	released = true
	first, second := <-results, <-results
	remaining = 0
	if first.err != nil || second.err != nil || first.job.ID != second.job.ID || len(s.List()) != 1 {
		t.Fatal("concurrent replay created duplicate or conflicting work", first, second)
	}
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("accepted executor did not start")
	}
	if e.starts.Load() != 1 {
		t.Fatal("same-key requests executed twice")
	}
}

func TestWebJobCancelledAcceptanceDoesNotStart(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, a)
	plan, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = s.Start(ctx, a, plan.ID, "cancelled-request"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request accepted durable work: %v", err)
	}
	if len(s.List()) != 0 || e.starts.Load() != 0 {
		t.Fatal("cancelled acceptance created work")
	}
}

func TestWebJobConcurrentDifferentKeysCannotClaimSameTarget(t *testing.T) {
	e := newProbingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, a)
	plan, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "slow", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	e.blocked.Store(true)
	results := make(chan acceptanceResult, 2)
	for _, key := range []string{"one", "two"} {
		go func() { j, err := s.Start(context.Background(), a, plan.ID, key); results <- acceptanceResult{j, err} }()
	}
	released, remaining := false, 2
	defer func() {
		if !released {
			close(e.release)
		}
		for remaining > 0 {
			select {
			case <-results:
				remaining--
			case <-time.After(3 * time.Second):
				t.Error("acceptance did not finish")
				return
			}
		}
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-e.entered:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("second probe blocked by first")
		}
	}
	close(e.release)
	released = true
	first, second := <-results, <-results
	remaining = 0
	accepted, conflicts := 0, 0
	for _, r := range []acceptanceResult{first, second} {
		if r.err == nil {
			accepted++
		} else if errors.Is(r.err, ErrDeploymentConflict) {
			conflicts++
		} else {
			t.Fatal(r.err)
		}
	}
	if accepted != 1 || conflicts != 1 || len(s.List()) != 1 {
		t.Fatal("simultaneous claims both accepted", first, second)
	}
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("accepted executor never started")
	}
	if e.starts.Load() != 1 {
		t.Fatal("overlapping work executed twice")
	}
}

func TestWebJobAcceptanceRechecksExpiryAndAuthorizationAfterProbe(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "revoked actor", true: "expired plan"}[expired], func(t *testing.T) {
			e := newProbingJobEngine()
			var revoked atomic.Bool
			s, err := OpenWebJobs(t.TempDir(), e, nil, func(DeploymentActor) error {
				if revoked.Load() {
					return ErrDeploymentForbidden
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			a := DeploymentActor{ID: "operator", Role: "operator"}
			closeFixtureJobs(t, s, DeploymentActor{ID: "admin", Role: "admin"})
			plan, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "slow", Operation: "chain.deploy"})
			if err != nil {
				t.Fatal(err)
			}
			e.blocked.Store(true)
			result := make(chan error, 1)
			go func() { _, err := s.Start(context.Background(), a, plan.ID, "pending"); result <- err }()
			select {
			case <-e.entered:
			case <-time.After(3 * time.Second):
				close(e.release)
				t.Fatal("probe did not start")
			}
			if expired {
				s.mu.Lock()
				p := s.state.Plans[plan.ID]
				p.Public.ExpiresAt = time.Now().Add(-time.Minute)
				s.state.Plans[plan.ID] = p
				s.mu.Unlock()
			} else {
				revoked.Store(true)
			}
			close(e.release)
			want := ErrDeploymentForbidden
			if expired {
				want = ErrDeploymentConflict
			}
			select {
			case err := <-result:
				if !errors.Is(err, want) {
					t.Fatalf("stale acceptance: %v; want %v", err, want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("acceptance did not finish")
			}
			revoked.Store(false)
			if len(s.List()) != 0 || e.starts.Load() != 0 {
				t.Fatal("invalidated review created durable work")
			}
		})
	}
}
