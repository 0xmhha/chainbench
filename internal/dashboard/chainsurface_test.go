package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestChainSurfaceRouteServesTheCatalogToEveryRole(t *testing.T) {
	root := t.TempDir()
	documents, err := app.OpenDeploymentStore(filepath.Join(root, "documents"))
	if err != nil {
		t.Fatal(err)
	}
	engine := app.NewWebChainEngine(root, "", documents, nil, nil, nil)
	engine.UseChainSurfaces(filepath.Join("..", "..", "docs", "chain-analysis"))
	bus := collector.NewBus()
	defer bus.Close()
	server := NewServer(bus, nil, WithChainSurfaces(engine, func(r *http.Request) (app.DeploymentActor, error) {
		if r.Header.Get("fixture-role") == "" {
			return app.DeploymentActor{}, errors.New("unauthenticated")
		}
		return app.DeploymentActor{ID: "reader", Role: r.Header.Get("fixture-role")}, nil
	}))
	call := func(role, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if role != "" {
			r.Header.Set("fixture-role", role)
		}
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	w := call("viewer", "/api/v1/chains/wemix/surface")
	if w.Code != 200 {
		t.Fatalf("viewer read %d %s", w.Code, w.Body)
	}
	var s app.WebChainSurface
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ChainID != "wemix" || s.Dialect != "geth110-wemix" || s.Verification != "documented" || len(s.Options) == 0 || len(s.Commands) == 0 {
		t.Fatalf("surface %s %s %s", s.ChainID, s.Dialect, s.Verification)
	}
	for _, c := range []struct {
		role, path string
		want       int
	}{
		{"viewer", "/api/v1/chains/unknown/surface", 404},
		{"viewer", "/api/v1/chains/stablenet/surface?assetId=absent", 403},
		{"operator", "/api/v1/chains/stablenet/surface?assetId=absent", 404},
	} {
		if got := call(c.role, c.path).Code; got != c.want {
			t.Errorf("%s %s: %d, want %d", c.role, c.path, got, c.want)
		}
	}
	if got := call("", "/api/v1/chains/stablenet/surface").Code; got != 401 {
		t.Errorf("unauthenticated read %d", got)
	}
}
