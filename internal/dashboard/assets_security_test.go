package dashboard

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestAssetUploadRequiresCookieRoleAndCSRF(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	setup, err := os.ReadFile(filepath.Join(root, "setup.token"))
	if err != nil {
		t.Fatal(err)
	}
	admin, _, err := auth.Bootstrap("asset-admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: admin.User.ID, Role: "admin"}
	deployments, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifests, err := app.OpenManifestStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := NewServer(bus, nil, WithWebSecurity(auth, deployments), WithManifests(manifests, WebAuthenticator(auth), nil, ""))
	for _, role := range []string{"operator", "viewer"} {
		_, err = auth.CreateUser(actor, "asset-"+role, "long-user-password", role)
		if err != nil {
			t.Fatal(err)
		}
		session, token, err := auth.Login("asset-"+role, "long-user-password")
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct {
			csrf, origin string
			want         int
		}{{"", "", 403}, {session.CSRFToken, "http://foreign.example", 403}, {session.CSRFToken, "", 201}} {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			_ = writer.WriteField("kind", "template")
			file, err := writer.CreateFormFile("file", "genesis.json")
			if err != nil {
				t.Fatal(err)
			}
			_, _ = file.Write([]byte(`{"config":{"chainId":123}}`))
			_ = writer.Close()
			r := httptest.NewRequest("POST", "/api/v1/assets", &body)
			r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
			r.Header.Set("Content-Type", writer.FormDataContentType())
			r.Header.Set("X-CSRF-Token", tt.csrf)
			r.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, r)
			want := tt.want
			if role == "viewer" {
				want = 403
			}
			if w.Code != want {
				t.Fatalf("%s asset upload: %d %s", role, w.Code, w.Body.String())
			}
		}
	}
	items, err := manifests.Assets()
	if err != nil || len(items) != 1 {
		t.Fatal("rejected upload persisted", len(items), err)
	}
}
