package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestWebRemoteLogsCollectOnlyWhileAttached(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] remote one\nINFO [10-09|00:46:59.000] remote 'two' $HOME\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	opened := 0
	open := func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		opened++
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	// Without a collection job only the absence is recorded.
	monitorCollect(t, f, m, 1)
	detach := m.attachRemote(f.network)
	if err := m.CollectRemoteLogs(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	monitorCollect(t, f, m, 1) // The background round no longer reports node1 as unavailable.
	logs, err := m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Entries) != 2 || logs.Entries[1].Text != "INFO [10-09|00:46:59.000] remote 'two' $HOME" || opened != 2 {
		t.Fatalf("remote lines = %+v opened=%d", logs.Entries, opened)
	}
	// node2's recorded log does not exist on the remote host.
	if reasons := strings.Join(gapReasons(logs.Coverage), ","); strings.Count(reasons, "node1 logs remote_log_collection_unavailable") != 1 {
		t.Fatalf("gap while no job ran must stay, once: %s", reasons)
	}
	other, _ := m.Logs(context.Background(), viewer, f.network+".node2", WebMonitorQuery{})
	if !strings.Contains(strings.Join(gapReasons(other.Coverage), ","), "node2 logs log_unavailable") {
		t.Fatalf("missing remote file not reported: %v", gapReasons(other.Coverage))
	}
	detach()
	f.clock = f.clock.Add(5e9)
	monitorCollect(t, f, m, 1)
	logs, _ = m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
	if strings.Count(strings.Join(gapReasons(logs.Coverage), ","), "remote_log_collection_unavailable") != 2 {
		t.Fatalf("detached collection must be reported again: %v", gapReasons(logs.Coverage))
	}
	// A cancelled (revoked) collector archives nothing.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] remote one\nINFO [10-09|00:46:59.000] remote 'two' $HOME\nafter revocation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = m.CollectRemoteLogs(ctx, f.network, open); err == nil {
		t.Fatal("cancelled collection reported success")
	}
	logs, _ = m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
	if len(logs.Entries) != 2 {
		t.Fatalf("cancelled collector archived lines: %+v", logs.Entries)
	}
}

func TestWebRemoteLogsKeepPositionsAndStopOnRefusal(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	detach := m.attachRemote(f.network)
	defer detach()

	// A slow remote read must not stall the background collector.
	release := make(chan struct{})
	blocked := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- m.CollectRemoteLogs(context.Background(), f.network, func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
			if ns.Label == "node2" {
				close(blocked)
				<-release
			}
			return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
		})
	}()
	<-blocked
	background := make(chan error, 1)
	go func() { background <- m.CollectOnce(context.Background()) }()
	select {
	case err := <-background:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("background collection waited for a remote read")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// Cancellation after node1 was archived keeps node1's position: the
	// next job does not archive the same line again.
	ctx, cancel := context.WithCancel(context.Background())
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] first\nINFO [10-09|00:46:59.000] second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.clock = f.clock.Add(5 * time.Second)
	err := m.CollectRemoteLogs(ctx, f.network, func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		if ns.Label == "node2" {
			cancel()
		}
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	})
	if err == nil {
		t.Fatal("cancelled round reported success")
	}
	f.clock = f.clock.Add(5 * time.Second)
	refused := m.CollectRemoteLogs(context.Background(), f.network, func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		if ns.Label == "node1" {
			return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
		}
		return nil, ErrDeploymentForbidden
	})
	if !errors.Is(refused, ErrDeploymentForbidden) {
		t.Fatalf("refused access must end collection: %v", refused)
	}
	logs, err := m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Entries) != 2 || logs.Entries[1].Text != "INFO [10-09|00:46:59.000] second" {
		t.Fatalf("positions lost or lines duplicated: %+v", logs.Entries)
	}
	other, _ := m.Logs(context.Background(), viewer, f.network+".node2", WebMonitorQuery{})
	for _, g := range other.Coverage.Gaps {
		if strings.Contains(g.Reason, "collector_error") {
			t.Fatalf("refusal recorded as an ordinary failure: %v", g)
		}
	}
}

// TestWebRemoteLogsReachTheEventStream: lines a remote collection job archives
// reach the events stream as local lines do, and so does a remote read that
// starts or stops failing.
func TestWebRemoteLogsReachTheEventStream(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] remote one\nWARN [10-09|00:46:59.000] remote two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	detach := m.attachRemote(f.network)
	events, stop := m.SubscribeObservations()
	defer stop()
	failing := false
	open := func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		if failing && ns.Label == "node1" {
			return nil, errors.New("remote read failed")
		}
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogs(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	logs := eventsOfType(drainObservations(events), "logs")
	if len(logs) != 1 || logs[0].NetworkID != f.network || logs[0].NodeID != "node1" || len(logs[0].Lines) != 2 || logs[0].Lines[1].Level != "WARN" {
		t.Fatalf("remote log events = %+v", logs)
	}
	failing = true
	f.clock = f.clock.Add(5 * time.Second)
	if err := m.CollectRemoteLogs(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	opened := eventsOfType(drainObservations(events), "gap")
	if len(opened) != 1 || opened[0].GapState != "opened" || opened[0].Gap.NodeID != "node1" || opened[0].Gap.Code != "collector_error" {
		t.Fatalf("remote gap opened events = %+v", opened)
	}
	failing = false
	f.clock = f.clock.Add(5 * time.Second)
	if err := m.CollectRemoteLogs(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	closed := eventsOfType(drainObservations(events), "gap")
	if len(closed) != 1 || closed[0].GapState != "closed" || closed[0].Gap.Code != "collector_error" {
		t.Fatalf("remote gap closed events = %+v", closed)
	}
	// Detaching closes what is still open: node2's log never existed remotely.
	detach()
	if gone := eventsOfType(drainObservations(events), "gap"); len(gone) != 1 || gone[0].GapState != "closed" || gone[0].Gap.NodeID != "node2" || gone[0].Gap.Code != "log_unavailable" {
		t.Fatalf("detach gap events = %+v", gone)
	}
}

// One round reads at most webMonitorReadLimit from each node. A job that holds
// its logins only until it ends has to read on until the logs are drained.
func TestWebRemoteLogsCollectToTheEndOfALongLog(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	line := "INFO [10-09|00:46:58.000] " + strings.Repeat("x", 1000) + "\n"
	content := strings.Repeat(line, webMonitorReadLimit*5/2/len(line))
	if err := os.WriteFile(f.logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	open := func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	cursor, err := m.readCursor(f.network)
	if err != nil {
		t.Fatal(err)
	}
	if got := cursor.Offsets["node1"]; got != int64(len(content)) {
		t.Fatalf("node1 offset = %d, want the whole log of %d bytes", got, len(content))
	}
}

// A run that cleaned up its nodes keeps their logs. Those logs are the reason
// to open a failed job, so the nodes stay readable after the record lost them.
func TestLogsOfNodesTheRecordNoLongerHasStayReadable(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] kept line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	open := func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(f.root, "networks", f.network, "chain-record.json")
	raw, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err = json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	state["nodes"] = []any{}
	writeMonitorJSON(t, record, state)

	if got := m.ArchivedNodes(f.network); !slices.Contains(got, "node1") {
		t.Fatalf("archived nodes = %v", got)
	}
	logs, err := m.Logs(context.Background(), DeploymentActor{ID: "viewer", Role: "viewer"}, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatalf("logs of a cleaned node: %v", err)
	}
	if len(logs.Entries) != 1 || logs.Entries[0].Text != "INFO [10-09|00:46:58.000] kept line" {
		t.Fatalf("entries = %+v", logs.Entries)
	}
	if _, err = m.Logs(context.Background(), DeploymentActor{ID: "viewer", Role: "viewer"}, f.network+".node9", WebMonitorQuery{}); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("an unknown node: %v", err)
	}
}
