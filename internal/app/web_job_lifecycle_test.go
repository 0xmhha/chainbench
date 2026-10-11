package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// revalidatingJobEngine refuses at the check before running, as a plan whose
// target or binary changed after acceptance would be refused.
type revalidatingJobEngine struct {
	blockingJobEngine
	refuse error
}

func (e *revalidatingJobEngine) Revalidate(context.Context, DeploymentActor, WebPreparedJob) error {
	return e.refuse
}

func TestJobThatFailsItsPreRunCheckNeverRuns(t *testing.T) {
	e := &revalidatingJobEngine{blockingJobEngine: *newBlockingJobEngine(), refuse: ErrDeploymentConflict}
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, j := startFixtureJob(t, s, DeploymentActor{ID: "alice", Role: "operator"}, WebPlanInput{WorkspaceID: "w", Operation: "chain.deploy"}, "k")
	v := waitJob(t, s, j.ID)
	if v.State != "failed" || e.starts.Load() != 0 || v.Error["code"] == "" {
		t.Fatalf("a refused pre-run check ran or did not fail: %+v starts=%d", v, e.starts.Load())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.state.Audit {
		if a.JobID == j.ID && a.Operation == "job.running" {
			t.Fatal("the job passed through running")
		}
	}
}

// failingCleanupEngine runs until cancelled and then fails to clean up.
type failingCleanupEngine struct{ blockingJobEngine }

func (e *failingCleanupEngine) Cleanup(context.Context, DeploymentActor, WebPreparedJob) (WebJobResult, error) {
	return WebJobResult{NodeDisposition: "cleanup_failed", UnresolvedResources: []string{"/fixture/nodes"}}, errors.New("node data busy")
}

func TestCancelWhoseCleanupFailsEndsFailed(t *testing.T) {
	e := &failingCleanupEngine{*newBlockingJobEngine()}
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	_, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "w", Operation: "chain.deploy", Retention: "cleanup"}, "k")
	<-e.started
	if _, err = s.CancelWithMessage(a, j.ID, "stop the test early"); err != nil {
		t.Fatal(err)
	}
	v := waitJob(t, s, j.ID)
	if v.State != "failed" || v.NodeDisposition != "cleanup_failed" || v.CancelReason != "user" {
		t.Fatalf("%+v", v)
	}
	if v.CancelMessage != "stop the test early" {
		t.Errorf("cancel message %q", v.CancelMessage)
	}
}

// unknownAfterCancelEngine reports that it cannot tell what its nodes did.
type unknownAfterCancelEngine struct{ blockingJobEngine }

func (e *unknownAfterCancelEngine) Execute(ctx context.Context, a DeploymentActor, p WebPreparedJob, report func(WebJobPhase) error) (WebJobResult, error) {
	_, err := e.blockingJobEngine.Execute(ctx, a, p, report)
	return WebJobResult{NodeDisposition: "unknown", UnresolvedResources: []string{"/fixture/nodes"}}, err
}

func TestRevocationKeepsTheDispositionTheEngineReported(t *testing.T) {
	e := &unknownAfterCancelEngine{*newBlockingJobEngine()}
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	_, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "w", Operation: "chain.deploy", CredentialBindings: map[string]string{"host": "cred-1"}}, "k")
	<-e.started
	if err = s.RevokeCredential(a.ID, "cred-1"); err != nil {
		t.Fatal(err)
	}
	if v := waitJob(t, s, j.ID); v.State != "cancelled" || v.NodeDisposition != "unknown" {
		t.Fatalf("revocation replaced the reported disposition: %+v", v)
	}
}

func TestBusyRefusalNamesTheJobHoldingTheResources(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	alice, bob := DeploymentActor{ID: "alice", Role: "operator"}, DeploymentActor{ID: "bob", Role: "operator"}
	_, held := startFixtureJob(t, s, alice, WebPlanInput{WorkspaceID: "one", Operation: "chain.deploy"}, "k1")
	<-e.started
	plan, err := s.Plan(context.Background(), bob, WebPlanInput{WorkspaceID: "alias", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Start(context.Background(), bob, plan.ID, "k2")
	var busy *WebBusyError
	if !errors.As(err, &busy) || !errors.Is(err, ErrDeploymentConflict) || len(busy.Holders) != 1 || busy.Holders[0].JobID != held.ID || busy.Holders[0].ActorID != "alice" || len(busy.Holders[0].Resources) == 0 {
		t.Fatalf("busy refusal %v %+v", err, busy)
	}
	if strings.Contains(err.Error(), "revision") {
		t.Errorf("a busy refusal reads as a revision change: %v", err)
	}
	_, _ = s.Cancel(alice, held.ID)
	waitJob(t, s, held.ID)
}

func TestJobRecordsItsCredentialsAndRequest(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	plan, err := s.Plan(context.Background(), a, WebPlanInput{WorkspaceID: "w", Operation: "chain.deploy", CredentialBindings: map[string]string{"host": "cred-1"}, AccountBindings: map[string]string{"funder": "key-1"}})
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.Start(ContextWithRequestID(context.Background(), "req-123"), a, plan.ID, "k")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(j.CredentialRefs, ",") != "cred-1,key-1" || j.RequestID != "req-123" {
		t.Fatalf("job %+v", j)
	}
	<-e.started
	_, _ = s.Cancel(a, j.ID)
	v := waitJob(t, s, j.ID)
	if v.Error != nil && v.Error["requestId"] != "req-123" {
		t.Errorf("job error request id %v", v.Error)
	}
}
