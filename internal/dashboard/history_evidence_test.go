package dashboard

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// TestHistoryEvidenceRoutesListAndPageAFailedTestsEvidence: a viewer lists a
// run's evidence and reads a page of a node log; a page outside the file is a
// bad request, not a server error.
func TestHistoryEvidenceRoutesListAndPageAFailedTestsEvidence(t *testing.T) {
	root := t.TempDir()
	s, err := session.New(filepath.Join(root, "sessions"), "test run", time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	r := s.Test(3, "fails")
	r.Status(session.StatusFail)
	r.Observation("node1.log", []byte("INFO one\nWARN two\nERROR three\n"))
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	history, err := app.OpenWebHistory(root, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	viewer := func(*http.Request) (app.DeploymentActor, error) {
		return app.DeploymentActor{ID: "reader", Role: "viewer"}, nil
	}
	server := httptest.NewServer(NewServer(nil, nil, WithWebHistory(history, viewer)))
	defer server.Close()
	run := url.PathEscape("web:" + session.IDFor(filepath.Join(root, "sessions"), s.Root()))
	get := func(path string) (int, string) {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, string(b)
	}
	status, body := get("/api/v1/history/" + run + "/evidence")
	var listed app.WebRunEvidence
	if status != 200 || json.Unmarshal([]byte(body), &listed) != nil || len(listed.Tests) != 1 || listed.Tests[0].Seq != 3 || listed.Tests[0].Files[0].Name != "node1.log" {
		t.Fatalf("evidence = %d %s", status, body)
	}
	status, body = get("/api/v1/history/" + run + "/evidence/3/node1.log?offset=1&limit=1")
	var page app.WebEvidencePage
	if status != 200 || json.Unmarshal([]byte(body), &page) != nil || len(page.Lines) != 1 || page.Lines[0] != "WARN two" || page.NextOffset == nil || *page.NextOffset != 2 {
		t.Fatalf("page = %d %s", status, body)
	}
	if status, body = get("/api/v1/history/" + run + "/evidence/3/node1.log?offset=99"); status != 400 {
		t.Fatalf("offset past the end = %d %s", status, body)
	}
	if status, body = get("/api/v1/history/" + run + "/evidence/3/status.json"); status != 404 {
		t.Fatalf("a file that is not evidence = %d %s", status, body)
	}
}
