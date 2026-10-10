package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestPublisherCannotUseControlOrDiscloseItsToken(t *testing.T) {
	const token = "publisher-test-capability-with-32-bytes"
	bus := collector.NewBus()
	defer bus.Close()
	store, err := app.OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(bus, nil, WithDeployments(store, func(*http.Request) (app.DeploymentActor, error) {
		return app.DeploymentActor{}, app.ErrDeploymentForbidden
	}), WithPublisherToken(token))
	sub, cancel := bus.SubscribeWithCancel()
	defer cancel()
	for _, entry := range []struct {
		path, bearer string
		status       int
	}{
		{"/api/events", "wrong", 401}, {"/api/v1/documents", token, 401}, {"/api/events", token, 202},
	} {
		r := httptest.NewRequest("POST", entry.path, strings.NewReader(`{"kind":"info","message":"`+token+`","fields":{"password":"secret-field"}}`))
		r.Header.Set("Authorization", "Bearer "+entry.bearer)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if w.Code != entry.status {
			t.Fatalf("%s %d: %s", entry.path, w.Code, w.Body.String())
		}
	}
	event := <-sub
	if strings.Contains(event.Message, token) || event.Fields["password"] == "secret-field" {
		t.Fatal("publisher secrets reached the bus")
	}
}

func TestUIRefreshDoesNotHideMissingAPIOrAssets(t *testing.T) {
	srv := NewServer(collector.NewBus(), nil)
	for _, path := range []string{"/", "/chains", "/tests", "/monitoring", "/history", "/settings"} {
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "/assets/") {
			t.Fatalf("%s missing current shell", path)
		}
	}
	for _, path := range []string{"/api/v1/missing", "/assets/missing.js", "/unknown-route"} {
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 || strings.Contains(w.Body.String(), "/assets/") {
			t.Fatalf("%s masked 404", path)
		}
	}
}
