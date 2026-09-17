package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSaveFailureData_LeavesEvidenceWhereThereIsNoSession is the point of
// splitting gathering from writing.
//
// A run that fails before its first test has no session and no test record, so
// the evidence used to have nowhere to go and was dropped. It reported "1 node
// still not ready" and threw away the only thing that could say which node.
func TestSaveFailureData_LeavesEvidenceWhereThereIsNoSession(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 9, 17, 4, 24, 48, 0, time.UTC)

	got, err := saveFailureData(func() time.Time { return at }, dir, []evidence{
		{Name: "health.json", Data: []byte(`{"producing":false}`)},
		{Name: "node1.log", Data: []byte("Fatal: something\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The directory is stamped so two failed attempts on one workspace do not
	// overwrite each other.
	if want := filepath.Join(dir, failureDir, "20260917-042448"); got != want {
		t.Fatalf("evidence went to %q, want %q", got, want)
	}
	for _, name := range []string{"health.json", "node1.log"} {
		if _, err := os.Stat(filepath.Join(got, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestSaveFailureData_ScrubsSecrets: the evidence holds a node's launch command,
// and that command carries a password path and an unlock address. It is written
// through the same scrubber a test record uses, so the two paths cannot disagree
// about what counts as a secret.
func TestSaveFailureData_ScrubsSecrets(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`gstable --password /k/password --unlock 0xc17d493883eaa3b4cceb0f214b273392d562f9d8`)

	got, err := saveFailureData(nil, dir, []evidence{{Name: "processes.json", Data: raw}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(got, "processes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "/k/password") {
		t.Errorf("the password path reached the evidence: %s", b)
	}
}

// TestSaveFailureData_NothingGatheredWritesNothing: an empty gather is not a
// failure to report. A workspace that never composed has nothing to say, and a
// directory holding no files would only make an operator look for one.
func TestSaveFailureData_NothingGatheredWritesNothing(t *testing.T) {
	dir := t.TempDir()
	got, err := saveFailureData(nil, dir, nil)
	if err != nil || got != "" {
		t.Fatalf("saveFailureData(nil) = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, failureDir)); !os.IsNotExist(err) {
		t.Errorf("an empty gather created %s", failureDir)
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
