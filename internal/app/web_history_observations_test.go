package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// historyObservationFixture archives four collection rounds for one network and
// records three jobs on it: two finished runs whose windows overlap, and one
// still running.
func historyObservationFixture(t *testing.T) (*monitorFixture, *WebMonitor, *WebHistory, *WebJobs, time.Time) {
	t.Helper()
	f := newMonitorFixture(t)
	if err := os.WriteFile(f.logPath, []byte("first line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	start := f.clock.UTC()
	for i := 0; i < 4; i++ { // rounds at start+0s, +5s, +10s, +15s
		if i == 2 {
			if err := os.WriteFile(f.logPath, []byte("first line\nthird round line\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if err := m.CollectOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		f.clock = f.clock.Add(5 * time.Second)
	}
	jobs, err := OpenWebJobs(f.root, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	finished := func(id string, from, to time.Duration) {
		jobs.state.Plans[id] = savedWebPlan{Prepared: WebPreparedJob{Fingerprint: strings.Repeat("c", 64)}}
		jobs.state.Jobs[id] = WebJob{ID: id, ActorID: "operator", WorkspaceID: f.network, PlanID: id, State: "succeeded", CreatedAt: start.Add(from),
			Phases: []WebJobPhase{{Name: "test.run", State: "succeeded", StartedAt: start.Add(from), FinishedAt: start.Add(to)}}, RunIDs: []string{}}
	}
	finished("early", -time.Second, 6*time.Second)    // owns rounds 0 and 5s
	finished("overlap", 4*time.Second, 7*time.Second) // shares the 5s round
	jobs.state.Plans["live"] = savedWebPlan{Prepared: WebPreparedJob{Fingerprint: strings.Repeat("c", 64)}}
	jobs.state.Jobs["live"] = WebJob{ID: "live", ActorID: "operator", WorkspaceID: f.network, PlanID: "live", State: "running", CreatedAt: start.Add(14 * time.Second), Phases: []WebJobPhase{}, RunIDs: []string{}}
	history, err := OpenWebHistory(f.root, "", jobs, nil)
	if err != nil {
		t.Fatal(err)
	}
	history.UseObservations(m)
	if err = history.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	// The engine session of the early job is a second record of the same
	// execution; it must not protect that execution's own observations.
	history.state.Captures["session-early"] = webHistoryCapture{Run: WebRun{ID: "session-early", JobID: "early", WorkspaceID: f.network, State: "succeeded", Summary: map[string]any{}}, Files: map[string]string{}}
	return f, m, history, jobs, start
}

func blockHeights(t *testing.T, m *WebMonitor, network string, q WebMonitorQuery) []time.Time {
	t.Helper()
	got, err := m.Metrics(DeploymentActor{ID: "viewer", Role: "viewer"}, network, q)
	if err != nil {
		t.Fatal(err)
	}
	var times []time.Time
	for _, s := range got.Series {
		if s.NodeID == "node1" && s.Name == "block_height" {
			for _, p := range s.Samples {
				times = append(times, p.Time)
			}
		}
	}
	return times
}

func TestWebHistoryLinksAndDeletesOnlyRunOwnedObservations(t *testing.T) {
	f, m, history, _, start := historyObservationFixture(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	admin := DeploymentActor{ID: "admin", Role: "admin"}
	run, err := history.Get("job-early")
	if err != nil {
		t.Fatal(err)
	}
	window, ok := run.Summary["observations"].(map[string]any)
	if !ok || window["networkId"] != f.network {
		t.Fatalf("run lacks its observation window: %+v", run.Summary)
	}
	for _, missing := range run.Summary["missingDimensions"].([]string) {
		if missing == "metrics" || missing == "logs" {
			t.Fatalf("archived dimensions still reported missing: %v", run.Summary["missingDimensions"])
		}
	}
	if got := blockHeights(t, m, f.network, WebMonitorQuery{RunID: "job-early"}); len(got) != 2 {
		t.Fatalf("run window samples = %v", got)
	}
	if _, err = m.Metrics(viewer, f.network, WebMonitorQuery{RunID: "job-early", From: start}); !errors.Is(err, ErrWebMonitorQuery) {
		t.Fatalf("run and explicit window together accepted: %v", err)
	}
	if _, err = m.Logs(viewer, "other.node1", WebMonitorQuery{RunID: "job-early"}); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("run window applied to another network: %v", err)
	}

	if err = history.Delete(admin, "job-early"); err != nil {
		t.Fatal(err)
	}
	all := blockHeights(t, m, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)})
	if len(all) != 3 || !all[0].Equal(start.Add(5*time.Second)) {
		t.Fatalf("deletion removed shared or other samples, or kept owned ones: %v", all)
	}
	metrics, _ := m.Metrics(viewer, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)})
	if !strings.Contains(strings.Join(gapReasons(metrics.Coverage), ","), "archive deleted_by_admin job-early") {
		t.Fatalf("deleted interval not distinguishable from missing collection: %v", gapReasons(metrics.Coverage))
	}
	logs, _ := m.Logs(viewer, f.network+".node1", WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)})
	if len(logs.Entries) != 1 || logs.Entries[0].Text != "third round line" {
		t.Fatalf("log deletion = %+v", logs.Entries)
	}
	// The early execution is still listed through its engine session, so the
	// shared 5s round stays; the running job's interval stays as well.
	if err = history.Delete(admin, "job-overlap"); err != nil {
		t.Fatal(err)
	}
	if got := blockHeights(t, m, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)}); len(got) != 3 {
		t.Fatalf("second deletion = %v", got)
	}
	if err = history.Delete(admin, "session-early"); err != nil {
		t.Fatal(err)
	}
	if got := blockHeights(t, m, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)}); len(got) != 2 || !got[0].Equal(start.Add(10*time.Second)) {
		t.Fatalf("last record of an execution must release its interval: %v", got)
	}
}

func TestWebHistoryDeletionCoversRemovedNodesAndUnboundedJobs(t *testing.T) {
	f, m, history, jobs, start := historyObservationFixture(t)
	admin := DeploymentActor{ID: "admin", Role: "admin"}
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	// A node removed from the chain record still has an archived log segment.
	old := webMonitorDay("logs/node9", start) + ".jsonl"
	line := `{"t":"` + start.Format(time.RFC3339Nano) + `","c":"` + start.Format(time.RFC3339Nano) + `","ts":"collection","x":"removed node"}` + "\n"
	if err := m.store.Append(f.network, old, []byte(line)); err != nil {
		t.Fatal(err)
	}
	if err := history.Delete(admin, "job-early"); err != nil {
		t.Fatal(err)
	}
	r, err := m.store.Open(f.network, old)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 64)
	n, _ := r.Read(b)
	_ = r.Close()
	if n != 0 {
		t.Fatalf("removed node's run-owned line kept: %q", b[:n])
	}
	// A terminal job without a recorded end has no bounded window.
	jobs.state.Jobs["unbounded"] = WebJob{ID: "unbounded", PlanID: "early", WorkspaceID: f.network, State: "failed", CreatedAt: start, Phases: []WebJobPhase{}, RunIDs: []string{}}
	history.state.Captures["job-unbounded"] = webHistoryCapture{Run: WebRun{ID: "job-unbounded", JobID: "unbounded", WorkspaceID: f.network, State: "failed", Summary: map[string]any{"missingDimensions": []any{"metrics", "logs"}}}, Files: map[string]string{}}
	run, err := history.Get("job-unbounded")
	if err != nil || run.Summary["observations"] != nil {
		t.Fatalf("unbounded run shows an archive window: %+v %v", run.Summary, err)
	}
	if err = history.Delete(admin, "job-unbounded"); err != nil {
		t.Fatal(err)
	}
	metrics, _ := m.Metrics(viewer, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)})
	for _, g := range metrics.Coverage.Gaps {
		if strings.Contains(g.Reason, "job-unbounded") {
			t.Fatalf("unbounded run recorded a deletion: %+v", g)
		}
	}
}

func TestWebHistoryObservationDeletionFailureKeepsRunAndRetries(t *testing.T) {
	f, m, history, _, start := historyObservationFixture(t)
	admin := DeploymentActor{ID: "admin", Role: "admin"}
	segment := filepath.Join(f.root, "observations", f.network)
	if err := os.Chmod(segment, 0o500); err != nil {
		t.Fatal(err)
	}
	err := history.Delete(admin, "job-early")
	_ = os.Chmod(segment, 0o700)
	if err == nil || !errors.Is(err, ErrHistoryObservationsIncomplete) {
		t.Fatalf("partial deletion reported as %v", err)
	}
	if _, err = history.Get("job-early"); err != nil {
		t.Fatalf("run removed although its observations were not: %v", err)
	}
	if last := history.state.Audit[len(history.state.Audit)-1]; last.Operation != "history.delete_failed" || last.RunID != "job-early" {
		t.Fatalf("failed deletion not audited: %+v", last)
	}
	if err = history.Delete(admin, "job-early"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if _, err = history.Get("job-early"); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatal("retried deletion kept the run")
	}
	if got := blockHeights(t, m, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: start.Add(time.Minute)}); len(got) != 3 {
		t.Fatalf("retry result = %v", got)
	}
}
