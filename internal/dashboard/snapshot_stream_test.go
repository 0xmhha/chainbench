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
	"github.com/0xmhha/chainbench/internal/core/collector"
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

// fakeObservations is a live observation source a test pushes events into.
type fakeObservations struct{ events chan app.WebObservationEvent }

func (f fakeObservations) SubscribeObservations() (<-chan app.WebObservationEvent, func()) {
	return f.events, func() {}
}
func (fakeObservations) ObservationDrops() uint64 { return 7 }

// TestJobStreamCarriesScopedObservationEventsWithoutMovingTheCursor: the one
// events stream also carries what the collector saw, scoped to the asked
// workspace, and those frames carry no id so a reconnect replays jobs exactly.
func TestJobStreamCarriesScopedObservationEventsWithoutMovingTheCursor(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), snapshotFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	source := fakeObservations{events: make(chan app.WebObservationEvent, 4)}
	viewer := func(*http.Request) (app.DeploymentActor, error) {
		return app.DeploymentActor{ID: "reader", Role: "viewer"}, nil
	}
	server := httptest.NewServer(NewServer(nil, nil, WithWebJobs(jobs, viewer), withObservations(source)))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/events?workspaceId=owned&cursor="+initial.Cursor, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	at := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	source.events <- app.WebObservationEvent{Type: "metrics", NetworkID: "another", ObservedAt: at, Samples: []app.WebMetricPoint{{NodeID: "node1", Name: "block_height", Metric: "block_height", Canonical: true, Source: "rpc", Time: at, Value: 1}}}
	source.events <- app.WebObservationEvent{Type: "node", NetworkID: "owned", ObservedAt: at, Node: &app.WebNodeReading{ID: "node4", State: "stopped", ObservationReason: "no_matching_process", ObservedAt: at}}
	reader := bufio.NewReader(response.Body)
	for {
		frame, err := readSnapshotFrame(reader)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Type == "observation" {
			var heartbeat map[string]any
			_ = json.Unmarshal([]byte(frame.Data), &heartbeat)
			if heartbeat["observationDropsTotal"] != float64(7) {
				t.Fatalf("heartbeat does not count dropped observations: %s", frame.Data)
			}
			continue
		}
		if strings.Contains(frame.Data, `"another"`) {
			t.Fatal("scoped stream leaks another workspace's observations", frame)
		}
		if frame.Type != "node" {
			t.Fatalf("frame = %+v", frame)
		}
		if frame.ID != "" {
			t.Fatal("an observation frame moved the replay cursor", frame)
		}
		var event app.WebObservationEvent
		if err = json.Unmarshal([]byte(frame.Data), &event); err != nil || event.Node.ID != "node4" || event.Node.State != "stopped" {
			t.Fatalf("node event = %s (%v)", frame.Data, err)
		}
		return
	}
}

// TestJobStreamCarriesEngineEventsOnlyWhenUnscoped: what a CLI run publishes
// to the bus reaches the Web through the one events stream, as engine frames
// without an id. A workspace- or job-scoped stream does not carry them: bus
// events name no workspace, so scoping cannot be checked.
func TestJobStreamCarriesEngineEventsOnlyWhenUnscoped(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), snapshotFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	viewer := func(*http.Request) (app.DeploymentActor, error) {
		return app.DeploymentActor{ID: "reader", Role: "viewer"}, nil
	}
	server := httptest.NewServer(NewServer(bus, nil, WithWebJobs(jobs, viewer)))
	defer server.Close()
	open := func(query string) *bufio.Reader {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)
		request, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/events?cursor="+initial.Cursor+query, nil)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { response.Body.Close() })
		return bufio.NewReader(response.Body)
	}
	scoped, unscoped := open("&workspaceId=owned"), open("")
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				bus.Publish(collector.Event{Phase: collector.PhaseTest, Kind: collector.KindResult, Message: "engine-says-hello"})
			}
		}
	}()
	for {
		frame, err := readSnapshotFrame(unscoped)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Type == "observation" {
			if !strings.Contains(frame.Data, "busDeliveryDropsTotal") {
				t.Fatalf("heartbeat lacks the bus drop count: %s", frame.Data)
			}
			continue
		}
		if frame.Type != "engine" || frame.ID != "" || !strings.Contains(frame.Data, "engine-says-hello") {
			t.Fatalf("engine frame = %+v", frame)
		}
		break
	}
	for i := 0; i < 2; i++ {
		frame, err := readSnapshotFrame(scoped)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Type == "engine" {
			t.Fatal("a scoped stream carried an engine event", frame)
		}
	}
}

// TestWebModeLeavesTheLegacyStreamToLegacyObservation: the old bus stream and
// the run index exist only in -legacy-observation mode.
func TestWebModeLeavesTheLegacyStreamToLegacyObservation(t *testing.T) {
	bus := collector.NewBus()
	defer bus.Close()
	web := httptest.NewServer(NewServer(bus, collector.NewMemStore()))
	defer web.Close()
	for _, path := range []string{"/events", "/api/runs"} {
		response, err := http.Get(web.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode == 200 && (strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") || strings.HasPrefix(response.Header.Get("Content-Type"), "application/json")) {
			t.Fatalf("Web mode still serves %s (%s)", path, response.Header.Get("Content-Type"))
		}
	}
	legacy := httptest.NewServer(NewServer(bus, collector.NewMemStore(), WithLegacyObservation()))
	defer legacy.Close()
	response, err := http.Get(legacy.URL + "/api/runs")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("legacy run index = %d", response.StatusCode)
	}
}
