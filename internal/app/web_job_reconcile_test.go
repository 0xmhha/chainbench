package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// reconcilingJobEngine reports the actual state a fixture decides.
type reconcilingJobEngine struct {
	blockingJobEngine
	outcome WebReconciliation
	err     error
}

func (e *reconcilingJobEngine) Reconcile(context.Context, DeploymentActor, WebPreparedJob) (WebReconciliation, error) {
	return e.outcome, e.err
}

func interruptedFixture(t *testing.T, e *reconcilingJobEngine) (*WebJobs, WebJob, DeploymentActor) {
	t.Helper()
	original := t.TempDir()
	s, err := OpenWebJobs(original, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	_, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy"}, "one")
	<-e.started
	b, err := os.ReadFile(filepath.Join(original, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Cancel(a, j.ID)
	waitJob(t, s, j.ID)
	root := t.TempDir()
	if err = os.WriteFile(filepath.Join(root, "jobs.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenWebJobs(root, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return restarted, j, a
}

func TestReconcilingAnInterruptedJobRecordsWhatIsActuallyThere(t *testing.T) {
	e := &reconcilingJobEngine{blockingJobEngine: *newBlockingJobEngine(), outcome: WebReconciliation{Outcome: "absent", Nodes: []WebNodeCheck{{NodeID: "node1", Process: "stopped", Data: "absent"}}}}
	s, j, a := interruptedFixture(t, e)
	if _, err := s.Reconcile(context.Background(), DeploymentActor{ID: "bob", Role: "operator"}, j.ID); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatalf("another operator reconciled the job: %v", err)
	}
	if holders := s.resourceConflicts("alias", []WebResourceClaim{{HostIdentity: "fixture-host", DataPath: "/fixture/nodes"}}); len(holders) != 1 {
		t.Fatalf("an unchecked interrupted job must keep its resources: %+v", holders)
	}
	v, err := s.Reconcile(context.Background(), a, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Reconciliation == nil || v.Reconciliation.Outcome != "absent" || v.NodeDisposition != "cleaned" || len(v.UnresolvedResources) != 0 || v.State != "interrupted" {
		t.Fatalf("absent nodes did not release the job: %+v", v)
	}
	if holders := s.resourceConflicts("alias", []WebResourceClaim{{HostIdentity: "fixture-host", DataPath: "/fixture/nodes"}}); len(holders) != 0 {
		t.Fatalf("resources confirmed absent are still held: %+v", holders)
	}
}

func TestReconcilingKeepsLiveOrUncheckedNodesLocked(t *testing.T) {
	present := &reconcilingJobEngine{blockingJobEngine: *newBlockingJobEngine(), outcome: WebReconciliation{Outcome: "present", Nodes: []WebNodeCheck{{NodeID: "node1", Process: "running", Data: "present"}}}}
	s, j, a := interruptedFixture(t, present)
	v, err := s.Reconcile(context.Background(), a, j.ID)
	if err != nil || v.NodeDisposition != "retained" || len(v.UnresolvedResources) != 0 {
		t.Fatalf("a live network is retained: %+v %v", v, err)
	}
	if holders := s.resourceConflicts("alias", []WebResourceClaim{{HostIdentity: "fixture-host", DataPath: "/fixture/nodes"}}); len(holders) != 1 {
		t.Fatal("a live network's resources were released")
	}
	unverified := &reconcilingJobEngine{blockingJobEngine: *newBlockingJobEngine(), outcome: WebReconciliation{Outcome: "unverified", Message: "target unreachable"}}
	s, j, a = interruptedFixture(t, unverified)
	v, err = s.Reconcile(context.Background(), a, j.ID)
	if err != nil || v.NodeDisposition != "unknown" || len(v.UnresolvedResources) == 0 || v.Reconciliation.CheckedAt.IsZero() {
		t.Fatalf("an unchecked network was resolved: %+v %v", v, err)
	}
	_ = time.Second
}

type localReconcilingEngine struct{ reconcilingJobEngine }

func (*localReconcilingEngine) ReconcilesLocally(WebPreparedJob) bool { return true }

func TestRestartChecksLocalInterruptedJobsOnItsOwn(t *testing.T) {
	e := &localReconcilingEngine{reconcilingJobEngine{blockingJobEngine: *newBlockingJobEngine(), outcome: WebReconciliation{Outcome: "present", Nodes: []WebNodeCheck{{NodeID: "node1", Process: "stopped", Data: "present"}}}}}
	original := t.TempDir()
	s, err := OpenWebJobs(original, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	_, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy"}, "one")
	<-e.started
	b, _ := os.ReadFile(filepath.Join(original, "jobs.json"))
	_, _ = s.Cancel(a, j.ID)
	waitJob(t, s, j.ID)
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "jobs.json"), b, 0o600)
	restarted, err := OpenWebJobs(root, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	restarted.ReconcileInterrupted(context.Background())
	v, _ := restarted.Get(j.ID)
	if v.Reconciliation == nil || v.Reconciliation.Nodes[0].NodeID != "node1" || v.NodeDisposition != "retained" {
		t.Fatalf("a local interrupted job was not checked after restart: %+v", v)
	}
}
