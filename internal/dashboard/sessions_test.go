package dashboard_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

// writeSession creates a real session with a chainstate sample under root and
// returns its id.
func writeSession(t *testing.T, root string) string {
	t.Helper()
	s, err := session.New(root, "run", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	rec := s.Test(1, "T1")
	rec.Status(session.StatusPass)
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	env, err := s.NewEnvironment("ffffffffffff0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	cs := filepath.Join(env.ChainstateDir(), "chainstate.jsonl")
	if err := os.WriteFile(cs, []byte(`{"seq":1,"forked":false}`+"\n"+`{"seq":2,"forked":true}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return s.ID()
}

func getResp(t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, body
}

func TestSessionsAPI(t *testing.T) {
	root := t.TempDir()
	id := writeSession(t, root)

	srv := httptest.NewServer(dashboard.NewServer(obs.NewBus(), nil, dashboard.WithArtifactRoot(root)))
	defer srv.Close()

	// List.
	_, body := getResp(t, srv.URL+"/api/sessions")
	var ids []string
	if err := json.Unmarshal(body, &ids); err != nil {
		t.Fatalf("unmarshal sessions: %v (%s)", err, body)
	}
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("sessions = %v, want [%s]", ids, id)
	}

	// Summary.
	resp, body := getResp(t, srv.URL+"/api/sessions/"+id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary status = %d", resp.StatusCode)
	}
	var summary struct {
		Tests []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tests"`
		Summary struct{ Pass int } `json:"summary"`
	}
	if err := json.Unmarshal(body, &summary); err != nil {
		t.Fatalf("unmarshal summary: %v (%s)", err, body)
	}
	if summary.Summary.Pass != 1 || len(summary.Tests) != 1 || summary.Tests[0].ID != "T1" {
		t.Fatalf("summary = %+v", summary)
	}

	// Chainstate.
	_, body = getResp(t, srv.URL+"/api/sessions/"+id+"/chainstate")
	var samples []map[string]any
	if err := json.Unmarshal(body, &samples); err != nil {
		t.Fatalf("unmarshal chainstate: %v (%s)", err, body)
	}
	if len(samples) != 2 || samples[1]["forked"] != true {
		t.Fatalf("chainstate samples = %v", samples)
	}
}

func TestSessionsAPI_UnknownAndBadID(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(obs.NewBus(), nil, dashboard.WithArtifactRoot(t.TempDir())))
	defer srv.Close()

	if resp, _ := getResp(t, srv.URL+"/api/sessions/UTC-nope"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown session status = %d, want 404", resp.StatusCode)
	}
	if resp, _ := getResp(t, srv.URL+"/api/sessions/..%2f..%2fetc"); resp.StatusCode == http.StatusOK {
		t.Fatalf("traversal id must not succeed, got %d", resp.StatusCode)
	}
}

func TestSessionsAPI_NoArtifactRoot(t *testing.T) {
	srv := httptest.NewServer(dashboard.NewServer(obs.NewBus(), nil))
	defer srv.Close()

	_, body := getResp(t, srv.URL+"/api/sessions")
	var ids []string
	if err := json.Unmarshal(body, &ids); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, body)
	}
	if len(ids) != 0 {
		t.Fatalf("sessions = %v, want empty without artifact root", ids)
	}
}
