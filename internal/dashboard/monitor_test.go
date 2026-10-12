package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestMonitorRoutesServeViewersAndRejectBadQueries(t *testing.T) {
	root := t.TempDir()
	network := "monitored"
	dir := filepath.Join(root, "networks", network)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "node1.log")
	if err := os.WriteFile(logPath, []byte("imported without a source timestamp\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]any{
		"chain-record.json": app.State{Chain: "wbft", Nodes: []node.Record{{Index: 1, Label: "node1", Host: "127.0.0.1", PID: 7, LogPath: logPath}}},
		"web-target.json":   resource.Inspection{HostIdentity: "fixture", Transport: "local"},
	} {
		raw, _ := json.Marshal(v)
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	monitor, err := app.OpenWebMonitor(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := monitor.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := NewServer(bus, nil, WithWebMonitor(monitor, func(r *http.Request) (app.DeploymentActor, error) {
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
	w := call("viewer", "/api/v1/networks/monitored/metrics")
	var metrics struct {
		NetworkID string            `json:"networkId"`
		Series    []json.RawMessage `json:"series"`
		Coverage  map[string]any    `json:"coverage"`
	}
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || json.Unmarshal(w.Body.Bytes(), &metrics) != nil || metrics.NetworkID != network || metrics.Series == nil || metrics.Coverage["complete"] != false {
		t.Fatalf("viewer metrics = %d %s", w.Code, w.Body.String())
	}
	w = call("viewer", "/api/v1/nodes/monitored.node1/logs?limit=1")
	var logs struct {
		Entries []map[string]any `json:"entries"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &logs) != nil || len(logs.Entries) != 1 || len(logs.Entries[0]) != 2 {
		t.Fatalf("viewer logs = %d %s", w.Code, w.Body.String())
	}
	for path, want := range map[string]int{
		"/api/v1/networks/monitored/metrics?from=yesterday":                                    400,
		"/api/v1/networks/monitored/metrics?from=2026-10-09T00:00:00Z&to=2026-10-08T00:00:00Z": 400,
		"/api/v1/nodes/monitored.node1/logs?limit=101":                                         400,
		"/api/v1/nodes/monitored.node1/logs?cursor=-1":                                         400,
		"/api/v1/networks/monitored/metrics?from=2026-01-01T00:00:00Z&to=2026-03-01T00:00:00Z": 400,
		"/api/v1/networks/absent/metrics":                                                      404,
		"/api/v1/nodes/monitored.node9/logs":                                                   404,
	} {
		if w := call("viewer", path); w.Code != want {
			t.Errorf("%s = %d, want %d", path, w.Code, want)
		}
	}
	if w := call("", "/api/v1/networks/monitored/metrics"); w.Code != 401 {
		t.Errorf("anonymous metrics = %d", w.Code)
	}
}
