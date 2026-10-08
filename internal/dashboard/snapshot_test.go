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

func TestJobSnapshotAuthenticatesViewersAndExcludesPrivatePlans(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), snapshotFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: "owner", Role: "operator"}
	plan, err := jobs.Plan(context.Background(), actor, app.WebPlanInput{WorkspaceID: "owned", Operation: "chain.setup", Arguments: json.RawMessage(`{"secret":"private-input-must-not-be-sent"}`)})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := jobs.Start(context.Background(), actor, plan.ID, "snapshot-proof")
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(3 * time.Second); ; {
		job, _ := jobs.Get(accepted.ID)
		if job.State == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture executor never completed")
		}
		time.Sleep(time.Millisecond)
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := NewServer(bus, nil, WithWebJobs(jobs, func(r *http.Request) (app.DeploymentActor, error) {
		if r.Header.Get("fixture-role") == "" {
			return app.DeploymentActor{}, errors.New("unauthenticated")
		}
		return app.DeploymentActor{ID: "reader", Role: r.Header.Get("fixture-role")}, nil
	}))
	call := func(role, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("fixture-role", role)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	if w := call("", "/api/v1/snapshot"); w.Code != 401 {
		t.Errorf("snapshot permits anonymous read: %d", w.Code)
	}
	w := call("viewer", "/api/v1/snapshot")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("viewer snapshot unavailable: %d %s", w.Code, w.Body.String())
	}
	var snapshot struct {
		Version    uint64
		Cursor     string
		Jobs       []app.WebJob
		Networks   []app.WebNetwork
		ObservedAt time.Time
	}
	if err = json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatal("snapshot is not JSON", err)
	}
	if snapshot.Version == 0 || snapshot.Cursor == "" || snapshot.ObservedAt.IsZero() || len(snapshot.Jobs) != 1 || snapshot.Jobs[0].ID != accepted.ID || snapshot.Jobs[0].State != "succeeded" || snapshot.Networks == nil {
		t.Fatal("current snapshot incomplete", w.Body.String())
	}
	for _, private := range []string{"private-input-must-not-be-sent", "never-return-this-payload", "prepared", "documentRefs"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatal("snapshot exposes private plans", private)
		}
	}
	if w = call("viewer", "/api/v1/snapshot?workspaceId=another"); w.Code != 200 {
		t.Fatal("workspace filter rejected", w.Code)
	}
	if err = json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil || len(snapshot.Jobs) != 0 {
		t.Fatal("snapshot filter ignored", w.Body.String(), err)
	}
	w = call("viewer", "/api/v1/snapshot?jobId="+accepted.ID)
	if err = json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil || len(snapshot.Networks) != 1 || snapshot.Networks[0].WorkspaceID != "owned" {
		t.Fatal("selected job snapshot includes unrelated networks", w.Body.String(), err)
	}
	if snapshot.Networks[0].Nodes[0].State != "unknown" || len(snapshot.Networks[0].Nodes[0].SupportedControls) != 0 {
		t.Fatal("cached PID presented as observed launch authority", w.Body.String())
	}
	if w = call("viewer", "/api/v1/snapshot?jobId=missing"); w.Code != 404 {
		t.Fatal("missing job presented as current", w.Code)
	}
	if w = call("viewer", "/api/v1/snapshot?path=/unapproved"); w.Code != 400 {
		t.Fatal("unknown scope accepted", w.Code)
	}
}

// Cached metadata deliberately includes another workspace and an unverified PID.
type snapshotFixtureEngine struct{ conflictFixtureEngine }

func (snapshotFixtureEngine) Networks(context.Context) ([]app.WebNetwork, error) {
	return []app.WebNetwork{
		{ID: "owned-network", WorkspaceID: "owned", Nodes: []app.WebNode{{ID: "node1", PID: 123, State: "unknown", SupportedControls: []string{"node.start"}}}},
		{ID: "unrelated-network", WorkspaceID: "another", Nodes: []app.WebNode{}},
	}, nil
}

func (snapshotFixtureEngine) Prepare(ctx context.Context, actor app.DeploymentActor, input app.WebPlanInput) (app.WebPreparedJob, error) {
	prepared, err := (conflictFixtureEngine{}).Prepare(ctx, actor, input)
	prepared.Claims[0].DataPath += "/" + input.WorkspaceID
	if input.WorkspaceID == "another" {
		prepared.Claims[0].Ports = []int{8546}
	}
	return prepared, err
}
