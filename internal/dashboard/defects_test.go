package dashboard

import (
	"context"
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

// heldJobEngine keeps a job running until it is cancelled.
type heldJobEngine struct{ conflictFixtureEngine }

func (heldJobEngine) Execute(ctx context.Context, _ app.DeploymentActor, _ app.WebPreparedJob, _ func(app.WebJobPhase) error) (app.WebJobResult, error) {
	<-ctx.Done()
	return app.WebJobResult{NodeDisposition: "retained"}, ctx.Err()
}

func TestHistoryDeletionAndRoleChangesAreGuardedBeforeHandlers(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	jobsDir := filepath.Join(root, "jobs")
	jobs, err := app.OpenWebJobs(jobsDir, heldJobEngine{}, nil, auth.AuthorizeJobActor)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebJobs(jobs, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	admin, adminToken, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	operator, err := auth.CreateUser(app.DeploymentActor{ID: admin.User.ID, Role: "admin"}, "operator", "long-user-password", "operator")
	if err != nil {
		t.Fatal(err)
	}
	opSession, opToken, _ := auth.Login("operator", "long-user-password")
	adminSession, _ := auth.Session(adminToken)
	call := func(method, path, body, token, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		r.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	// No history route is registered here: only the middleware can refuse.
	if w := call("DELETE", "/api/v1/history/run-1", "", opToken, opSession.CSRFToken); w.Code != 403 {
		t.Errorf("an operator's history deletion reached routing: %d", w.Code)
	}
	actor := app.DeploymentActor{ID: operator.ID, Role: "operator"}
	plan, err := jobs.Plan(context.Background(), actor, app.WebPlanInput{WorkspaceID: "w", Operation: "chain.deploy"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := jobs.Start(context.Background(), actor, plan.ID, "k")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	// The job store cannot record the revocation: the role change must not
	// land without it.
	if err = os.Chmod(jobsDir, 0o500); err != nil {
		t.Fatal(err)
	}
	w := call("PATCH", "/api/v1/users/"+operator.ID, `{"role":"viewer"}`, adminToken, adminSession.CSRFToken)
	_ = os.Chmod(jobsDir, 0o700)
	if w.Code < 400 {
		t.Fatalf("a role change whose revocation failed succeeded: %d", w.Code)
	}
	for _, u := range auth.Users() {
		if u.ID == operator.ID && u.Role != "operator" {
			t.Fatalf("the role changed although its jobs were not revoked: %+v", u)
		}
	}
	if w := call("PATCH", "/api/v1/users/"+operator.ID, `{"role":"viewer"}`, adminToken, adminSession.CSRFToken); w.Code != 200 {
		t.Fatalf("retry: %d %s", w.Code, w.Body)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		v, _ := jobs.Get(job.ID)
		if v.CancelReason == "account_revoked" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the retried change did not revoke the job: %+v", v)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
