package dashboard_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

func TestHealthAndIndex(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(collector.NewBus(), collector.NewMemStore()))
	defer srv.Close()

	if b := get(t, srv.URL+"/healthz"); b != "ok" {
		t.Errorf("healthz: %q", b)
	}
	// / serves the Svelte SPA index, which loads its bundle from /assets/.
	if b := get(t, srv.URL+"/"); !strings.Contains(b, "chainbench") || !strings.Contains(b, "/assets/") {
		t.Errorf("SPA index missing content: %q", b)
	}
	// The legacy build-free page remains available as a no-JS fallback.
	if b := get(t, srv.URL+"/legacy"); !strings.Contains(b, "chainbench") || !strings.Contains(b, "EventSource") {
		t.Errorf("legacy page missing content")
	}
}

// Hashed assets never change, so a browser keeps them; text assets travel gzipped.
func TestSPAAssetsAreCachedAndCompressed(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(collector.NewBus(), collector.NewMemStore()))
	defer srv.Close()
	idx := get(t, srv.URL+"/")
	start := strings.Index(idx, "/assets/")
	if start < 0 {
		t.Fatal("no asset ref in SPA index")
	}
	assetPath := idx[start : start+strings.IndexAny(idx[start:], "\"'")]

	plain, err := http.DefaultTransport.RoundTrip(mustRequest(t, srv.URL+assetPath, ""))
	if err != nil {
		t.Fatal(err)
	}
	want, err := io.ReadAll(plain.Body)
	plain.Body.Close()
	if err != nil || plain.Header.Get("Content-Encoding") != "" || len(want) == 0 {
		t.Fatalf("plain read: encoding %q, %d bytes, %v", plain.Header.Get("Content-Encoding"), len(want), err)
	}
	if got := plain.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}

	packed, err := http.DefaultTransport.RoundTrip(mustRequest(t, srv.URL+assetPath, "gzip"))
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Body.Close()
	if packed.Header.Get("Content-Encoding") != "gzip" || packed.Header.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("headers %v", packed.Header)
	}
	if !strings.HasPrefix(packed.Header.Get("Content-Type"), plain.Header.Get("Content-Type")) {
		t.Fatalf("type changed: %q then %q", plain.Header.Get("Content-Type"), packed.Header.Get("Content-Type"))
	}
	reader, err := gzip.NewReader(packed.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("gzip body differs: %d vs %d bytes, %v", len(got), len(want), err)
	}
}

func mustRequest(t *testing.T, url, encoding string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if encoding != "" {
		r.Header.Set("Accept-Encoding", encoding)
	}
	return r
}

func TestSPAAssetServed(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(collector.NewBus(), collector.NewMemStore()))
	defer srv.Close()

	// A hashed asset referenced by the SPA index must be served (non-empty).
	idx := get(t, srv.URL+"/")
	start := strings.Index(idx, "/assets/")
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
	store := collector.NewMemStore()
	_ = store.SaveRun(collector.RunRecord{ID: "test/a", Phase: collector.PhaseTest, Status: collector.RunSucceeded})
	srv := httptest.NewServer(dashboard.NewServer(collector.NewBus(), store, dashboard.WithLegacyObservation()))
	defer srv.Close()

	var runs []collector.RunRecord
	body := get(t, srv.URL+"/api/runs")
	if err := json.Unmarshal([]byte(body), &runs); err != nil {
		t.Fatalf("runs json: %v (%s)", err, body)
	}
	if len(runs) != 1 || runs[0].ID != "test/a" {
		t.Errorf("runs: %+v", runs)
	}
}

func TestSSE_StreamsPublishedEvents(t *testing.T) {
	bus := collector.NewBus()
	defer bus.Close()
	srv := httptest.NewServer(dashboard.NewServer(bus, nil, dashboard.WithLegacyObservation()))
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
				bus.Publish(collector.Event{Phase: collector.PhaseSetup, Kind: collector.KindResult, Message: "hello-dash"})
			}
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var e collector.Event
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &e); err != nil {
				t.Fatalf("bad SSE json: %v", err)
			}
			if e.Message == "hello-dash" && e.Phase == collector.PhaseSetup {
				return // success
			}
		}
	}
	t.Fatalf("did not receive expected SSE event: %v", scanner.Err())
}

func TestPublishEndpoint(t *testing.T) {
	bus := collector.NewBus()
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
