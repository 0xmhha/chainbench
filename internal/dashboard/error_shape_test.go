package dashboard

import (
	"context"
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
)

func TestErrorsCarryMachineCodesAndFieldPaths(t *testing.T) {
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
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebJobs(jobs, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	_, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	session, _ := auth.Session(token)
	call := func(method, path, body string, header map[string]string) (int, map[string]any) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		r.Header.Set("X-CSRF-Token", session.CSRFToken)
		for k, v := range header {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	code, body := call("POST", "/api/v1/documents", `{"kind":"server-set","name":"pool","contractVersion":"2","content":{"version":2,"pool":{"hosts":[],"slots":2}},"assetRefs":[]}`, nil)
	fields, _ := body["fieldErrors"].([]any)
	if code != 422 || body["code"] != "invalid" || len(fields) != 1 || fields[0].(map[string]any)["path"] != "/content/pool/hosts" {
		t.Errorf("invalid document: %d %v", code, body)
	}
	// Validation points at the same field the save would, for every kind.
	for want, doc := range map[string]string{
		"/content/pool/hosts/1/addr": `{"kind":"server-set","name":"pool","contractVersion":"2","content":{"version":2,"pool":{"hosts":["10.0.0.1","10.0.0.1."],"slots":1}},"assetRefs":[]}`,
		"/name":                      `{"kind":"workspace-config","name":" ","contractVersion":"2","content":{},"assetRefs":[]}`,
		"/content/steps/0/expect":    `{"kind":"case","name":"c","contractVersion":"2","content":{"schemaVersion":"2","kind":"case","id":"c","chainPreset":{"chain":"stablenet","binaries":{"default":"gstable"}},"steps":[{"expect":"unknown","is":1}]},"assetRefs":[]}`,
	} {
		code, body = call("POST", "/api/v1/documents/validate", doc, nil)
		issues, _ := body["errors"].([]any)
		if code != 200 || len(issues) != 1 || !strings.HasPrefix(issues[0].(map[string]any)["path"].(string), want) {
			t.Errorf("validate %s: %d %v", want, code, body)
		}
	}
	if code, body = call("GET", "/api/v1/documents/absent", "", nil); code != 404 || body["code"] != "not_found" {
		t.Errorf("absent document: %d %v", code, body)
	}
	actor := app.DeploymentActor{ID: session.User.ID, Role: "admin"}
	plan, err := jobs.Plan(context.Background(), actor, app.WebPlanInput{WorkspaceID: "w", Operation: "chain.setup"})
	if err != nil {
		t.Fatal(err)
	}
	if code, body = call("POST", "/api/v1/jobs", `{"planId":"`+plan.ID+`"}`, nil); code != 400 || body["code"] != "bad_request" {
		t.Errorf("job without an idempotency key: %d %v", code, body)
	}
}

func TestBusyStartNamesTheHolderOverHTTP(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), conflictFixtureEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertBusyStartNamesTheHolder(t, jobs)
}

func assertBusyStartNamesTheHolder(t *testing.T, jobs *app.WebJobs) {
	t.Helper()
	alice, bob := app.DeploymentActor{ID: "alice", Role: "operator"}, app.DeploymentActor{ID: "bob", Role: "operator"}
	owner, err := jobs.Plan(context.Background(), alice, app.WebPlanInput{WorkspaceID: "owner", Operation: "chain.setup"})
	if err != nil {
		t.Fatal(err)
	}
	held, err := jobs.Start(context.Background(), alice, owner.ID, "one")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deadline := time.Now().Add(3 * time.Second)
		for {
			job, err := jobs.Get(held.ID)
			if err != nil {
				t.Errorf("read accepted job before fixture cleanup: %v", err)
				return
			}
			// Get owns the same lock as the final durable commit. Observing
			// its terminal state means every disk write is already finished.
			if job.State == "succeeded" {
				return
			}
			if job.State == "failed" || job.State == "cancelled" {
				t.Errorf("fixture job ended %s: %v", job.State, job.Error)
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("fixture job did not finish before cleanup: %s", job.State)
				return
			}
			time.Sleep(time.Millisecond)
		}
	})
	alias, err := jobs.Plan(context.Background(), bob, app.WebPlanInput{WorkspaceID: "alias", Operation: "chain.setup"})
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithWebJobs(jobs, func(r *http.Request) (app.DeploymentActor, error) { return bob, nil }))
	r := httptest.NewRequest("POST", "/api/v1/jobs", strings.NewReader(`{"planId":"`+alias.ID+`"}`))
	r.Header.Set("Idempotency-Key", "two")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	var body struct {
		Message string `json:"message"`
		Details struct {
			Holders []app.WebResourceConflict `json:"holders"`
		} `json:"details"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 409 || len(body.Details.Holders) != 1 || body.Details.Holders[0].JobID != held.ID || body.Details.Holders[0].ActorID != "alice" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}

type delayedConflictEngine struct {
	conflictFixtureEngine
	entered, release, returned chan struct{}
}

func (e delayedConflictEngine) Execute(context.Context, app.DeploymentActor, app.WebPreparedJob, func(app.WebJobPhase) error) (app.WebJobResult, error) {
	close(e.entered)
	<-e.release
	// Represents slow work after the HTTP assertion has finished. Completion
	// must govern teardown, rather than how quickly this fixture usually runs.
	time.Sleep(50 * time.Millisecond)
	close(e.returned)
	return app.WebJobResult{NodeDisposition: "retained"}, nil
}

func TestBusyStartFinishesDiskWritesBeforeFixtureCleanup(t *testing.T) {
	engine := delayedConflictEngine{entered: make(chan struct{}), release: make(chan struct{}), returned: make(chan struct{})}
	var jobs *app.WebJobs
	var root string
	t.Run("slow execution", func(t *testing.T) {
		root = t.TempDir()
		var err error
		jobs, err = app.OpenWebJobs(root, engine, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer close(engine.release)
		assertBusyStartNamesTheHolder(t, jobs)
		select {
		case <-engine.entered:
		case <-time.After(3 * time.Second):
			t.Fatal("job never entered execution")
		}
	})
	// Even the engine's return is earlier than its final durable commit. The
	// scenario must drain that commit before testing.T removes its own TempDir.
	select {
	case <-engine.returned:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture execution did not return")
	}
	for _, job := range jobs.List() {
		if job.State != "succeeded" {
			t.Errorf("fixture directory reclaimed before final job commit: state %s", job.State)
		}
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("fixture directory was recreated after cleanup: %v", err)
	}
}

type missingAccessEngine struct{ conflictFixtureEngine }

func (missingAccessEngine) Prepare(context.Context, app.DeploymentActor, app.WebPlanInput) (app.WebPreparedJob, error) {
	return app.WebPreparedJob{}, &app.WebAccessError{Missing: []string{"ssh:server2"}}
}

func TestPlanWithoutSSHAccessNamesTheMissingServers(t *testing.T) {
	jobs, err := app.OpenWebJobs(t.TempDir(), missingAccessEngine{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	srv := NewServer(bus, nil, WithWebJobs(jobs, func(r *http.Request) (app.DeploymentActor, error) {
		return app.DeploymentActor{ID: "bob", Role: "operator"}, nil
	}))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/plans", strings.NewReader(`{"workspaceId":"w","operation":"chain.setup"}`)))
	var body struct {
		Details struct {
			MissingAccess []string `json:"missingAccess"`
		} `json:"details"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 422 || len(body.Details.MissingAccess) != 1 || body.Details.MissingAccess[0] != "ssh:server2" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}
