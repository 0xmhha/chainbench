package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestDeploymentSharedDocumentsHTTP(t *testing.T) {
	store, err := app.OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithDeployments(store, func(r *http.Request) (app.DeploymentActor, error) {
		user, pass, ok := r.BasicAuth()
		if !ok || pass != "test-password" {
			return app.DeploymentActor{}, fmt.Errorf("denied")
		}
		return app.DeploymentActor{ID: user, Role: "operator"}, nil
	}))
	call := func(user, method, path, body, revision string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != "" {
			r.SetBasicAuth(user, "test-password")
		}
		if revision != "" {
			r.Header.Set("If-Match", revision)
		}
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		return w
	}
	input := `{"kind":"server-set","name":"Shared","contractVersion":"2","content":{"version":2,"pool":{"hosts":["127.0.0.1"],"slots":1}}}`
	if w := call("", "POST", "/api/v1/documents", input, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	created := call("alice", "POST", "/api/v1/documents", input, "")
	if created.Code != 201 {
		t.Fatal(created.Body.String())
	}
	var d app.DeploymentDocument
	if err = json.Unmarshal(created.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if w := call("bob", "GET", "/api/v1/documents/"+d.ID, "", ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := call("bob", "PATCH", "/api/v1/documents/"+d.ID, input, ""); w.Code != 428 {
		t.Fatal(w.Code)
	}
	if w := call("bob", "PATCH", "/api/v1/documents/"+d.ID, input, `"1"`); w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("alice", "PATCH", "/api/v1/documents/"+d.ID, input, `"1"`); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w := call("bob", "GET", "/api/v1/documents/"+d.ID+"/export", "", ""); w.Code != 200 || strings.Contains(w.Body.String(), "password") {
		t.Fatal(w.Body.String())
	}
	if w := call("bob", "POST", "/api/v1/documents", input+` {}`, ""); w.Code != 400 {
		t.Fatal("trailing input accepted")
	}
	c := call("alice", "POST", "/api/v1/credentials", `{"label":"key","kind":"password","sshUser":"test","password":"private-marker"}`, "")
	if c.Code != 201 {
		t.Fatal(c.Body.String())
	}
	var credential app.DeploymentCredential
	if err = json.Unmarshal(c.Body.Bytes(), &credential); err != nil {
		t.Fatal(err)
	}
	if w := call("bob", "GET", "/api/v1/credentials/"+credential.ID, "", ""); w.Code != 404 {
		t.Fatal("foreign credential visible")
	}
	if w := call("alice", "GET", "/api/v1/credentials", "", ""); strings.Contains(w.Body.String(), "private-marker") {
		t.Fatal("secret returned")
	}
}
