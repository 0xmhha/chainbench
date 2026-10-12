package dashboard

import (
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

func TestAuditTrailRecordsTargetsRequestsAndIsAdministratorOnly(t *testing.T) {
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
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebSecurity(auth, store), WithWebAudit(auth, store, jobs, nil, nil))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	admin, adminToken, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	operator, err := auth.CreateUser(app.DeploymentActor{ID: admin.User.ID, Role: "admin"}, "operator", "long-user-password", "operator")
	if err != nil {
		t.Fatal(err)
	}
	opSession, opToken, err := auth.Login("operator", "long-user-password")
	if err != nil {
		t.Fatal(err)
	}
	adminSession, _ := auth.Session(adminToken)
	call := func(method, path, body, token, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		r.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	refused := call("PATCH", "/api/v1/users/"+operator.ID, `{"role":"administrator"}`, opToken, opSession.CSRFToken)
	if refused.Code != 403 {
		t.Fatalf("operator changed a role: %d", refused.Code)
	}
	var refusal map[string]any
	_ = json.Unmarshal(refused.Body.Bytes(), &refusal)
	again := call("PATCH", "/api/v1/users/"+operator.ID, `{"role":"administrator"}`, opToken, opSession.CSRFToken)
	var second map[string]any
	_ = json.Unmarshal(again.Body.Bytes(), &second)
	if refusal["requestId"] == "" || refusal["requestId"] == second["requestId"] || refused.Header().Get("X-Request-ID") != refusal["requestId"] {
		t.Errorf("request ids %v %v header %q", refusal["requestId"], second["requestId"], refused.Header().Get("X-Request-ID"))
	}
	if w := call("PATCH", "/api/v1/users/"+operator.ID, `{"role":"viewer"}`, adminToken, adminSession.CSRFToken); w.Code != 200 {
		t.Fatalf("administrator role change %d %s", w.Code, w.Body)
	}
	if w := call("GET", "/api/v1/audit", "", opToken, ""); w.Code != 401 && w.Code != 403 {
		t.Fatalf("a non-administrator read the audit trail: %d", w.Code)
	}
	w := call("GET", "/api/v1/audit", "", adminToken, "")
	if w.Code != 200 {
		t.Fatalf("audit %d %s", w.Code, w.Body)
	}
	var page struct {
		Items []app.WebAuditRecord `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	var change, refusalRecord *app.WebAuditRecord
	ids := map[string]bool{}
	for i, rec := range page.Items {
		if ids[rec.ID] || rec.ID == "" {
			t.Errorf("record id %q repeats or is empty", rec.ID)
		}
		ids[rec.ID] = true
		if rec.Target == "users/"+operator.ID && rec.Result == "200" {
			change = &page.Items[i]
		}
		if rec.Target == "users/"+operator.ID && rec.Result == "403" && rec.RequestID == refusal["requestId"] {
			refusalRecord = &page.Items[i]
		}
	}
	if change == nil || change.ActorID != admin.User.ID || change.RequestID == "" {
		t.Errorf("the role change is not in the trail with its target: %+v", page.Items)
	}
	if refusalRecord == nil || refusalRecord.ActorID != operator.ID {
		t.Errorf("the refusal is not in the trail with its request id")
	}
	if strings.Contains(w.Body.String(), "long-user-password") {
		t.Error("the trail contains a password")
	}
}
