package dashboard

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestWebSecuritySessionsRolesAndRedaction(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithChainPresets("../../presets/chain"), WithDeployments(store, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	call := func(method, path, body, token, csrf, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		}
		r.Header.Set("X-CSRF-Token", csrf)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/runs", "", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	if w := call("POST", "/api/v1/bootstrap", `{"username":"admin","password":"long-admin-password","setupToken":"wrong"}`, "", "", ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "long-admin-password", "setupToken": string(setup)})
	w := call("POST", "/api/v1/bootstrap", string(body), "", "", "")
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var session app.WebSession
	_ = json.Unmarshal(w.Body.Bytes(), &session)
	token := w.Result().Cookies()[0].Value
	if !w.Result().Cookies()[0].HttpOnly || w.Result().Cookies()[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	if w := call("POST", "/api/v1/bootstrap", string(body), "", "", ""); w.Code != 409 {
		t.Fatal(w.Code)
	}
	for _, role := range []string{"administrator", "operator", "viewer"} {
		a := app.DeploymentActor{ID: session.User.ID, Role: "admin"}
		u, err := auth.CreateUser(a, "user-"+role, "long-user-password", role)
		if err != nil {
			t.Fatal(err)
		}
		ss, tt, err := auth.Login(u.Username, "long-user-password")
		if err != nil {
			t.Fatal(err)
		}
		want := 403
		if role == "administrator" {
			want = 200
		}
		if w := call("GET", "/api/v1/users", "", tt, "", ""); w.Code != want {
			t.Fatalf("%s users %d", role, w.Code)
		}
		want = 400
		if role == "viewer" {
			want = 403
		}
		if w := call("POST", "/api/v1/documents", "{", tt, ss.CSRFToken, ""); w.Code != want {
			t.Fatalf("%s write %d", role, w.Code)
		}
		if w := call("POST", "/api/v1/documents", "{}", tt, "", ""); w.Code != 403 {
			t.Fatal("missing CSRF accepted")
		}
		if w := call("POST", "/api/v1/documents", "{}", tt, ss.CSRFToken, "http://evil.example"); w.Code != 403 {
			t.Fatal("cross origin accepted")
		}
	}
	actor := app.DeploymentActor{ID: session.User.ID, Role: "admin"}
	c, err := store.SaveCredential(actor, app.DeploymentCredentialInput{Label: "key", Kind: "password", SSHUser: "ssh", Password: "unique-redaction-secret"})
	if err != nil {
		t.Fatal(err)
	}
	event := `{"kind":"info","message":"unique-redaction-secret","fields":{"password":"another-secret"}}`
	if w := call("POST", "/api/events", event, token, session.CSRFToken, ""); w.Code != 202 {
		t.Fatal(w.Code)
	}
	// Check the same response wrapper used by SSE, JSON export, and legacy reads.
	out := httptest.NewRecorder()
	rw := &secureResponse{ResponseWriter: out, status: 200, redact: store.RedactWeb}
	_, _ = rw.Write([]byte(event))
	if strings.Contains(out.Body.String(), "unique-redaction-secret") || strings.Contains(out.Body.String(), "another-secret") {
		t.Fatal(out.Body.String())
	}
	foreign, ft, _ := auth.Login("user-operator", "long-user-password")
	if w := call("GET", "/api/v1/credentials/"+c.ID, "", ft, "", ""); w.Code != 404 {
		t.Fatal("foreign credential exposed", w.Code)
	}
	if w := call("POST", "/api/v1/auth/logout", "", ft, foreign.CSRFToken, ""); w.Code != 204 {
		t.Fatal(w.Code)
	}
	if w := call("GET", "/api/v1/auth/me", "", ft, "", ""); w.Code != 401 {
		t.Fatal("logged-out session accepted")
	}
	// A restart retains accounts and encrypted material but invalidates browser tokens.
	restarted, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.Session(token); err == nil {
		t.Fatal("session survived restart")
	}
	if _, _, err = restarted.Login("admin", "long-admin-password"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "users.json"))
	if strings.Contains(string(b), "long-admin-password") {
		t.Fatal("plaintext account storage")
	}
}

func TestWebSecurityClosesRevokedStream(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "personal")
	if err != nil {
		t.Fatal(err)
	}
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	_, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := httptest.NewServer(NewServer(bus, nil, WithWebSecurity(auth, store)))
	defer server.Close()
	req, _ := http.NewRequest("GET", server.URL+"/events", nil)
	req.AddCookie(&http.Cookie{Name: webCookie, Value: token})
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	auth.Logout(token)
	b, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal("revoked SSE did not close", err)
	}
	if len(b) != 0 {
		t.Fatal("revoked session received event")
	}
}

func TestWebSecurityRedactsActualSSE(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "personal")
	if err != nil {
		t.Fatal(err)
	}
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	session, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.SaveCredential(app.DeploymentActor{ID: session.User.ID, Role: "admin"}, app.DeploymentCredentialInput{Label: "sentinel", Kind: "password", SSHUser: "ssh", Password: "unique-stream-secret"})
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := httptest.NewServer(NewServer(bus, nil, WithWebSecurity(auth, store)))
	defer server.Close()
	req, _ := http.NewRequest("GET", server.URL+"/events", nil)
	req.AddCookie(&http.Cookie{Name: webCookie, Value: token})
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal(response.Header)
	}
	bus.Publish(collector.Event{Kind: collector.KindInfo, Message: "unique-stream-secret"})
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err, line)
	}
	var event collector.Event
	if err = json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data: "))), &event); err != nil {
		t.Fatal(err, line)
	}
	if event.Message != "[REDACTED]" {
		t.Fatal(line)
	}
}

func TestWebSecurityShowsRedactedValidationReasonsOnly(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	session, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	const secret = "registered-validation-secret-7731"
	if _, err = store.SaveCredential(app.DeploymentActor{ID: session.User.ID, Role: "admin"}, app.DeploymentCredentialInput{Label: "private", Kind: "password", SSHUser: "ops", Password: secret}); err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if authenticated {
			r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
			r.Header.Set("X-CSRF-Token", session.CSRFToken)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	type errorBody struct{ Code, Message, RequestID string }
	read := func(w *httptest.ResponseRecorder) errorBody {
		var b errorBody
		if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
			t.Fatalf("error body is not JSON: %q", w.Body.String())
		}
		return b
	}
	invalid := `{"kind":"server-set","name":"bad","contractVersion":"2","assetRefs":[],"content":{"version":2,"` + secret + `":1,"pool":{"hosts":[{"name":"local","addr":"127.0.0.1"}],"slots":1}}}`
	w := call("POST", "/api/v1/documents", invalid, true)
	if w.Code != 422 {
		t.Fatalf("invalid document status %d: %s", w.Code, w.Body.String())
	}
	b := read(w)
	if b.Message == "Unprocessable Entity" || b.Message == "" {
		t.Fatalf("validation reason hidden: %+v", b)
	}
	if strings.Contains(w.Body.String(), secret) {
		t.Fatalf("validation message leaked a registered secret: %s", w.Body.String())
	}
	if w := call("POST", "/api/v1/documents", invalid, false); w.Code != 401 || read(w).Message != "Unauthorized" {
		t.Fatalf("authentication failure must stay generic: %d %s", w.Code, w.Body.String())
	}
}

func TestWebErrorsExplainOnlyRefusedChangesNotServerFailures(t *testing.T) {
	w := httptest.NewRecorder()
	deploymentError(w, &fs.PathError{Op: "open", Path: "/private/state/deployment.json", Err: errors.New("no space left on device")})
	if w.Code != 500 {
		t.Fatalf("storage failure reported as %d", w.Code)
	}
	identity := func(s string) string { return s }
	body := string(webErrorBody(500, []byte("open /private/state/deployment.json: no space left"), identity, true))
	if strings.Contains(body, "/private/state") {
		t.Fatalf("server failure text exposed: %s", body)
	}
	if body := string(webErrorBody(422, []byte("pool.slots must be at least 1"), identity, false)); strings.Contains(body, "pool.slots") {
		t.Fatalf("a read request exposed a validation reason: %s", body)
	}
	if body := string(webErrorBody(422, []byte(`{"message":{"nested":true},"errors":[{"message":"bad"}]}`), identity, true)); !strings.Contains(body, `"message":"Unprocessable Entity"`) || !strings.Contains(body, `"bad"`) {
		t.Fatalf("non-string message or listed errors mishandled: %s", body)
	}
}
