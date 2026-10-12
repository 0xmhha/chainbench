package dashboard_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

func TestManifestAPIUsesEngineValidationAndTrustedActor(t *testing.T) {
	store, err := app.OpenManifestStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	auth := func(r *http.Request) (app.DeploymentActor, error) {
		role := r.Header.Get("Test-Role")
		if role == "" {
			return app.DeploymentActor{}, errors.New("unauthenticated")
		}
		return app.DeploymentActor{ID: "account", Role: role}, nil
	}
	bus := collector.NewBus()
	defer bus.Close()
	server := dashboard.NewServer(bus, nil, dashboard.WithManifests(store, auth, nil, ""))
	request := func(method, path, body, role, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Test-Role", role)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/v1/manifests", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := request("GET", "/api/v1/manifests", "", "viewer", "")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var list struct {
		Items []app.ManagedManifest `json:"items"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Items) != 3 {
		t.Fatalf("%v %s", err, w.Body.String())
	}
	input := list.Items[0].ManifestInput
	var manifest map[string]any
	if err = json.Unmarshal(input.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["protocol"] = manifest["id"]
	manifest["id"] = "custom-network"
	input.Manifest, _ = json.Marshal(manifest)
	raw, _ := json.Marshal(input)
	for _, test := range []struct {
		role, origin string
		code         int
	}{{"viewer", "", 403}, {"operator", "https://foreign.example", 403}, {"operator", "", 201}} {
		w = request("POST", "/api/v1/manifests", string(raw), test.role, test.origin)
		if w.Code != test.code {
			t.Fatalf("%s %s: %d %s", test.role, test.origin, w.Code, w.Body.String())
		}
	}
	manifest["dialect"] = "unknown"
	input.Manifest, _ = json.Marshal(manifest)
	raw, _ = json.Marshal(input)
	w = request("POST", "/api/v1/manifests/validate", string(raw), "operator", "")
	if !strings.Contains(w.Body.String(), `"valid":false`) {
		t.Fatal(w.Body.String())
	}
	w = request("POST", "/api/v1/manifests", string(raw), "operator", "")
	if w.Code != 422 {
		t.Fatal(w.Code)
	}
	w = request("POST", "/api/v1/manifests/stablenet/setup", `{"assetId":"arbitrary","path":"/bin/sh"}`, "operator", "")
	if w.Code != 400 {
		t.Fatal("raw path accepted")
	}
}
