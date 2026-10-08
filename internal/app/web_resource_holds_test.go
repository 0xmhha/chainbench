package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func closeFixtureJobs(t *testing.T, s *WebJobs, a DeploymentActor) {
	t.Helper()
	t.Cleanup(func() {
		for _, job := range s.List() {
			if !terminalWebJob(job.State) {
				_, _ = s.Cancel(a, job.ID)
				waitJob(t, s, job.ID)
			}
		}
	})
}

func TestWebJobsRetainedClaimsBlockAnotherWorkspace(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, actor)
	_, owned := startFixtureJob(t, s, actor, WebPlanInput{WorkspaceID: "owner", Operation: "chain.deploy", Retention: "retain"}, "owned")
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("owner never entered executor")
	}
	if _, err = s.Cancel(actor, owned.ID); err != nil {
		t.Fatal(err)
	}
	completed := waitJob(t, s, owned.ID)
	if completed.NodeDisposition != "retained" {
		t.Fatal("fixture did not retain resources")
	}
	other, err := s.Plan(context.Background(), actor, WebPlanInput{WorkspaceID: "alias", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Start(context.Background(), actor, other.ID, "alias"); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("terminal job released retained resources to a physical alias: %v", err)
	}
	if e.starts.Load() != 1 {
		t.Fatal("conflicting alias reached executor")
	}
}

func TestWebJobsRetainedClaimsPermitOwnerControlsAndReleaseAfterCleanup(t *testing.T) {
	e := newBlockingJobEngine()
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, actor)
	_, owned := startFixtureJob(t, s, actor, WebPlanInput{WorkspaceID: "owner", Operation: "chain.deploy", Retention: "retain"}, "owned")
	<-e.started
	_, _ = s.Cancel(actor, owned.ID)
	waitJob(t, s, owned.ID)
	_, control := startFixtureJob(t, s, actor, WebPlanInput{WorkspaceID: "owner", Operation: "node.stop", Retention: "retain"}, "control")
	<-e.started
	_, _ = s.Cancel(actor, control.ID)
	waitJob(t, s, control.ID)
	_, cleanup := startFixtureJob(t, s, actor, WebPlanInput{WorkspaceID: "owner", Operation: "chain.deploy", Retention: "cleanup"}, "cleanup")
	<-e.started
	_, _ = s.Cancel(actor, cleanup.ID)
	if completed := waitJob(t, s, cleanup.ID); completed.NodeDisposition != "cleaned" {
		t.Fatal("cleanup did not complete")
	}
	_, alias := startFixtureJob(t, s, actor, WebPlanInput{WorkspaceID: "alias", Operation: "chain.deploy", Retention: "retain"}, "alias")
	<-e.started
	_, _ = s.Cancel(actor, alias.ID)
	waitJob(t, s, alias.ID)
}

func TestWebJobsInterruptedClaimsSurviveRestartForOtherWorkspaces(t *testing.T) {
	e := newBlockingJobEngine()
	root := t.TempDir()
	s, err := OpenWebJobs(root, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	closeFixtureJobs(t, s, actor)
	_, owned := startFixtureJob(t, s, actor, WebPlanInput{WorkspaceID: "owner", Operation: "chain.deploy"}, "owned")
	<-e.started
	// Copy the durable snapshot to model restart without two writers sharing
	// a store. This tests persistence, not actual node reconciliation.
	snapshot, err := os.ReadFile(filepath.Join(root, "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	restartRoot := t.TempDir()
	if err = os.WriteFile(filepath.Join(restartRoot, "jobs.json"), snapshot, 0o600); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWebJobs(restartRoot, e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	closeFixtureJobs(t, reopened, actor)
	previous, err := reopened.Get(owned.ID)
	if err != nil || previous.State != "interrupted" {
		t.Fatal(previous, err)
	}
	alias, err := reopened.Plan(context.Background(), actor, WebPlanInput{WorkspaceID: "alias", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.Start(context.Background(), actor, alias.ID, "alias"); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("restart released unresolved physical resources: %v", err)
	}
}

// These snapshots exercise release boundaries independently of executable nodes.
func TestWebResourceHoldReleaseBoundaries(t *testing.T) {
	original := WebResourceClaim{HostIdentity: "host", DataPath: "/net/a", Ports: []int{8545, 8546}}
	for _, tc := range []struct {
		name         string
		disposition  string
		cleanup      WebResourceClaim
		unresolved   []string
		cleanupFirst bool
		wantConflict bool
	}{
		{name: "same canonical root", disposition: "cleaned", cleanup: WebResourceClaim{HostIdentity: "host", DataPath: "/net/a/.", Ports: []int{8545}}, wantConflict: false},
		{name: "different target sharing port", disposition: "cleaned", cleanup: WebResourceClaim{HostIdentity: "host", DataPath: "/net/b", Ports: []int{8545}}, wantConflict: true},
		{name: "ancestor is not cleanup proof", disposition: "cleaned", cleanup: WebResourceClaim{HostIdentity: "host", DataPath: "/net"}, wantConflict: true},
		{name: "different host", disposition: "cleaned", cleanup: WebResourceClaim{HostIdentity: "other", DataPath: "/net/a"}, wantConflict: true},
		{name: "failed cleanup", disposition: "cleanup_failed", cleanup: original, wantConflict: true},
		{name: "unresolved cleanup", disposition: "cleaned", cleanup: original, unresolved: []string{"node unresolved"}, wantConflict: true},
		{name: "cleanup predates retained job", disposition: "cleaned", cleanup: original, cleanupFirst: true, wantConflict: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &WebJobs{state: webJobState{Jobs: map[string]WebJob{}, Plans: map[string]savedWebPlan{}}}
			s.state.Jobs["owned"] = WebJob{ID: "owned", WorkspaceID: "owner", PlanID: "owned-plan", State: "cancelled", NodeDisposition: "retained"}
			s.state.Jobs["cleanup"] = WebJob{ID: "cleanup", WorkspaceID: "owner", PlanID: "cleanup-plan", State: "succeeded", NodeDisposition: tc.disposition, UnresolvedResources: tc.unresolved}
			s.state.Plans["owned-plan"] = savedWebPlan{Prepared: WebPreparedJob{Claims: []WebResourceClaim{original}}}
			s.state.Plans["cleanup-plan"] = savedWebPlan{Prepared: WebPreparedJob{Claims: []WebResourceClaim{tc.cleanup}}}
			s.state.Audit = []webJobAudit{{JobID: "owned", Operation: "job.accepted"}, {JobID: "cleanup", Operation: "job.accepted"}}
			if tc.cleanupFirst {
				s.state.Audit[0], s.state.Audit[1] = s.state.Audit[1], s.state.Audit[0]
			}
			got := s.resourceConflict("alias", []WebResourceClaim{original})
			if got != tc.wantConflict {
				t.Fatalf("resource conflict = %v; want %v", got, tc.wantConflict)
			}
		})
	}
}

func TestWebResourceCleanupPortOnlyAndMissingOrder(t *testing.T) {
	held := WebResourceClaim{HostIdentity: "host", Ports: []int{8545, 8546}}
	if claimCleanupCovers(held, WebResourceClaim{HostIdentity: "host", Ports: []int{8545}}) {
		t.Fatal("partial port cleanup released all ports")
	}
	if !claimCleanupCovers(held, WebResourceClaim{HostIdentity: "host", Ports: []int{8546, 8545, 8547}}) {
		t.Fatal("complete port cleanup not recognized")
	}
	s := &WebJobs{state: webJobState{Jobs: map[string]WebJob{
		"owned":   {ID: "owned", WorkspaceID: "owner", PlanID: "p1", State: "cancelled", NodeDisposition: "retained"},
		"cleanup": {ID: "cleanup", WorkspaceID: "owner", PlanID: "p2", State: "succeeded", NodeDisposition: "cleaned"},
	}, Plans: map[string]savedWebPlan{"p1": {Prepared: WebPreparedJob{Claims: []WebResourceClaim{held}}}, "p2": {Prepared: WebPreparedJob{Claims: []WebResourceClaim{held}}}}}}
	if !s.resourceConflict("alias", []WebResourceClaim{held}) {
		t.Fatal("missing durable order treated as cleanup proof")
	}
	s.state.Jobs["owned"] = WebJob{ID: "owned", WorkspaceID: "owner", PlanID: "p1", State: "running"}
	if !s.resourceConflict("owner", []WebResourceClaim{held}) {
		t.Fatal("owner started overlapping active work")
	}
	if s.resourceConflict("alias", []WebResourceClaim{{HostIdentity: "other", Ports: held.Ports}}) {
		t.Fatal("different physical hosts blocked")
	}
	if s.resourceConflict("alias", []WebResourceClaim{{HostIdentity: "host", Ports: []int{18545}}}) {
		t.Fatal("independent ports blocked")
	}
}
