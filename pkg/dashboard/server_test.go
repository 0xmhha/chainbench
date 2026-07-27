package dashboard_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/dashboard"
)

func TestHealthAndIndex(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(obs.NewBus(), obs.NewMemStore()))
	defer srv.Close()

	if b := get(t, srv.URL+"/healthz"); b != "ok" {
		t.Errorf("healthz: %q", b)
	}
	if b := get(t, srv.URL+"/"); !strings.Contains(b, "chainbench") || !strings.Contains(b, "EventSource") {
		t.Errorf("index page missing content")
	}
}

func TestSPAServed(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(obs.NewBus(), obs.NewMemStore()))
	defer srv.Close()

	// The SPA index loads its bundled module from /app/assets/.
	if b := get(t, srv.URL+"/app/"); !strings.Contains(b, "chainbench") || !strings.Contains(b, "/app/assets/") {
		t.Errorf("SPA index missing content: %q", b)
	}
	// A hashed asset referenced by the index must be served (non-empty, JS type).
	idx := get(t, srv.URL+"/app/")
	start := strings.Index(idx, "/app/assets/")
	if start < 0 {
		t.Fatal("no asset ref in SPA index")
	}
	end := strings.IndexAny(idx[start:], "\"'")
	assetPath := idx[start : start+end]
	resp, err := http.Get(srv.URL + assetPath)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("asset %s not served: err=%v status=%v", assetPath, err, resp)
	}
	resp.Body.Close()
}

func TestRunsAPI(t *testing.T) {
	store := obs.NewMemStore()
	_ = store.SaveRun(obs.RunRecord{ID: "test/a", Phase: obs.PhaseTest, Status: obs.RunSucceeded})
	srv := httptest.NewServer(dashboard.NewServer(obs.NewBus(), store))
	defer srv.Close()

	var runs []obs.RunRecord
	body := get(t, srv.URL+"/api/runs")
	if err := json.Unmarshal([]byte(body), &runs); err != nil {
		t.Fatalf("runs json: %v (%s)", err, body)
	}
	if len(runs) != 1 || runs[0].ID != "test/a" {
		t.Errorf("runs: %+v", runs)
	}
}

func TestSSE_StreamsPublishedEvents(t *testing.T) {
	bus := obs.NewBus()
	defer bus.Close()
	srv := httptest.NewServer(dashboard.NewServer(bus, nil))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	// Publish repeatedly until the reader sees a frame (avoids the
	// subscribe-vs-publish race deterministically).
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
				bus.Publish(obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindResult, Message: "hello-dash"})
			}
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var e obs.Event
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &e); err != nil {
				t.Fatalf("bad SSE json: %v", err)
			}
			if e.Message == "hello-dash" && e.Phase == obs.PhaseSetup {
				return // success
			}
		}
	}
	t.Fatalf("did not receive expected SSE event: %v", scanner.Err())
}

func TestPublishEndpoint(t *testing.T) {
	bus := obs.NewBus()
	defer bus.Close()
	sub := bus.Subscribe()
	srv := httptest.NewServer(dashboard.NewServer(bus, nil))
	defer srv.Close()

	body := `{"phase":"test","kind":"result","message":"pushed"}`
	resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: %d", resp.StatusCode)
	}
	select {
	case e := <-sub:
		if e.Message != "pushed" {
			t.Errorf("event: %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("published event not received on bus")
	}
}

func get(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	sb := new(strings.Builder)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return strings.TrimSpace(sb.String())
}
