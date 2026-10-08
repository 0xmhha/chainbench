package dashboard_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

// Exercise the public multipart contract, rather than a storage implementation.
func TestUploadedAssetsAreImmutableSharedMetadata(t *testing.T) {
	root := t.TempDir()
	bus := collector.NewBus()
	defer bus.Close()
	auth := func(r *http.Request) (app.DeploymentActor, error) {
		role := r.Header.Get("Test-Role")
		if role == "" {
			return app.DeploymentActor{}, errors.New("login required")
		}
		return app.DeploymentActor{ID: "alice", Role: role}, nil
	}
	open := func() *dashboard.Server {
		store, err := app.OpenManifestStore(root)
		if err != nil {
			t.Fatal(err)
		}
		return dashboard.NewServer(bus, nil, dashboard.WithManifests(store, auth, nil, ""))
	}
	server := open()
	call := func(method, path, kind, name, body, role, origin string) *httptest.ResponseRecorder {
		var data bytes.Buffer
		writer := multipart.NewWriter(&data)
		if method == "POST" {
			_ = writer.WriteField("kind", kind)
			file, err := writer.CreateFormFile("file", name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = file.Write([]byte(body))
		}
		_ = writer.Close()
		r := httptest.NewRequest(method, path, &data)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		r.Header.Set("Test-Role", role)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	for _, tt := range []struct {
		role, origin string
		status       int
	}{{"", "", 401}, {"viewer", "", 403}, {"operator", "http://foreign.example", 403}} {
		w := call("POST", "/api/v1/assets", "configuration", "settings.json", `{"chainId":99}`, tt.role, tt.origin)
		if w.Code != tt.status {
			t.Fatalf("upload authority %s: %d %s", tt.role, w.Code, w.Body.String())
		}
	}
	body := `{"chainId":99}`
	w := call("POST", "/api/v1/assets", "configuration", "settings.json", body, "operator", "")
	if w.Code != 201 {
		t.Fatalf("registered upload: %d %s", w.Code, w.Body.String())
	}
	var asset struct {
		ID, Kind, Name, Checksum, UploaderID string
		Bytes                                int64
	}
	if err := json.Unmarshal(w.Body.Bytes(), &asset); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(body))
	if asset.ID == "" || asset.Kind != "configuration" || asset.Name != "settings.json" || asset.Checksum != hex.EncodeToString(sum[:]) || asset.Bytes != int64(len(body)) || asset.UploaderID != "alice" {
		t.Fatal(w.Body.String())
	}
	if strings.Contains(w.Body.String(), root) || strings.Contains(w.Body.String(), body) {
		t.Fatal("private path or raw content exposed")
	}
	server = open() // A new owner reads persisted receipts and checks the immutable bytes.
	for _, path := range []string{"/api/v1/assets", "/api/v1/assets/" + asset.ID} {
		w = call("GET", path, "", "", "", "viewer", "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), asset.Checksum) {
			t.Fatalf("persisted metadata: %d %s", w.Code, w.Body.String())
		}
	}
	for _, tt := range []struct{ kind, name, body string }{{"shell", "script.sh", "echo unsafe"}, {"binary", "node", "not a native binary"}, {"configuration", "settings.json", "{"}, {"configuration", "secret.json", `{"ssh":{"password":"private"}}`}, {"material", "key.pem", "-----BEGIN PRIVATE KEY-----"}, {"template", "empty.json", ""}} {
		w = call("POST", "/api/v1/assets", tt.kind, tt.name, tt.body, "operator", "")
		if w.Code != 422 {
			t.Fatalf("invalid %s %s accepted: %d %s", tt.kind, tt.name, w.Code, w.Body.String())
		}
	}
	if w := call("GET", "/api/v1/assets/missing", "", "", "", "viewer", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
}
