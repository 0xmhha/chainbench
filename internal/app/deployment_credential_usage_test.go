package app

import (
	"testing"
	"time"
)

// Settings shows where each personal credential is bound and which unfinished
// jobs still hold it, so revoking it is a visible decision.
func TestCredentialUsageNamesOwnBindingsAndActiveJobs(t *testing.T) {
	s, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	alice := DeploymentActor{ID: "alice", Role: "operator"}
	bob := DeploymentActor{ID: "bob", Role: "operator"}
	c, err := s.SaveCredential(alice, DeploymentCredentialInput{Label: "login", Kind: "password", SSHUser: "ssh", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.SaveCredential(bob, DeploymentCredentialInput{Label: "login", Kind: "password", SSHUser: "ssh", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	set, _ := s.SaveDocument(alice, "", 0, deploymentTestSet())
	config, _ := s.SaveDocument(alice, "", 0, deploymentTestConfig())
	w, err := s.SaveWorkspace(alice, "", 0, DeploymentWorkspaceInput{Name: "Team", Documents: []DeploymentDocumentRef{{set.ID, set.Revision}, {config.ID, config.Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(alice, w.ID, "ssh", c.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(bob, w.ID, "ssh", other.ID); err != nil {
		t.Fatal(err)
	}
	bindings := s.CredentialBindings(alice)
	if len(bindings) != 1 || len(bindings[c.ID]) != 1 || bindings[c.ID][0] != (DeploymentCredentialBinding{WorkspaceID: w.ID, ServerRef: "ssh"}) {
		t.Fatalf("bindings = %+v", bindings)
	}

	e := newBlockingJobEngine()
	jobs, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, j := startFixtureJob(t, jobs, alice, WebPlanInput{WorkspaceID: w.ID, Operation: "chain.deploy", CredentialBindings: map[string]string{"ssh": c.ID}}, "one")
	select {
	case <-e.started:
	case <-time.After(3 * time.Second):
		t.Fatal("not started")
	}
	active := jobs.ActiveCredentialJobs(alice.ID)
	if len(active[c.ID]) != 1 || active[c.ID][0] != j.ID {
		t.Fatalf("active = %+v", active)
	}
	if len(jobs.ActiveCredentialJobs(bob.ID)) != 0 {
		t.Fatal("another user's job was listed")
	}
	if err = jobs.RevokeCredential(alice.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	waitJob(t, jobs, j.ID)
	if len(jobs.ActiveCredentialJobs(alice.ID)) != 0 {
		t.Fatal("a finished job was still listed as using the credential")
	}
}
