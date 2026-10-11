package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestContractRouteNamesTheContractsThisServerServes(t *testing.T) {
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithChainPresets(t.TempDir()))
	for path, want := range map[string]int{
		"/api/v1/contracts/chain-preset?chain=stablenet": 200,
		"/api/v1/contracts/unknown":                      404,
	} {
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != want {
			t.Errorf("%s: %d, want %d", path, w.Code, want)
		}
		if want == 404 && !strings.Contains(w.Body.String(), "chain-preset") {
			t.Errorf("%s does not name the contracts that exist: %q", path, w.Body.String())
		}
	}
}

type networkFixtureEngine struct{ conflictFixtureEngine }

func (networkFixtureEngine) Networks(context.Context) ([]app.WebNetwork, error) {
	return []app.WebNetwork{{ID: "w1", WorkspaceID: "w1", Ownership: "owned", Nodes: []app.WebNode{}}, {ID: "w2", WorkspaceID: "w2", Ownership: "owned", Nodes: []app.WebNode{}}}, nil
}

func TestWorkspaceNetworksListOnlyThatWorkspace(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), networkFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithWebJobs(jobs, func(r *http.Request) (app.DeploymentActor, error) {
		if r.Header.Get("fixture-actor") == "" {
			return app.DeploymentActor{}, errors.New("unauthenticated")
		}
		return app.DeploymentActor{ID: "reader", Role: "viewer"}, nil
	}))
	r := httptest.NewRequest("GET", "/api/v1/workspaces/w2/networks", nil)
	r.Header.Set("fixture-actor", "reader")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	var page struct {
		Items      []app.WebNetwork `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != 1 || page.Items[0].WorkspaceID != "w2" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}
