package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestTestCaseHTTPContractAndImport(t *testing.T) {
	bus := collector.NewBus()
	defer bus.Close()
	auth := func(r *http.Request) (app.DeploymentActor, error) {
		if r.Header.Get("Authorization") != "test-login" {
			return app.DeploymentActor{}, app.ErrDeploymentForbidden
		}
		return app.DeploymentActor{ID: "editor", Role: "operator"}, nil
	}
	server := NewServer(bus, nil, WithTestCases(auth))
	request := func(method, path, body string, login bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if login {
			r.Header.Set("Authorization", "test-login")
		}
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/api/v1/vocabulary", "/api/v1/contracts/dsl"} {
		if w := request("GET", path, "", false); w.Code != 401 {
			t.Fatalf("unauthenticated %s: %d", path, w.Code)
		}
		w := request("GET", path, "", true)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var decoded map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	content := `{"schemaVersion":"2","kind":"case","id":"http","chainPreset":{"chain":"stablenet","binaries":{"default":"gstable"}},"steps":[{"expect":"blockNumber","is":1}]}`
	w := request("POST", "/api/v1/test-cases/import", `{"content":`+content+`}`, true)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = request("POST", "/api/v1/documents/validate", `{"kind":"case","name":"http","contractVersion":"2","content":`+content+`}`, true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"valid":true`) {
		t.Fatal(w.Body.String())
	}
	w = request("POST", "/api/v1/test-cases/import", `{"content":`+strings.Replace(content, `"blockNumber"`, `"unknown"`, 1)+`}`, true)
	if w.Code != 422 {
		t.Fatal(w.Body.String())
	}
	w = request("POST", "/api/v1/test-cases/import", `{"content":`+content+`} {}`, true)
	if w.Code != 400 {
		t.Fatal("trailing input accepted")
	}
}
