package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

type snapshotFrame struct{ ID, Type, Data string }

func readSnapshotFrame(reader *bufio.Reader) (snapshotFrame, error) {
	var frame snapshotFrame
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return frame, err
		}
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			return frame, nil
		}
		switch {
		case strings.HasPrefix(line, "id: "):
			frame.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			frame.Type = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			frame.Data = strings.TrimPrefix(line, "data: ")
		}
	}
}

func TestJobSnapshotStreamReplaysScopedChangesAndClosesOnRevocation(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), snapshotFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: "owner", Role: "operator"}
	for _, workspace := range []string{"owned", "another"} {
		plan, err := jobs.Plan(context.Background(), actor, app.WebPlanInput{WorkspaceID: workspace, Operation: "chain.setup", Arguments: json.RawMessage(`{"secret":"private-stream-payload"}`)})
		if err != nil {
			t.Fatal(err)
		}
		accepted, err := jobs.Start(context.Background(), actor, plan.ID, workspace)
		if err != nil {
			t.Fatal(err)
		}
		for deadline := time.Now().Add(3 * time.Second); ; {
			job, _ := jobs.Get(accepted.ID)
			if job.State == "succeeded" {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("fixture job timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	final, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var revoked atomic.Bool
	server := httptest.NewServer(NewServer(nil, nil, WithWebJobs(jobs, func(r *http.Request) (app.DeploymentActor, error) {
		if revoked.Load() || r.Header.Get("fixture-role") != "viewer" {
			return app.DeploymentActor{}, errors.New("revoked")
		}
		return app.DeploymentActor{ID: "reader", Role: "viewer"}, nil
	})))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/events?cursor=invalid&workspaceId=owned", nil)
	request.Header.Set("fixture-role", "viewer")
	request.Header.Set("Last-Event-ID", initial.Cursor) // Header wins over query.
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/event-stream" || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("viewer stream unavailable", response.Status)
	}
	reader := bufio.NewReader(response.Body)
	for version := initial.Version; version < final.Version; {
		frame, err := readSnapshotFrame(reader)
		if err != nil {
			t.Fatal(err)
		}
		var event app.WebJobChange
		if err = json.Unmarshal([]byte(frame.Data), &event); err != nil {
			t.Fatal(err)
		}
		if event.Version != version+1 || frame.ID != event.Cursor || frame.Type != event.Type || event.Type == "resync_required" {
			t.Fatal("header replay skipped a committed version", frame)
		}
		if event.Job != nil && event.Job.WorkspaceID != "owned" {
			t.Fatal("scoped stream leaks unrelated job", frame)
		}
		for _, private := range []string{"private-stream-payload", "never-return-this-payload", "prepared", "documentRefs"} {
			if strings.Contains(frame.Data, private) {
				t.Fatal("stream exposes private plan", frame)
			}
		}
		version = event.Version
	}
	heartbeat, err := readSnapshotFrame(reader)
	if err != nil {
		t.Fatal(err)
	}
	if heartbeat.Type != "observation" || heartbeat.ID != "" {
		t.Fatal("heartbeat advances replay cursor", heartbeat)
	}
	revoked.Store(true)
	if _, err = readSnapshotFrame(reader); err == nil {
		t.Fatal("revoked viewer kept receiving updates")
	}
}

func TestJobSnapshotStreamInvalidCursorRequiresResyncWithoutEcho(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), snapshotFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServer(nil, nil, WithWebJobs(jobs, func(*http.Request) (app.DeploymentActor, error) {
		return app.DeploymentActor{ID: "reader", Role: "viewer"}, nil
	})))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/events?cursor=private-invalid-cursor", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	frame, err := readSnapshotFrame(bufio.NewReader(response.Body))
	if err != nil {
		t.Fatal(err)
	}
	var event app.WebJobChange
	if err = json.Unmarshal([]byte(frame.Data), &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "resync_required" || event.Reason != "cursor_invalid" || frame.ID != event.Cursor || strings.Contains(frame.Data, "private-invalid-cursor") {
		t.Fatal("invalid input became replay position", frame)
	}
}
