package app

import (
	"context"
	"testing"
	"time"
)

// unresolvedOnCancelEngine leaves its composition unresolved when cancelled,
// as a native run does, and its cleanup then removes all of it.
type unresolvedOnCancelEngine struct{ *blockingJobEngine }

func (e unresolvedOnCancelEngine) Execute(ctx context.Context, a DeploymentActor, p WebPreparedJob, report func(WebJobPhase) error) (WebJobResult, error) {
	r, err := e.blockingJobEngine.Execute(ctx, a, p, report)
	r.UnresolvedResources = []string{"/fixture/nodes", "/fixture/record"}
	return r, err
}

// A successful cleanup removes what the cancelled run left, so the job must
// not keep naming those resources as unresolved.
func TestCleanedJobDoesNotKeepResourcesItsCleanupRemoved(t *testing.T) {
	e := unresolvedOnCancelEngine{newBlockingJobEngine()}
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	_, j := startFixtureJob(t, s, a, WebPlanInput{WorkspaceID: "workspace", Operation: "chain.deploy", Retention: "cleanup"}, "one")
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("not started")
	}
	if _, err = s.CancelWithMessage(a, j.ID, ""); err != nil {
		t.Fatal(err)
	}
	v := waitJob(t, s, j.ID)
	if v.NodeDisposition != "cleaned" || len(v.UnresolvedResources) != 0 {
		t.Fatalf("cleaned job still lists %v (%s)", v.UnresolvedResources, v.NodeDisposition)
	}
}
