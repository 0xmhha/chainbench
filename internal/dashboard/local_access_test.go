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

func TestLocalBrowserRequestsRefuseRemotePeersAndRebinding(t *testing.T) {
	for _, tc := range []struct {
		host, peer string
		allowed    bool
	}{
		{"localhost:8787", "127.0.0.1:4000", true},
		{"127.0.0.1:8787", "127.0.0.1:4000", true},
		{"[::1]:8787", "[::1]:4000", true},
		{"localhost", "[::ffff:127.0.0.1]:4000", true},
		{"hostile.example:8787", "127.0.0.1:4000", false},
		{"127.0.0.1:8787", "192.0.2.1:4000", false},
		{"127.0.0.1:8787", "invalid", false},
	} {
		r := httptest.NewRequest(http.MethodPost, "http://localhost/api/v1/auth/local", nil)
		r.Host, r.RemoteAddr = tc.host, tc.peer
		if got := localBrowserRequest(r); got != tc.allowed {
			t.Errorf("host=%q peer=%q: allowed=%v", tc.host, tc.peer, got)
		}
	}
}

// A same-host reverse proxy turns every remote browser into a loopback peer,
// and the Host it forwards can be loopback too. Its forwarding headers are the
// one trace left, so a request carrying any of them is not a local browser.
func TestLocalBrowserRequestsRefuseProxyForwardedRequests(t *testing.T) {
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"} {
		r := httptest.NewRequest(http.MethodPost, "http://localhost/api/v1/auth/local", nil)
		r.Host, r.RemoteAddr = "127.0.0.1:8787", "127.0.0.1:4000"
		r.Header.Set(header, "203.0.113.9")
		if localBrowserRequest(r) {
			t.Errorf("%s: a proxied request was treated as a local browser", header)
		}
	}
}

func TestLocalBrowserAccessKeepsControlsCSRFAndAudit(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenLocalWebAuth(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := app.OpenWebJobs(root, conflictFixtureEngine{}, store.RedactWeb, auth.AuthorizeJobActor)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithWebSecurity(auth, store), WithDeployments(store, WebAuthenticator(auth)), WithWebJobs(jobs, WebAuthenticator(auth)), WithWebAudit(auth, store, jobs, nil, nil))
	call := func(method, path, body, token, csrf, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:4000"
		if token != "" {
			r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		}
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		return w
	}
	config := call("GET", "/api/v1/auth/config", "", "", "", "")
	var options struct{ AccountsEnabled, BootstrapRequired bool }
	if err := json.Unmarshal(config.Body.Bytes(), &options); err != nil || config.Code != 200 || options.AccountsEnabled || options.BootstrapRequired {
		t.Fatal("account-free config", config.Code, err)
	}
	if w := call("POST", "/api/v1/auth/local", "", "", "", "http://hostile.example"); w.Code != 403 {
		t.Fatal("cross-site local session issued")
	}
	login := call("POST", "/api/v1/auth/local", "", "", "", "http://localhost")
	var session app.WebSession
	if err := json.Unmarshal(login.Body.Bytes(), &session); err != nil || login.Code != 200 || session.User.ID == "" || session.AccountsEnabled {
		t.Fatal("automatic local browser access", login.Code, err)
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("local browser cookie lacks protection")
	}
	token := cookies[0].Value
	if w := call("POST", "/api/v1/auth/local", "", token, "", ""); w.Code != 200 || len(w.Result().Cookies()) != 0 {
		t.Fatal("refresh did not reuse the local browser session")
	}
	for _, path := range []string{"/api/runs", "/api/v1/auth/me", "/api/v1/contracts/access", "/api/v1/documents", "/api/v1/credentials", "/api/v1/snapshot", "/api/v1/audit"} {
		if w := call("GET", path, "", token, "", ""); w.Code != 200 {
			t.Fatalf("local read %s: %d", path, w.Code)
		}
	}
	for _, tc := range []struct{ method, path string }{{"POST", "/api/v1/bootstrap"}, {"POST", "/api/v1/auth/login"}, {"POST", "/api/v1/auth/logout"}, {"GET", "/api/v1/users"}, {"PATCH", "/api/v1/users/unused"}} {
		if w := call(tc.method, tc.path, "{}", token, session.CSRFToken, ""); w.Code != 404 {
			t.Fatalf("account-only endpoint %s: %d", tc.path, w.Code)
		}
	}
	body := `{"label":"local credential","kind":"password","sshUser":"ssh","password":"local-redaction-marker"}`
	for _, tc := range []struct{ csrf, origin string }{{"", ""}, {session.CSRFToken, "http://hostile.example"}} {
		if w := call("POST", "/api/v1/credentials", body, token, tc.csrf, tc.origin); w.Code != 403 {
			t.Fatal("unprotected write accepted", w.Code)
		}
	}
	saved := call("POST", "/api/v1/credentials", body, token, session.CSRFToken, "http://localhost")
	if saved.Code != 201 || strings.Contains(saved.Body.String(), "local-redaction-marker") {
		t.Fatal("credential write failed or disclosed its secret", saved.Code)
	}
	actor := app.DeploymentActor{ID: session.User.ID, Role: "admin"}
	if err := auth.AuthorizeJobActor(actor); err != nil || len(store.Credentials(actor)) != 1 {
		t.Fatal("local execution actor cannot use its credential", err)
	}
	audit := call("GET", "/api/v1/audit", "", token, "", "")
	if audit.Code != 200 || !strings.Contains(audit.Body.String(), session.User.ID) || strings.Contains(audit.Body.String(), "local-redaction-marker") {
		t.Fatal("missing or unsafe local audit", audit.Code)
	}
	if w := call("POST", "/api/v1/plans", `{"workspaceId":"local-workspace","operation":"chain.setup"}`, token, session.CSRFToken, ""); w.Code != 201 {
		t.Fatal("local plan refused", w.Code, w.Body.String())
	}
}
