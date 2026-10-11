package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestErrorsCarryMachineCodesAndFieldPaths(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := app.OpenWebJobs(t.TempDir(), conflictFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebJobs(jobs, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	_, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	session, _ := auth.Session(token)
	call := func(method, path, body string, header map[string]string) (int, map[string]any) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		r.Header.Set("X-CSRF-Token", session.CSRFToken)
		for k, v := range header {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	code, body := call("POST", "/api/v1/documents", `{"kind":"server-set","name":"pool","contractVersion":"2","content":{"version":2,"pool":{"hosts":[],"slots":2}},"assetRefs":[]}`, nil)
	fields, _ := body["fieldErrors"].([]any)
	if code != 422 || body["code"] != "invalid" || len(fields) != 1 || fields[0].(map[string]any)["path"] != "/content" {
		t.Errorf("invalid document: %d %v", code, body)
	}
	if code, body = call("GET", "/api/v1/documents/absent", "", nil); code != 404 || body["code"] != "not_found" {
		t.Errorf("absent document: %d %v", code, body)
	}
	actor := app.DeploymentActor{ID: session.User.ID, Role: "admin"}
	plan, err := jobs.Plan(context.Background(), actor, app.WebPlanInput{WorkspaceID: "w", Operation: "chain.setup"})
	if err != nil {
		t.Fatal(err)
	}
	if code, body = call("POST", "/api/v1/jobs", `{"planId":"`+plan.ID+`"}`, nil); code != 400 || body["code"] != "bad_request" {
		t.Errorf("job without an idempotency key: %d %v", code, body)
	}
}
