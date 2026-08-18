package app_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// seedSession writes a session directory. A completed session carries a
// session.json stamped at age; an in-progress one has none.
func seedSession(t *testing.T, root, id string, completedAt time.Time, done bool) {
	t.Helper()
	dir := session.SessionDir(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if !done {
		return
	}
	path := session.SessionFilePath(root, id)
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chtimes(path, completedAt, completedAt); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

// depsAt pins "now" so age policies are deterministic.
func depsAt(at time.Time) app.Deps {
	return app.Deps{Clock: func() time.Time { return at }}
}

func TestGCSessions_RemovesOnlyTheAgedOnes(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	seedSession(t, root, "20260801-000000-aaaa", now.Add(-17*24*time.Hour), true)
	seedSession(t, root, "20260817-000000-bbbb", now.Add(-24*time.Hour), true)

	out, err := app.GCSessions(context.Background(), depsAt(now), app.GCSessionsIn{
		Root: root, OlderThan: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("GCSessions: %v", err)
	}
	if len(out.Removed) != 1 || out.Removed[0] != "20260801-000000-aaaa" {
		t.Fatalf("removed = %v, want only the aged session", out.Removed)
	}
	if _, err := os.Stat(session.SessionDir(root, "20260817-000000-bbbb")); err != nil {
		t.Errorf("the recent session was removed: %v", err)
	}
}

func TestGCSessions_PreservesRunsInProgress(t *testing.T) {
	// A run that has not written its verdict must survive any policy: it is
	// still producing the artifacts the policy would be judging.
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	seedSession(t, root, "20260101-000000-live", now.Add(-200*24*time.Hour), false)

	out, err := app.GCSessions(context.Background(), depsAt(now), app.GCSessionsIn{
		Root: root, OlderThan: time.Hour,
	})
	if err != nil {
		t.Fatalf("GCSessions: %v", err)
	}
	if len(out.Removed) != 0 {
		t.Fatalf("removed = %v, want nothing", out.Removed)
	}
	if _, err := os.Stat(session.SessionDir(root, "20260101-000000-live")); err != nil {
		t.Errorf("in-progress session was removed: %v", err)
	}
}

func TestGCSessions_KeepLastProtectsTheNewest(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	for _, id := range []string{"20260801-000000-a", "20260802-000000-b", "20260803-000000-c"} {
		seedSession(t, root, id, now.Add(-30*24*time.Hour), true)
	}

	out, err := app.GCSessions(context.Background(), depsAt(now), app.GCSessionsIn{
		Root: root, KeepLast: 2,
	})
	if err != nil {
		t.Fatalf("GCSessions: %v", err)
	}
	if len(out.Removed) != 1 || out.Removed[0] != "20260801-000000-a" {
		t.Fatalf("removed = %v, want the oldest only", out.Removed)
	}
}

func TestGCSessions_RequiresAPolicy(t *testing.T) {
	// Without a policy the loop would delete everything, which a caller that
	// forgot a flag never meant.
	root := t.TempDir()
	seedSession(t, root, "20260801-000000-a", time.Now(), true)

	if _, err := app.GCSessions(context.Background(), app.Deps{}, app.GCSessionsIn{Root: root}); err == nil {
		t.Fatal("want an error when neither policy is set")
	}
	if _, err := os.Stat(session.SessionDir(root, "20260801-000000-a")); err != nil {
		t.Errorf("policyless call still removed a session: %v", err)
	}
}

func TestGCSessions_RequiresARoot(t *testing.T) {
	if _, err := app.GCSessions(context.Background(), app.Deps{}, app.GCSessionsIn{KeepLast: 1}); err == nil {
		t.Error("want an error without an artifact root")
	}
}
