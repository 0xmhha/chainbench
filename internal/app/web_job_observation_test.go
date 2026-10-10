package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestJobObservationReplaysPublicChangesAndResyncsAfterRestart(t *testing.T) {
	root := t.TempDir()
	engine := newBlockingJobEngine()
	jobs, err := OpenWebJobs(root, engine, func(s string) string { return strings.ReplaceAll(s, "private-job-marker", "[REDACTED]") }, nil)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	changes, unsubscribe, err := jobs.SubscribeChanges(initial.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	actor := DeploymentActor{ID: "owner", Role: "operator"}
	_, accepted := startFixtureJob(t, jobs, actor, WebPlanInput{WorkspaceID: "owned", Operation: "chain.setup", Arguments: json.RawMessage(`{"secret":"private-plan-marker"}`)}, "observation-proof")
	select {
	case <-engine.started:
	case <-time.After(3 * time.Second):
		t.Fatal("executor not running")
	}
	current, err := jobs.Snapshot(context.Background())
	if err != nil || len(current.Jobs) != 1 || current.Jobs[0].State != "running" || current.Version <= initial.Version {
		t.Fatal("snapshot not current", current, err)
	}
	previous := initial.Version
	for previous < current.Version {
		select {
		case event := <-changes:
			if event.Version != previous+1 || event.Cursor == "" || event.Type == "resync_required" {
				t.Fatal("inconsistent committed change", event)
			}
			raw, _ := json.Marshal(event)
			if strings.Contains(string(raw), "private-plan-marker") || strings.Contains(string(raw), "private-job-marker") {
				t.Fatal("stream leaks private inputs", string(raw))
			}
			previous = event.Version
			if event.Job != nil {
				event.Job.State = "poisoned-client-copy"
			}
		case <-time.After(3 * time.Second):
			t.Fatal("committed update absent")
		}
	}
	replay, cancelReplay, err := jobs.SubscribeChanges(initial.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer cancelReplay()
	for v := initial.Version; v < current.Version; {
		select {
		case event := <-replay:
			v = event.Version
			if event.Job != nil && event.Job.State == "poisoned-client-copy" {
				t.Fatal("subscriber modified retained replay")
			}
		case <-time.After(3 * time.Second):
			t.Fatal("bounded replay absent")
		}
	}
	if _, err = jobs.Cancel(actor, accepted.ID); err != nil {
		t.Fatal(err)
	}
	terminal := waitJob(t, jobs, accepted.ID)
	saved, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWebJobs(root, engine, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := reopened.Snapshot(context.Background())
	if err != nil || after.Version != saved.Version || after.Cursor == saved.Cursor || after.Jobs[0].State != terminal.State || engine.starts.Load() != 1 {
		t.Fatal("restart lost current state or resumed execution", after, err)
	}
	stale, cancelStale, err := reopened.SubscribeChanges(saved.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer cancelStale()
	select {
	case event := <-stale:
		if event.Type != "resync_required" || event.Reason != "server_restarted" {
			t.Fatal("old instance silently accepted", event)
		}
	case <-time.After(time.Second):
		t.Fatal("restart resync absent")
	}
}

func TestJobObservationBoundsBacklogAndRefusesExpiredCursors(t *testing.T) {
	jobs, err := OpenWebJobs(t.TempDir(), newBlockingJobEngine(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	slow, cancel, err := jobs.SubscribeChanges(snapshot.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	for i := 0; i < 300; i++ {
		jobs.mu.Lock()
		err = jobs.commit("system", "", "observation-fixture", func(*webJobState) {})
		jobs.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(slow) > 128 {
		t.Fatal("subscriber backlog unbounded", len(slow))
	}
	_, _, drops := jobs.StreamPosition()
	if drops == 0 {
		t.Fatal("delivery loss hidden")
	}
	expired, release, err := jobs.SubscribeChanges(snapshot.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	select {
	case event := <-expired:
		if event.Type != "resync_required" || event.Reason != "cursor_expired" {
			t.Fatal("expired replay pretends complete", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expired resync absent")
	}
	for _, cursor := range []string{"", snapshot.Cursor + "\nsecret", strings.Split(snapshot.Cursor, ":")[0] + ":999999999"} {
		ch, stop, err := jobs.SubscribeChanges(cursor)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case event := <-ch:
			if event.Type != "resync_required" {
				t.Fatal("invalid cursor accepted", event)
			}
		case <-time.After(time.Second):
			t.Fatal("invalid cursor stalls")
		}
		stop()
	}
	cancelled, cancelOnce, err := jobs.SubscribeChanges("")
	if err != nil {
		t.Fatal(err)
	}
	cancelOnce()
	cancelOnce()
	for range cancelled {
	}
	if len(jobs.feed.subscribers) != 2 {
		t.Fatal("cancelled subscription retained", len(jobs.feed.subscribers))
	}
}

// A read may race with a committed job update; these are metadata-only fixtures.
type observationReadEngine struct {
	*blockingJobEngine
	read func(context.Context) ([]WebNetwork, error)
}

func (e *observationReadEngine) Networks(ctx context.Context) ([]WebNetwork, error) {
	return e.read(ctx)
}

func TestJobObservationSnapshotRetriesAndDetachesCachedNetworks(t *testing.T) {
	cached := []WebNetwork{{ID: "owned", Nodes: []WebNode{{ID: "node1", State: "unknown", PID: 99, SupportedControls: []string{"node.start"}}}}}
	engine := &observationReadEngine{blockingJobEngine: newBlockingJobEngine()}
	jobs, err := OpenWebJobs(t.TempDir(), engine, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	engine.read = func(context.Context) ([]WebNetwork, error) {
		reads++
		if reads == 1 {
			jobs.mu.Lock()
			err = jobs.commit("system", "", "snapshot-race-fixture", func(*webJobState) {})
			jobs.mu.Unlock()
			if err != nil {
				return nil, err
			}
		}
		return cached, nil
	}
	snapshot, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 1 || reads != 2 {
		t.Fatal("inconsistent snapshot escaped", snapshot.Version, reads)
	}
	if len(snapshot.Networks[0].Nodes[0].SupportedControls) != 0 {
		t.Fatal("cached controls presented as proof")
	}
	if len(cached[0].Nodes[0].SupportedControls) != 1 {
		t.Fatal("snapshot changed the reader's network metadata")
	}
	snapshot.Networks[0].Nodes[0].State = "poisoned"
	if cached[0].Nodes[0].State != "unknown" {
		t.Fatal("snapshot shares mutable network data")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = jobs.Snapshot(ctx); err != context.Canceled {
		t.Fatal("cancelled observation continued", err)
	}
}

func TestJobObservationFailedDurableWritePublishesNothing(t *testing.T) {
	root := t.TempDir()
	jobs, err := OpenWebJobs(root, newBlockingJobEngine(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := jobs.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	changes, stop, err := jobs.SubscribeChanges(snapshot.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	// Rename only this test's store, forcing the durable write to fail before commit.
	moved := root + "-temporarily-moved"
	if err = os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = os.Rename(moved, root); err != nil {
			t.Error(err)
		}
	}()
	jobs.mu.Lock()
	err = jobs.commit("system", "", "failed-write-fixture", func(*webJobState) {})
	jobs.mu.Unlock()
	if err == nil {
		t.Fatal("unavailable store accepted a write")
	}
	version, cursor, drops := jobs.StreamPosition()
	if version != snapshot.Version || cursor != snapshot.Cursor || drops != 0 || len(jobs.feed.journal) != 0 {
		t.Fatal("failed write published an observation", version, cursor, drops)
	}
	select {
	case event := <-changes:
		t.Fatal("failed durable write emitted update", event)
	default:
	}
}
