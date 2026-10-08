package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// This controllable executor tests job ownership and persistence; it is not
// evidence of chain deployment or end-to-end acceptance.
type blockingJobEngine struct {
	started     chan struct{}
	cleanups    atomic.Int32
	starts      atomic.Int32
	fingerprint string
}

func newBlockingJobEngine() *blockingJobEngine {
	return &blockingJobEngine{started: make(chan struct{}, 8), fingerprint: strings.Repeat("a", 64)}
}
func (e *blockingJobEngine) Prepare(_ context.Context, _ DeploymentActor, in WebPlanInput) (WebPreparedJob, error) {
	return WebPreparedJob{Fingerprint: e.fingerprint, Claims: []WebResourceClaim{{HostIdentity: "fixture-host", DataPath: "/fixture/nodes", Ports: []int{8545}}}, Phases: []string{"deploy"}, Payload: json.RawMessage(`{}`)}, nil
}
func (e *blockingJobEngine) Execute(ctx context.Context, _ DeploymentActor, _ WebPreparedJob, report func(WebJobPhase) error) (WebJobResult, error) {
	e.starts.Add(1)
	if err := report(WebJobPhase{Name: "deploy", State: "running", Message: "password=private-job-marker"}); err != nil {
		return WebJobResult{}, err
	}
	e.started <- struct{}{}
	<-ctx.Done()
	return WebJobResult{NodeDisposition: "retained", PartialEffects: []string{"one node initialized"}}, ctx.Err()
}
func (e *blockingJobEngine) Cleanup(context.Context, DeploymentActor, WebPreparedJob) (WebJobResult, error) {
	e.cleanups.Add(1)
	return WebJobResult{NodeDisposition: "cleaned"}, nil
}

func waitJob(t *testing.T, s *WebJobs, id string) WebJob {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, err := s.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if terminalWebJob(job.State) {
			return job
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("job did not reach a terminal state")
	return WebJob{}
}
func startFixtureJob(t *testing.T, s *WebJobs, a DeploymentActor, in WebPlanInput, key string) (WebPlan, WebJob) {
	t.Helper()
	p, err := s.Plan(context.Background(), a, in)
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.Start(context.Background(), a, p.ID, key)
	if err != nil {
		t.Fatal(err)
	}
	return p, j
}

func TestWebJobsDetachedIdempotentOwnershipAndPhysicalConflict(t *testing.T) {
	e := newBlockingJobEngine()
	root := t.TempDir()
	secrets, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = secrets.SaveCredential(DeploymentActor{ID: "alice", Role: "operator"}, DeploymentCredentialInput{Label: "fixture", Kind: "password", SSHUser: "alice", Password: "private-job-marker"}); err != nil {
		t.Fatal(err)
	}
	s, err := OpenWebJobs(root, e, secrets.RedactWeb, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	bob := DeploymentActor{ID: "bob", Role: "operator"}
	p, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "first-alias", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	request, cancel := context.WithCancel(context.Background())
	j, err := s.Start(request, a, p.ID, "one")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("job cancelled with HTTP context")
	}
	repeat, err := s.Start(context.Background(), a, p.ID, "one")
	if err != nil || repeat.ID != j.ID || e.starts.Load() != 1 {
		t.Fatalf("duplicate: %v %v", repeat, err)
	}
	other, err := s.Plan(context.Background(), bob, WebPlanInput{WorkspaceID: "other-alias", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Start(context.Background(), bob, other.ID, "two"); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("physical alias collision accepted: %v", err)
	}
	if _, err = s.Start(context.Background(), a, other.ID, "one"); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("key/body mismatch accepted: %v", err)
	}
	if _, err = s.Cancel(bob, j.ID); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatalf("foreign cancellation: %v", err)
	}
	if _, err = s.Cancel(DeploymentActor{ID: "viewer", Role: "viewer"}, j.ID); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal(err)
	}
	if _, err = s.Cancel(DeploymentActor{ID: "admin", Role: "admin"}, j.ID); err != nil {
		t.Fatal(err)
	}
	terminal := waitJob(t, s, j.ID)
	if terminal.State != "cancelled" || terminal.Retention != "retain" || terminal.NodeDisposition != "retained" || len(terminal.PartialEffects) != 1 || e.cleanups.Load() != 0 {
		t.Fatalf("terminal: %+v", terminal)
	}
	raw, err := os.ReadFile(filepath.Join(root, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-job-marker") {
		t.Fatal("phase secret persisted")
	}
	reopened, err := OpenWebJobs(root, e, RedactWebText, nil)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err = reopened.Start(context.Background(), a, p.ID, "one")
	if err != nil || repeat.ID != j.ID || e.starts.Load() != 1 {
		t.Fatal("restart replay executed again")
	}
}

func TestWebJobsRevocationRetainsAndDoesNotCancelUnrelatedCredentials(t *testing.T) {
	for _, actorRevocation := range []bool{false, true} {
		t.Run(map[bool]string{false: "credential", true: "account"}[actorRevocation], func(t *testing.T) {
			e := newBlockingJobEngine()
			s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			a := DeploymentActor{ID: "alice", Role: "operator"}
			_, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy", Retention: "cleanup", CredentialBindings: map[string]string{"host": "owned-credential"}}, "one")
			select {
			case <-e.started:
			case <-time.After(3 * time.Second):
				t.Fatal("not started")
			}
			if err = s.RevokeCredential(a.ID, "unrelated"); err != nil {
				t.Fatal(err)
			}
			before, _ := s.Get(j.ID)
			if before.CancelReason != "" {
				t.Fatal("unrelated credential cancelled job")
			}
			if actorRevocation {
				err = s.RevokeActor(a.ID)
			} else {
				err = s.RevokeCredential(a.ID, "owned-credential")
			}
			if err != nil {
				t.Fatal(err)
			}
			v := waitJob(t, s, j.ID)
			if v.State != "cancelled" || v.Retention != "retain" || v.NodeDisposition != "retained" || e.cleanups.Load() != 0 {
				t.Fatalf("revocation cleaned: %+v", v)
			}
		})
	}
}

func TestWebJobsInterruptedOnRestartNeverResume(t *testing.T) {
	e := newBlockingJobEngine()
	original := t.TempDir()
	s, err := OpenWebJobs(original, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	p, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy"}, "one")
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("not started")
	}
	b, err := os.ReadFile(filepath.Join(original, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	restartedRoot := t.TempDir()
	if err = os.WriteFile(filepath.Join(restartedRoot, "jobs.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenWebJobs(restartedRoot, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := restarted.Get(j.ID)
	if err != nil || v.State != "interrupted" || len(v.UnresolvedResources) == 0 || e.starts.Load() != 1 {
		t.Fatalf("restart: %+v %v", v, err)
	}
	v, err = restarted.Start(context.Background(), a, p.ID, "one")
	if err != nil || v.State != "interrupted" || e.starts.Load() != 1 {
		t.Fatal("replay resumed interrupted execution")
	}
	_, _ = s.Cancel(a, j.ID)
	waitJob(t, s, j.ID)
}

func TestWebJobsStalePlanAndUnresolvedEngineFailClosed(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	p, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	e.fingerprint = strings.Repeat("b", 64)
	if _, err = s.Start(context.Background(), a, p.ID, "one"); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("stale fingerprint accepted")
	}
	if e.starts.Load() != 0 {
		t.Fatal("stale plan executed")
	}
	if err = validatePrepared(WebPreparedJob{Fingerprint: e.fingerprint, Payload: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("unresolved footprint accepted")
	}
}

func TestWebJobPlanDetachesCallerOwnedReferences(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	in := WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy", DocumentRefs: []DeploymentDocumentRef{{ID: "document", Revision: 1}}}
	p, err := s.Plan(context.Background(), DeploymentActor{ID: "alice", Role: "operator"}, in)
	if err != nil {
		t.Fatal(err)
	}
	in.DocumentRefs[0].Revision = 99
	p.DocumentRefs[0].Revision = 100
	s.mu.Lock()
	defer s.mu.Unlock()
	if stored := s.state.Plans[p.ID]; stored.Input.DocumentRefs[0].Revision != 1 || stored.Public.DocumentRefs[0].Revision != 1 {
		t.Fatal("accepted input can be changed through caller references")
	}
}

func TestWebPhysicalClaimsCompareAncestorsPortsAndHosts(t *testing.T) {
	cases := []struct {
		a, b WebResourceClaim
		want bool
	}{
		{WebResourceClaim{HostIdentity: "host", DataPath: "/nodes/a"}, WebResourceClaim{HostIdentity: "host", DataPath: "/nodes"}, true},
		{WebResourceClaim{HostIdentity: "host", DataPath: "/nodes/a"}, WebResourceClaim{HostIdentity: "host", DataPath: "/nodes/ab"}, false},
		{WebResourceClaim{HostIdentity: "host", Ports: []int{8545}}, WebResourceClaim{HostIdentity: "host", Ports: []int{8545}}, true},
		{WebResourceClaim{HostIdentity: "host", Ports: []int{8545}}, WebResourceClaim{HostIdentity: "other", Ports: []int{8545}}, false},
	}
	for _, c := range cases {
		if got := claimsOverlap(c.a, c.b); got != c.want {
			t.Errorf("%+v / %+v: %v", c.a, c.b, got)
		}
	}
}
