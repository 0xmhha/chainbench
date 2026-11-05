package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// Remote logs can be read back after the fact. Once a job has drained a node's
// log, nothing before that moment is missing; what comes after still is.
func TestDrainedRemoteLogsClearTheGapBeforeThem(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] kept line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	gaps := func() string {
		t.Helper()
		logs, err := m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(gapReasons(logs.Coverage), ",")
	}
	monitorCollect(t, f, m, 1)
	if !strings.Contains(gaps(), "node1 logs remote_log_collection_unavailable") {
		t.Fatalf("the gap before collection is missing: %s", gaps())
	}
	f.clock = f.clock.Add(5e9)
	open := func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	if got := gaps(); strings.Contains(got, "node1 logs remote_log_collection_unavailable") {
		t.Fatalf("a drained log still reads as missing: %s", got)
	}
	f.clock = f.clock.Add(5e9)
	monitorCollect(t, f, m, 1)
	if got := gaps(); !strings.Contains(got, "node1 logs remote_log_collection_unavailable") {
		t.Fatalf("time after the drain must still read as missing: %s", got)
	}
}

// A new run's log replaces the last run's at the same path. Noticing that must
// not keep the node out of the drain: its log was still read to the end.
func TestDrainingAReplacedLogStillClearsTheGap(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] the last run wrote this longer line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	open := func(_ context.Context, _ State, _ node.Record) (webLogSource, error) {
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:47:58.000] new run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.clock = f.clock.Add(5e9)
	monitorCollect(t, f, m, 1)
	f.clock = f.clock.Add(5e9)
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	logs, err := m.Logs(context.Background(), DeploymentActor{ID: "viewer", Role: "viewer"}, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(gapReasons(logs.Coverage), ","); strings.Contains(got, "node1 logs remote_log_collection_unavailable") {
		t.Fatalf("a replaced log kept the node out of the drain: %s", got)
	}
}

// Cleaning a network ends its nodes and their logs, so the next run's log is a
// first read: nothing was rotated away, and the last lines before the stop are kept.
func TestFinishingRemoteLogsReadsStoppedNodesAndForgetsTheirPositions(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] the last run wrote this longer line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	open := func(_ context.Context, _ State, _ node.Record) (webLogSource, error) {
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] the last run wrote this longer line\nINFO [10-09|00:46:59.000] stopping node\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(f.root, "networks", f.network, "chain-record.json")
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	stopped := regexp.MustCompile(`(?i)"pid":[1-9][0-9]*`).ReplaceAll(raw, []byte(`"pid":0`))
	if bytes.Equal(stopped, raw) {
		t.Fatal("the fixture recorded no running node")
	}
	if err = os.WriteFile(statePath, stopped, 0o600); err != nil {
		t.Fatal(err)
	}
	f.clock = f.clock.Add(5e9)
	if err = m.FinishRemoteLogs(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	logs, err := m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fmt.Sprint(logs.Entries), "stopping node") {
		t.Fatalf("the stopped node's last line was not kept: %v", logs.Entries)
	}
	if err = os.WriteFile(f.logPath, []byte("INFO [10-09|00:47:06.000] new run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(statePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f.clock = f.clock.Add(5e9)
	if err = m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	logs, err = m.Logs(context.Background(), viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(gapReasons(logs.Coverage), ","); strings.Contains(got, "log_rotated") || strings.Contains(got, "log_truncated") {
		t.Fatalf("the next run's log read as a lost one: %s", got)
	}
	if !strings.Contains(fmt.Sprint(logs.Entries), "new run") {
		t.Fatalf("the next run's log was not read from its start: %v", logs.Entries)
	}
}

// A stopped node writes nothing, so its stopped time is missing metrics, not
// missing log lines, and it says nothing about another node's log.
func TestLogCoverageLeavesOutStoppedNodes(t *testing.T) {
	f := newMonitorFixture(t)
	m := f.monitor()
	monitorCollect(t, f, m, 2)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	for _, label := range []string{"node1", "node3"} {
		logs, err := m.Logs(context.Background(), viewer, f.network+"."+label, WebMonitorQuery{})
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(gapReasons(logs.Coverage), ","); strings.Contains(got, "node_stopped") {
			t.Fatalf("%s log coverage reads a stopped node as lost lines: %s", label, got)
		}
	}
	metrics, err := m.Metrics(viewer, f.network, WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(gapReasons(metrics.Coverage), ","); !strings.Contains(got, "node_stopped") {
		t.Fatalf("metrics coverage lost the stopped node: %s", got)
	}
}

func TestFilledARead(t *testing.T) {
	const big = webMonitorReadLimit - 100
	cases := []struct {
		name          string
		before, after int64
		want          bool
	}{
		{"a live node adds little", 1000, 1500, false},
		{"a full read leaves more behind", 1000, 1000 + big, true},
		{"a replaced log started over and filled a read", 5 * big, big, true},
		{"a replaced log started over and ended", 5 * big, 200, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filledARead(map[string]int64{"node1": tc.before}, map[string]int64{"node1": tc.after})
			if got != tc.want {
				t.Fatalf("filledARead = %v, want %v", got, tc.want)
			}
		})
	}
}

// growingLog is a node still running: every read finds a little more.
type growingLog struct {
	webLogSource
	path  string
	reads int
}

func (g *growingLog) Snapshot(ctx context.Context, path string, offset int64, limit int) (webLogSnapshot, error) {
	g.reads++
	file, err := os.OpenFile(g.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return webLogSnapshot{}, err
	}
	_, err = file.WriteString("INFO [10-09|00:46:59.000] another block\n")
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return webLogSnapshot{}, err
	}
	return g.webLogSource.Snapshot(ctx, path, offset, limit)
}

// A live node never stops logging, so "nothing new" never comes. Catching up
// has to end once a read no longer fills, or every job waits out the bound.
func TestWebRemoteLogsCatchUpWithANodeThatKeepsLogging(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:58.000] first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	log := &growingLog{path: f.logPath, webLogSource: webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}}
	open := func(_ context.Context, _ State, ns node.Record) (webLogSource, error) {
		if ns.NodeLabel() == "node1" {
			return log, nil
		}
		return webCLILogs{reader: unitTargetReader(t, nil), spec: resource.Spec{DataRoot: f.root}}, nil
	}
	if err := m.CollectRemoteLogsToEnd(context.Background(), f.network, open); err != nil {
		t.Fatal(err)
	}
	if log.reads > 4 {
		t.Fatalf("read a live log %d times to catch up", log.reads)
	}
	cursor, err := m.readCursor(f.network)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.RemoteDrained["node1"].IsZero() {
		t.Fatal("a caught-up log was not recorded as read")
	}
}
