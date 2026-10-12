package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

// This engine tests HTTP disclosure and job metadata, not native deployment.
type conflictFixtureEngine struct{}

func (conflictFixtureEngine) Prepare(context.Context, app.DeploymentActor, app.WebPlanInput) (app.WebPreparedJob, error) {
	return app.WebPreparedJob{Fingerprint: strings.Repeat("a", 64), Claims: []app.WebResourceClaim{{HostIdentity: "physical-host", DataPath: "/owned/nodes", Ports: []int{8545}}}, Payload: json.RawMessage(`{"private":"never-return-this-payload"}`)}, nil
}
func (conflictFixtureEngine) Execute(context.Context, app.DeploymentActor, app.WebPreparedJob, func(app.WebJobPhase) error) (app.WebJobResult, error) {
	return app.WebJobResult{NodeDisposition: "retained"}, nil
}
func (conflictFixtureEngine) Cleanup(context.Context, app.DeploymentActor, app.WebPreparedJob) (app.WebJobResult, error) {
	return app.WebJobResult{NodeDisposition: "cleaned"}, nil
}

func TestWebPlanConflictHTTPShowsOwnerWithoutPrivateInputs(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), conflictFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	alice := app.DeploymentActor{ID: "alice", Role: "operator"}
	owner, err := jobs.Plan(context.Background(), alice, app.WebPlanInput{WorkspaceID: "owner", Operation: "chain.setup"})
	if err != nil {
		t.Fatal(err)
	}
	held, err := jobs.Start(context.Background(), alice, owner.ID, "one")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		job, _ := jobs.Get(held.ID)
		if job.State == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture job never completed")
		}
		time.Sleep(time.Millisecond)
	}
	bob := app.DeploymentActor{ID: "bob", Role: "operator"}
	alias, err := jobs.Plan(context.Background(), bob, app.WebPlanInput{WorkspaceID: "alias", Operation: "chain.setup"})
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithWebJobs(jobs, func(r *http.Request) (app.DeploymentActor, error) {
		id := r.Header.Get("fixture-actor")
		if id == "" {
			return app.DeploymentActor{}, errors.New("unauthenticated")
		}
		role := "operator"
		if id == "viewer" {
			role = "viewer"
		}
		return app.DeploymentActor{ID: id, Role: role}, nil
	}))
	call := func(actor, plan string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/v1/plans/"+plan+"/conflicts", nil)
		r.Header.Set("fixture-actor", actor)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		return w
	}
	w := call("bob", alias.ID)
	if w.Code != 200 {
		t.Fatalf("conflict review missing: %d %s", w.Code, w.Body.String())
	}
	var result struct {
		Items []struct {
			JobID, WorkspaceID, ActorID, State, NodeDisposition string
			Resources                                           []app.WebResourceClaim
		}
	}
	if err = json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Fatal(w.Body.String())
	}
	item := result.Items[0]
	if item.JobID != held.ID || item.WorkspaceID != "owner" || item.ActorID != "alice" || item.State != "succeeded" || item.NodeDisposition != "retained" || len(item.Resources) != 1 {
		t.Fatal(w.Body.String())
	}
	if strings.Contains(w.Body.String(), "never-return-this-payload") || strings.Contains(w.Body.String(), "credentialBindings") {
		t.Fatal("private plan disclosed")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("conflict review cacheable")
	}
	for _, tc := range []struct {
		actor, plan string
		status      int
	}{{"", alias.ID, 401}, {"alice", alias.ID, 404}, {"viewer", alias.ID, 403}, {"bob", "missing", 404}} {
		if response := call(tc.actor, tc.plan); response.Code != tc.status {
			t.Fatalf("%s %s: got %d want %d", tc.actor, tc.plan, response.Code, tc.status)
		}
	}
	if response := call("alice", owner.ID); response.Code != 200 || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatal("owner control falsely blocked", response.Body.String())
	}
}
