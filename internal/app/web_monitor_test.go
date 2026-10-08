package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

type monitorFixture struct {
	root, network, logPath string
	clock                  time.Time
}

func newMonitorFixture(t *testing.T) *monitorFixture {
	t.Helper()
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		result := map[string]string{"eth_blockNumber": "0x7", "net_peerCount": "0x3"}[req.Method]
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%q}`, result)
	}))
	t.Cleanup(rpc.Close)
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/debug/metrics/prometheus" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, "# TYPE chain_head_block gauge\nchain_head_block 7\np2p_peers 3\ntxpool_queued 0\n")
	}))
	t.Cleanup(metrics.Close)
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedPort := closed.Addr().(*net.TCPAddr).Port
	_ = closed.Close()

	root := t.TempDir()
	network := "0123456789abcdef0123456789abcdef"
	dir := filepath.Join(root, "networks", network)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "node1.log")
	state := State{Chain: "wbft", Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{
		{Index: 1, Label: "node1", Role: "bp", Host: "127.0.0.1", PID: 101, LogPath: logPath,
			Endpoints: node.Endpoints{HTTP: monitorPort(t, rpc.URL), Metrics: monitorPort(t, metrics.URL)}},
		{Index: 2, Label: "node2", Role: "bp", Host: "127.0.0.1", PID: 102, LogPath: filepath.Join(root, "node2.log"),
			Endpoints: node.Endpoints{HTTP: closedPort}},
		{Index: 3, Label: "node3", Role: "bp", Host: "127.0.0.1", LogPath: filepath.Join(root, "node3.log"),
			Endpoints: node.Endpoints{HTTP: closedPort}},
	}}
	writeMonitorJSON(t, filepath.Join(dir, "chain-record.json"), state)
	writeMonitorJSON(t, filepath.Join(dir, "web-target.json"), resource.Inspection{HostIdentity: "fixture", Transport: "local"})
	clock := time.Date(2026, 10, 9, 0, 47, 0, 0, time.Local)
	return &monitorFixture{root: root, network: network, logPath: logPath, clock: clock}
}

func monitorPort(t *testing.T, raw string) int {
	t.Helper()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(raw, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	var n int
	_, _ = fmt.Sscan(port, &n)
	return n
}

func writeMonitorJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f *monitorFixture) monitor() *WebMonitor {
	m, err := OpenWebMonitor(f.root, func(s string) string { return strings.ReplaceAll(s, "secret-value", "[redacted]") })
	if err != nil {
		panic(err)
	}
	m.now = func() time.Time { return f.clock }
	return m
}

func seriesValues(metrics WebMetrics, nodeID, name, source string) []float64 {
	for _, s := range metrics.Series {
		if s.NodeID == nodeID && s.Name == name && s.Source == source {
			out := []float64{}
			for _, sample := range s.Samples {
				if sample.Value != nil {
					out = append(out, *sample.Value)
				}
			}
			return out
		}
	}
	return nil
}

func gapReasons(c WebCoverage) []string {
	out := []string{}
	for _, g := range c.Gaps {
		out = append(out, g.Reason)
	}
	return out
}

func TestWebMonitorRecordsSourcesUnitsAndGapsWithoutInventingValues(t *testing.T) {
	f := newMonitorFixture(t)
	m := f.monitor()
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	for i := 0; i < 2; i++ {
		if err := m.CollectOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		f.clock = f.clock.Add(5 * time.Second)
	}
	got, err := m.Metrics(viewer, f.network, WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got.NetworkID != f.network || got.Coverage.SampleIntervalSeconds != 5 || got.Coverage.Complete {
		t.Fatalf("coverage = %+v", got.Coverage)
	}
	for _, want := range []struct {
		name, source, unit string
		value              float64
	}{{"block_height", "rpc", "blocks", 7}, {"peer_count", "rpc", "peers", 3}, {"chain_head_block", "metrics", "blocks", 7}, {"p2p_peers", "metrics", "peers", 3}, {"txpool_queued", "metrics", "transactions", 0}} {
		values := seriesValues(got, "node1", want.name, want.source)
		if len(values) != 2 || values[0] != want.value {
			t.Fatalf("%s/%s = %v", want.source, want.name, values)
		}
		for _, s := range got.Series {
			if s.NodeID == "node1" && s.Name == want.name && s.Unit != want.unit {
				t.Fatalf("%s unit = %q", want.name, s.Unit)
			}
		}
	}
	for _, s := range got.Series {
		if s.NodeID != "node1" {
			t.Fatalf("unreachable or stopped node gained invented series: %+v", s)
		}
	}
	reasons := strings.Join(gapReasons(got.Coverage), ",")
	for _, want := range []string{"node1 metrics txpool_pending not_exposed", "node2 rpc rpc_unavailable", "node2 metrics metrics_disabled"} {
		if strings.Count(reasons, want) != 1 {
			t.Fatalf("gap %q must appear once (merged across samples): %s", want, reasons)
		}
	}
	if strings.Contains(reasons, "node3") {
		t.Fatalf("node without a recorded process was probed: %s", reasons)
	}
	if _, err = m.Metrics(viewer, f.network, WebMonitorQuery{NodeID: "node1", Metric: "p2p_peers"}); err != nil {
		t.Fatal(err)
	}
	filtered, _ := m.Metrics(viewer, f.network, WebMonitorQuery{NodeID: "node1", Metric: "p2p_peers"})
	if len(filtered.Series) != 1 {
		t.Fatalf("node/metric filter = %+v", filtered.Series)
	}
	if _, err = m.Metrics(viewer, "missing", WebMonitorQuery{}); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("unknown network err = %v", err)
	}
	for _, role := range []string{"admin", "operator", "viewer"} {
		if _, err = m.Metrics(DeploymentActor{ID: role, Role: role}, f.network, WebMonitorQuery{}); err != nil {
			t.Fatalf("%s cannot read shared observations: %v", role, err)
		}
	}
	if _, err = m.Metrics(DeploymentActor{ID: "x", Role: "administrator"}, f.network, WebMonitorQuery{}); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatalf("unmapped account role err = %v", err)
	}
	if _, err = m.Metrics(DeploymentActor{}, f.network, WebMonitorQuery{}); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatalf("anonymous err = %v", err)
	}
}

func TestWebMonitorArchivesTimeLinkedLogsAcrossRestart(t *testing.T) {
	f := newMonitorFixture(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	if err := os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:20.597] Blockchain stopped key=secret-value\nplain line without timestamp\nINFO [10-09|00:46:21"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	if err := m.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	logs, err := m.Logs(viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Entries) != 2 || logs.NextCursor != nil {
		t.Fatalf("complete lines only: %+v", logs.Entries)
	}
	first := logs.Entries[0]
	if !first.Time.Equal(time.Date(2026, 10, 9, 0, 46, 20, 597000000, time.Local)) || first.TimestampSource != "source" || strings.Contains(first.Text, "secret-value") {
		t.Fatalf("source-timed redacted entry = %+v", first)
	}
	if second := logs.Entries[1]; !second.Time.Equal(f.clock) || second.TimestampSource != "collection" {
		t.Fatalf("collection-timed entry = %+v", second)
	}
	if logs.Coverage.TimestampSource != "mixed" {
		t.Fatalf("coverage timestamp source = %q", logs.Coverage.TimestampSource)
	}

	// The dashboard restarts; archived samples and logs stay, the unobserved
	// interval becomes an explicit gap, and the partial line is read once.
	f.clock = f.clock.Add(time.Minute)
	if err = os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:20.597] Blockchain stopped key=secret-value\nplain line without timestamp\nINFO [10-09|00:46:21.000] resumed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	restarted := f.monitor()
	if err = restarted.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	logs, err = restarted.Logs(viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Entries) != 3 || !strings.HasSuffix(logs.Entries[2].Text, "resumed") {
		t.Fatalf("restart duplicated or lost lines: %+v", logs.Entries)
	}
	window, err := restarted.Logs(viewer, f.network+".node1", WebMonitorQuery{From: time.Date(2026, 10, 9, 0, 46, 20, 900000000, time.Local), To: time.Date(2026, 10, 9, 0, 46, 21, 500000000, time.Local)})
	if err != nil || len(window.Entries) != 1 || !strings.HasSuffix(window.Entries[0].Text, "resumed") {
		t.Fatalf("time-linked window = %+v, %v", window.Entries, err)
	}
	metrics, err := restarted.Metrics(viewer, f.network, WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if values := seriesValues(metrics, "node1", "block_height", "rpc"); len(values) != 2 {
		t.Fatalf("archived samples lost across restart: %v", values)
	}
	if !strings.Contains(strings.Join(gapReasons(metrics.Coverage), ","), "collector_stopped") {
		t.Fatalf("restart interval not reported: %v", gapReasons(metrics.Coverage))
	}
	if _, err = restarted.Logs(viewer, f.network+".node9", WebMonitorQuery{}); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("unknown node err = %v", err)
	}
}

func TestWebMonitorReportsRemoteLogsAsUnavailable(t *testing.T) {
	f := newMonitorFixture(t)
	writeMonitorJSON(t, filepath.Join(f.root, "networks", f.network, "web-target.json"), resource.Inspection{HostIdentity: "remote", Transport: "ssh"})
	m := f.monitor()
	if err := m.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	logs, err := m.Logs(DeploymentActor{ID: "viewer", Role: "viewer"}, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Entries) != 0 || logs.Coverage.Complete || !strings.Contains(strings.Join(gapReasons(logs.Coverage), ","), "remote_log_collection_unavailable") {
		t.Fatalf("remote log coverage = %+v", logs.Coverage)
	}
}

func monitorCollect(t *testing.T, f *monitorFixture, m *WebMonitor, rounds int) {
	t.Helper()
	for i := 0; i < rounds; i++ {
		_ = m.CollectOnce(context.Background())
		f.clock = f.clock.Add(5 * time.Second)
	}
}

func TestWebMonitorLogArchiveSurvivesOversizedRotatedAndMultilineSecrets(t *testing.T) {
	f := newMonitorFixture(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	huge := strings.Repeat("<", webMonitorReadLimit+10)
	content := "INFO [10-09|00:46:00.000] before\n-----BEGIN PRIVATE KEY-----\nMIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA\n-----END PRIVATE KEY-----\n" + huge + "\nINFO [10-09|00:46:01.000] after <tag>\n"
	if err := os.WriteFile(f.logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	monitorCollect(t, f, m, 4)
	logs, err := m.Logs(viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil {
		t.Fatal(err)
	}
	texts := []string{}
	for _, e := range logs.Entries {
		texts = append(texts, e.Text)
	}
	joined := strings.Join(texts, "\n")
	if strings.Contains(joined, "MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA") {
		t.Fatalf("multi-line key archived: %q", joined)
	}
	if !strings.HasSuffix(joined, "after <tag>") || strings.Contains(joined, strings.Repeat("<", 100)) {
		t.Fatalf("oversized line blocked or stored: last=%q count=%d", texts[len(texts)-1], len(texts))
	}
	if !strings.Contains(strings.Join(gapReasons(logs.Coverage), ","), "log_line_oversized") {
		t.Fatalf("oversized line not reported: %v", gapReasons(logs.Coverage))
	}
	// Replace the file with a new one larger than the old offset.
	rotated := strings.Repeat("INFO [10-09|00:46:30.000] rotated line\n", 40000)
	if err = os.Remove(f.logPath); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(f.logPath, []byte(rotated), 0o600); err != nil {
		t.Fatal(err)
	}
	monitorCollect(t, f, m, 3)
	logs, err = m.Logs(viewer, f.network+".node1", WebMonitorQuery{From: time.Date(2026, 10, 9, 0, 46, 29, 0, time.Local), To: time.Date(2026, 10, 9, 0, 46, 31, 0, time.Local), Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Entries) != 100 || logs.Entries[0].Text != "INFO [10-09|00:46:30.000] rotated line" {
		t.Fatalf("rotation read from the middle: %d %q", len(logs.Entries), logs.Entries[0].Text)
	}
	// The unread interval is known only by collection time, so it is reported there.
	recent, err := m.Logs(viewer, f.network+".node1", WebMonitorQuery{Limit: 1})
	if err != nil || !strings.Contains(strings.Join(gapReasons(recent.Coverage), ","), "node1 logs log_rotated") {
		t.Fatalf("rotation unreported: %v %v", gapReasons(recent.Coverage), err)
	}
	next, err := m.Logs(viewer, f.network+".node1", WebMonitorQuery{Cursor: *logs.NextCursor})
	if err != nil || len(next.Entries) != 100 {
		t.Fatalf("cursor must keep its window: %d %v", len(next.Entries), err)
	}
	if _, err = m.Logs(viewer, f.network+".node1", WebMonitorQuery{Cursor: *logs.NextCursor, From: time.Date(2026, 10, 9, 0, 0, 0, 0, time.Local)}); !errors.Is(err, ErrWebMonitorQuery) {
		t.Fatalf("cursor reused with another window: %v", err)
	}
}

func TestWebMonitorCoverageReflectsRealIntervalsAndCorruption(t *testing.T) {
	f := newMonitorFixture(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	m := f.monitor()
	start := f.clock
	monitorCollect(t, f, m, 1)
	f.clock = f.clock.Add(40 * time.Second) // A slow round: nothing observed meanwhile.
	monitorCollect(t, f, m, 1)
	got, err := m.Metrics(viewer, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: f.clock})
	if err != nil {
		t.Fatal(err)
	}
	var rpcGap *WebCoverageGap
	for i, g := range got.Coverage.Gaps {
		if g.Reason == "node2 rpc rpc_unavailable" {
			rpcGap = &got.Coverage.Gaps[i]
		}
	}
	if rpcGap == nil || rpcGap.To.Sub(rpcGap.From) < 45*time.Second {
		t.Fatalf("long round shortened the outage: %+v", got.Coverage.Gaps)
	}
	if !strings.Contains(strings.Join(gapReasons(got.Coverage), ","), "collector not_collected") {
		t.Fatalf("window before the first round reported as covered: %v", gapReasons(got.Coverage))
	}
	if _, err = m.Metrics(viewer, f.network, WebMonitorQuery{From: start.AddDate(-1, 0, 0), To: start}); !errors.Is(err, ErrWebMonitorQuery) {
		t.Fatalf("unbounded window accepted: %v", err)
	}
	// A torn or corrupt archive line is skipped and reported, not fatal.
	if err = m.store.Append(f.network, webMonitorDay("samples", start)+".jsonl", []byte("{broken\n{\"t\":\"partial")); err != nil {
		t.Fatal(err)
	}
	got, err = m.Metrics(viewer, f.network, WebMonitorQuery{From: start.Add(-time.Minute), To: f.clock})
	if err != nil {
		t.Fatal(err)
	}
	if values := seriesValues(got, "node1", "block_height", "rpc"); len(values) != 2 || !strings.Contains(strings.Join(gapReasons(got.Coverage), ","), "archive corrupt_record") {
		t.Fatalf("corrupt archive handling: %v %v", values, gapReasons(got.Coverage))
	}
}

func TestWebMonitorKeepsRecordingWhenOneNodeFails(t *testing.T) {
	f := newMonitorFixture(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	// node2's log path is a directory: reading it fails, node1 must still be archived.
	state, target, err := (&WebMonitor{root: f.root}).loadNetwork(f.network)
	if err != nil {
		t.Fatal(err)
	}
	_ = target
	if err = os.MkdirAll(state.Nodes[1].LogPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(f.logPath, []byte("INFO [10-09|00:46:59.000] one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := f.monitor()
	monitorCollect(t, f, m, 2)
	logs, err := m.Logs(viewer, f.network+".node1", WebMonitorQuery{})
	if err != nil || len(logs.Entries) != 1 {
		t.Fatalf("a failing sibling duplicated or lost lines: %+v %v", logs.Entries, err)
	}
	other, err := m.Logs(viewer, f.network+".node2", WebMonitorQuery{})
	if err != nil || other.Coverage.Complete || len(other.Coverage.Gaps) == 0 {
		t.Fatalf("failing node not reported: %+v %v", other.Coverage, err)
	}
}
