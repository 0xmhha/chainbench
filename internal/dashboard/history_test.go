package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestHistoryCookieRolesCSRFAndLegacyPreservation(t *testing.T) {
	root, artifacts := t.TempDir(), t.TempDir()
	for i := range 2 {
		s, err := session.New(artifacts, "test", time.Date(2026, 10, 8, 1, 0, i, 0, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		s.Test(1, "case").Status(session.StatusPass)
		if err = s.Save(); err != nil {
			t.Fatal(err)
		}
	}
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	admin, adminToken, err := auth.Bootstrap("admin", "long-history-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: admin.User.ID, Role: "admin"}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	history, err := app.OpenWebHistory(root, artifacts, nil, store.RedactWeb)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := NewServer(bus, nil, WithArtifactRoot(artifacts), WithWebSecurity(auth, store), WithWebHistory(history, WebAuthenticator(auth)))
	call := func(method, path, body, token, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		}
		r.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/v1/history", "", "", ""); w.Code != 401 {
		t.Fatal("anonymous history", w.Code)
	}
	w := call("GET", "/api/v1/history", "", adminToken, "")
	var page app.WebRunPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != 2 {
		t.Fatal(w.Code, w.Body.String())
	}
	ids := []string{page.Items[0].ID, page.Items[1].ID}
	compare, _ := json.Marshal(map[string]any{"runIds": ids})
	for _, role := range []string{"viewer", "operator"} {
		u, err := auth.CreateUser(actor, "history-"+role, "long-other-password", role)
		if err != nil {
			t.Fatal(err)
		}
		ss, token, err := auth.Login(u.Username, "long-other-password")
		if err != nil {
			t.Fatal(err)
		}
		if w = call("GET", "/api/v1/history/"+ids[0]+"/export", "", token, ""); w.Code != 200 || !json.Valid(w.Body.Bytes()) {
			t.Fatal("result export", role, w.Code, w.Body.String())
		}
		if w = call("POST", "/api/v1/history/compare", string(compare), token, ""); w.Code != 403 {
			t.Fatal("comparison without CSRF", w.Code)
		}
		if w = call("POST", "/api/v1/history/compare", string(compare), token, ss.CSRFToken); w.Code != 200 {
			t.Fatal("read-only comparison refused", role, w.Code, w.Body.String())
		}
		var result app.WebComparison
		_ = json.Unmarshal(w.Body.Bytes(), &result)
		if result.Comparable || len(result.Limitations) < 2 {
			t.Fatal("legacy metadata inferred", result)
		}
		if w = call("DELETE", "/api/v1/history/"+ids[0], "", token, ss.CSRFToken); w.Code != 403 {
			t.Fatal("non-admin deletion", role, w.Code)
		}
	}
	if w = call("GET", "/api/v1/history?limit=201", "", adminToken, ""); w.Code != 400 {
		t.Fatal("unbounded listing accepted", w.Code)
	}
	if w = call("DELETE", "/api/v1/history/"+ids[0], "", adminToken, admin.CSRFToken); w.Code != 204 {
		t.Fatal("admin deletion", w.Code, w.Body.String())
	}
	if w = call("GET", "/api/v1/history/"+ids[0], "", adminToken, ""); w.Code != 404 {
		t.Fatal("deleted record remains", w.Code)
	}
	w = call("GET", "/api/sessions", "", adminToken, "")
	var legacy []string
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &legacy) != nil || len(legacy) != 2 {
		t.Fatal("legacy contract or source sessions changed", w.Code, w.Body.String())
	}
}
