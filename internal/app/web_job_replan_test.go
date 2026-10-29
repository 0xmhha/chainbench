package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// This executor ends at once; it tests replanning bookkeeping, not a chain.
type finishingJobEngine struct{ blockingJobEngine }

func (e *finishingJobEngine) Execute(context.Context, DeploymentActor, WebPreparedJob, func(WebJobPhase) error) (WebJobResult, error) {
	return WebJobResult{NodeDisposition: "retained"}, nil
}

func TestWebJobReplanReusesTheRecordedInputForItsOwner(t *testing.T) {
	e := &finishingJobEngine{*newBlockingJobEngine()}
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	alice := DeploymentActor{ID: "alice", Role: "operator"}
	in := WebPlanInput{
		WorkspaceID:        "workspace",
		Operation:          "test.run",
		DocumentRefs:       []DeploymentDocumentRef{{ID: "deployment", Revision: 1}},
		CredentialBindings: map[string]string{"server1": "credential-a"},
		Arguments:          json.RawMessage(`{"caseRefs":[{"id":"case","revision":2}]}`),
		Retention:          "retain",
	}
	first, job := startFixtureJob(t, s, alice, in, "first")
	waitJob(t, s, job.ID)

	again, err := s.Replan(context.Background(), alice, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID == first.ID || again.WorkspaceID != in.WorkspaceID || again.Operation != in.Operation || !reflect.DeepEqual(again.DocumentRefs, in.DocumentRefs) {
		t.Fatalf("replan is not a fresh plan of the same operation: %+v", again)
	}
	s.mu.Lock()
	stored := s.state.Plans[again.ID].Input
	s.mu.Unlock()
	// The pinned revisions and bindings are the original ones, so it reruns
	// the same experiment rather than whatever changed since.
	if !bytes.Equal(stored.Arguments, in.Arguments) || !reflect.DeepEqual(stored.CredentialBindings, in.CredentialBindings) || stored.Retention != in.Retention {
		t.Fatalf("replan changed the recorded input: %+v", stored)
	}
}

func TestWebJobReplanRefusesOthersUnknownAndRunningJobs(t *testing.T) {
	e := &finishingJobEngine{*newBlockingJobEngine()}
	s, err := OpenWebJobs(t.TempDir(), e, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	alice := DeploymentActor{ID: "alice", Role: "operator"}
	_, job := startFixtureJob(t, s, alice, WebPlanInput{WorkspaceID: "workspace", Operation: "test.run"}, "first")
	waitJob(t, s, job.ID)
	// The bindings name the owner's own credentials, so nobody else may reuse them.
	for _, a := range []DeploymentActor{{ID: "bob", Role: "operator"}, {ID: "admin", Role: "administrator"}, {ID: "alice", Role: "viewer"}} {
		if _, err := s.Replan(context.Background(), a, job.ID); !errors.Is(err, ErrDeploymentForbidden) {
			t.Fatalf("%s replanned another's job: %v", a.ID, err)
		}
	}
	if _, err := s.Replan(context.Background(), alice, "missing"); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("unknown job: %v", err)
	}

	running, err := OpenWebJobs(t.TempDir(), newBlockingJobEngine(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, live := startFixtureJob(t, running, alice, WebPlanInput{WorkspaceID: "workspace", Operation: "test.run"}, "live")
	if _, err := running.Replan(context.Background(), alice, live.ID); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("a running job was replanned: %v", err)
	}
}
