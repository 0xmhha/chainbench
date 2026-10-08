package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func historyFixture(t *testing.T, artifactRoot string, second int, status session.TestStatus) (string, string) {
	t.Helper()
	s, err := session.New(artifactRoot, "test fixture", time.Date(2026, 10, 8, 1, 0, second, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.NewEnvironment(session.Fingerprint(strings.Repeat("a", 64)))
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Save(); err != nil {
		t.Fatal(err)
	}
	r := s.Test(1, "shared-case")
	r.SetEnvRef(e.ID())
	r.Status(status)
	r.Assert(session.AssertResult{ID: "block", Assert: "height", Expected: 1, Actual: 2, Pass: status == session.StatusPass})
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	// Historical artifacts may precede today's scrubber. Structured redaction
	// must handle quoted secret values without corrupting the captured JSON.
	if err = os.WriteFile(filepath.Join(r.Dir(), "spec.json"), []byte(`{"key":"escaped\"private","mnemonic":"do not export these words","note":"registered-history-secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(r.Dir(), "steps.json"), []byte(`[{"nonce":18446744073709551615,"block":9007199254740993}]`), 0600); err != nil {
		t.Fatal(err)
	}
	return session.IDFor(artifactRoot, s.Root()), s.Root()
}

func TestWebHistoryCaptureCompareDeleteAndRestart(t *testing.T) {
	ctx, root, artifacts := context.Background(), t.TempDir(), t.TempDir()
	ref1, original1 := historyFixture(t, artifacts, 1, session.StatusPass)
	ref2, original2 := historyFixture(t, artifacts, 2, session.StatusFail)
	deployments, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "admin", Role: "admin"}
	if _, err = deployments.SaveCredential(a, DeploymentCredentialInput{Label: "test", Kind: "password", SSHUser: "test", Password: "registered-history-secret"}); err != nil {
		t.Fatal(err)
	}
	jobs, err := OpenWebJobs(root, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(webChainPayload{Binary: ManifestBinaryEvidence{Chain: "wbft", SHA256: strings.Repeat("b", 64)}})
	for i, ref := range []string{ref1, ref2} {
		id := []string{"one", "two"}[i]
		jobs.state.Plans[id] = savedWebPlan{Prepared: WebPreparedJob{Fingerprint: strings.Repeat("c", 64), Payload: payload}}
		jobs.state.Jobs[id] = WebJob{ID: id, ActorID: a.ID, WorkspaceID: "workspace", PlanID: id, State: "succeeded", CreatedAt: time.Now(), RunIDs: []string{ref}}
	}
	jobs.state.Jobs["active"] = WebJob{ID: "active", PlanID: "one", State: "running"}
	history, err := OpenWebHistory(root, artifacts, jobs, deployments.RedactWeb)
	if err != nil {
		t.Fatal(err)
	}
	if err = history.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := history.List(WebHistoryQuery{CaseID: "shared-case", Chain: "wbft", WorkspaceID: "workspace", ActorID: a.ID, Limit: 1})
	if err != nil || len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatal(page, err)
	}
	page2, err := history.List(WebHistoryQuery{CaseID: "shared-case", Chain: "wbft", WorkspaceID: "workspace", ActorID: a.ID, Limit: 1, Cursor: *page.NextCursor})
	if err != nil || len(page2.Items) != 1 || page2.NextCursor != nil || page2.Items[0].ID == page.Items[0].ID {
		t.Fatal(page2, err)
	}
	if _, err = history.List(WebHistoryQuery{State: "failed", Cursor: historySessionID(ref1)}); err == nil {
		t.Fatal("cursor from a different filtered result accepted")
	}
	id1, id2 := historySessionID(ref1), historySessionID(ref2)
	comparison, err := history.Compare([]string{id1, id2})
	if err != nil || !comparison.Comparable || len(comparison.Results) != 1 {
		t.Fatal(comparison, err)
	}
	statuses := comparison.Results[0]["statuses"].(map[string]string)
	if statuses[id1] != "pass" || statuses[id2] != "fail" {
		t.Fatal(statuses)
	}
	data, err := history.Export(id1)
	if err != nil || !json.Valid(data) {
		t.Fatal(string(data), err)
	}
	for _, secret := range []string{"private", "do not export these words", "registered-history-secret"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("secret exported: %s", secret)
		}
	}
	var exported webHistoryCapture
	_ = json.Unmarshal(data, &exported)
	if !strings.Contains(string(data), "18446744073709551615") || !strings.Contains(string(data), "9007199254740993") {
		t.Fatal("large integer evidence rounded during capture/export", string(data))
	}
	for name, content := range exported.Files {
		if strings.HasSuffix(name, ".json") && !json.Valid([]byte(content)) {
			t.Fatalf("redaction corrupted %s: %s", name, content)
		}
	}
	for _, role := range []string{"viewer", "operator"} {
		if !errors.Is(history.Delete(DeploymentActor{ID: role, Role: role}, id1), ErrDeploymentForbidden) {
			t.Fatal("non-administrator deleted history")
		}
	}
	if !errors.Is(history.Delete(a, "job-active"), ErrDeploymentConflict) {
		t.Fatal("active job deleted")
	}
	if err = history.Delete(a, id1); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{original1, original2} {
		if _, err = os.Stat(filepath.Join(dir, "session.json")); err != nil {
			t.Fatal("source engine artifacts deleted", err)
		}
	}
	if err = history.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = history.Get(id1); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatal("deleted capture resurrected", err)
	}
	if len(history.state.Audit) != 1 || history.state.Audit[0].ActorID != a.ID || history.state.Audit[0].RunID != id1 {
		t.Fatal("deletion audit missing")
	}
	if err = os.RemoveAll(original2); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWebHistory(root, artifacts, nil, deployments.RedactWeb)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.Get(id2); err != nil {
		t.Fatal("captured result lost after source removal/restart", err)
	}
	afterRestart, err := reopened.Export(id2)
	if err != nil || !strings.Contains(string(afterRestart), "18446744073709551615") || !strings.Contains(string(afterRestart), "9007199254740993") {
		t.Fatal("large integer evidence rounded after restart", err)
	}
	if _, err = reopened.Get(id1); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatal("tombstone lost after restart")
	}
	if _, err = reopened.Compare([]string{id2, "job-one"}); err != nil {
		t.Fatal(err)
	}
	comparison, _ = reopened.Compare([]string{id2, "job-one"})
	if comparison.Comparable || len(comparison.Limitations) < 2 {
		t.Fatal("job without test results claimed comparable", comparison)
	}
}

type failingHistoryWrite struct{ historyStorage }

func (failingHistoryWrite) Write([]byte) error { return errors.New("storage unavailable") }

func TestWebHistoryDeletionWriteFailurePreservesEvidence(t *testing.T) {
	root, artifacts := t.TempDir(), t.TempDir()
	ref, _ := historyFixture(t, artifacts, 3, session.StatusPass)
	history, err := OpenWebHistory(root, artifacts, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = history.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	id := historySessionID(ref)
	history.files = failingHistoryWrite{history.files}
	if err = history.Delete(DeploymentActor{ID: "admin", Role: "admin"}, id); err == nil {
		t.Fatal("failed deletion reported success")
	}
	if _, err = history.Get(id); err != nil || history.state.Deleted[id] || len(history.state.Audit) != 0 {
		t.Fatal("failed atomic deletion changed evidence or tombstone", err)
	}
	reopened, err := OpenWebHistory(root, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.Get(id); err != nil {
		t.Fatal("prior disk snapshot lost", err)
	}
}
