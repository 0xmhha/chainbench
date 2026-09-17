package testengine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// TestRecordBlockedBySetup_FilesTheFailureUnderTheTestThatAskedForIt.
//
// A network is composed for the test that asked for it, so a network that will
// not come up is that test's failure. It used to be nobody's: the evidence was
// dumped into the workspace under failures/<stamp>/ and the run reported "1 node
// still not ready" with no verdict at all, because the session was created by
// the engine and the engine had not started. A reader of the workspace dump then
// had to work out which of several attempts it belonged to.
func TestRecordBlockedBySetup_FilesTheFailureUnderTheTestThatAskedForIt(t *testing.T) {
	root := t.TempDir()
	sess, err := session.New(root, "chainbench", time.Date(2026, 9, 17, 4, 24, 48, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var out RunSuiteOut
	setupErr := errors.New("1 node(s) still not ready after 1m30s")

	recordBlockedBySetup(context.Background(), chainsetup.Deps{}, t.TempDir(), composed{}, &out, setupErr, blockedRun{
		sess:  sess,
		raw:   [][]byte{[]byte(`{"id":"gov-01"}`)},
		specs: []dsl.Spec{{ID: "gov-01"}},
	})

	dir := filepath.Join(sess.Root(), "tests", "001_gov-01")
	b, err := os.ReadFile(filepath.Join(dir, "status.json"))
	if err != nil {
		t.Fatalf("the blocked test left no verdict: %v", err)
	}
	if !strings.Contains(string(b), `"blocked"`) || !strings.Contains(string(b), "still not ready") {
		t.Errorf("the verdict does not say what happened: %s", b)
	}
	// The gather has no node table and no workspace here, so what it can say is
	// that it could not say anything — which is itself evidence, and has to land
	// in the same place the rest would.
	if _, err := os.Stat(filepath.Join(dir, "observations", "gather-problems.txt")); err != nil {
		t.Errorf("no evidence under the test: %v", err)
	}
	// And the run's own summary counts it, so the command reports a blocked test
	// rather than only an error string.
	if out.Summary.Summary.Blocked != 1 {
		t.Errorf("summary counted %d blocked, want 1", out.Summary.Summary.Blocked)
	}
}

// TestRecordBlockedBySetup_EveryWaitingTestGetsItsOwnCopy: several definitions
// given to one command are several attempts. Pointing all of them at one copy of
// the evidence would save space and lose the only thing that makes it readable
// later — which attempt it came from.
func TestRecordBlockedBySetup_EveryWaitingTestGetsItsOwnCopy(t *testing.T) {
	sess, err := session.New(t.TempDir(), "chainbench", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	var out RunSuiteOut
	recordBlockedBySetup(context.Background(), chainsetup.Deps{}, t.TempDir(), composed{}, &out, errors.New("no"), blockedRun{
		sess:  sess,
		raw:   [][]byte{[]byte(`{}`), []byte(`{}`)},
		specs: []dsl.Spec{{ID: "a"}, {ID: "b"}},
	})
	for _, name := range []string{"001_a", "002_b"} {
		if _, err := os.Stat(filepath.Join(sess.Root(), "tests", name, "observations", "gather-problems.txt")); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestRecordBlockedBySetup_NoSessionRecordsNothing: the failure paths that run
// before a session exists (and the tests that drive only the teardown decision)
// must not panic on the way past.
func TestRecordBlockedBySetup_NoSessionRecordsNothing(t *testing.T) {
	var out RunSuiteOut
	recordBlockedBySetup(context.Background(), chainsetup.Deps{}, t.TempDir(), composed{}, &out, errors.New("no"), blockedRun{})
	if len(out.SetupSteps) != 0 {
		t.Errorf("it recorded %v", out.SetupSteps)
	}
}

// TestArtifactRoot_LayersLikeEverythingElse.
//
// The workspace-config layer was written and never read: the file requires
// control.artifactRoot and nothing called the resolver, so an operator was told
// to name a path that was then ignored. The layers are the ones this track uses
// everywhere — harness default, then declaration, then invocation.
func TestArtifactRoot_LayersLikeEverythingElse(t *testing.T) {
	dir := t.TempDir()
	configured := filepath.Join(dir, "from-config")
	cfg := filepath.Join(dir, "workspace-config.yaml")
	writeConfig(t, cfg, configured)

	t.Run("no config, no flag: beside the workspace", func(t *testing.T) {
		got, err := artifactRoot("", "", "/ws")
		if err != nil || got != filepath.Join("/ws", "sessions") {
			t.Fatalf("artifactRoot = %q, %v", got, err)
		}
	})
	t.Run("the config is read", func(t *testing.T) {
		got, err := artifactRoot("", cfg, "/ws")
		if err != nil {
			t.Fatal(err)
		}
		if got != configured {
			t.Fatalf("artifactRoot = %q, want the configured %q", got, configured)
		}
		// It is created, because a run that cannot write there has to fail here
		// and not halfway through its first test.
		if _, err := os.Stat(configured); err != nil {
			t.Errorf("the configured root was not created: %v", err)
		}
	})
	t.Run("the invocation wins", func(t *testing.T) {
		got, err := artifactRoot("/explicit", cfg, "/ws")
		if err != nil || got != "/explicit" {
			t.Fatalf("artifactRoot = %q, %v", got, err)
		}
	})
}

// TestArtifactRoot_AnUnusableConfiguredRootIsAnError: falling back would put the
// results somewhere the operator did not ask for and say nothing, and they would
// go looking in the path they wrote.
func TestArtifactRoot_AnUnusableConfiguredRootIsAnError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "workspace-config.yaml")
	writeConfig(t, cfg, filepath.Join(blocker, "under-a-file"))

	if _, err := artifactRoot("", cfg, "/ws"); err == nil {
		t.Fatal("a configured root that cannot be created must fail, not fall back")
	}
}

// writeConfig writes the smallest workspace-config that parses, with the
// artifact root under test.
func writeConfig(t *testing.T, path, artifactRoot string) {
	t.Helper()
	body := `version: 1
dataRoot: /data
paths:
  binaries: bin
  configs: configs
  genesis: genesis
  keystore: keystore
  keyrings: keys
  nodes: node
  runtime: runtime
  logs: logs
inputs:
  mode: generated
execution:
  chain: fresh
control:
  artifactRoot: ` + artifactRoot + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
