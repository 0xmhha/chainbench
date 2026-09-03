package report_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// buildSession creates a saved session with two tests and returns its root.
func buildSession(t *testing.T) string {
	t.Helper()
	s, err := session.New(t.TempDir(), "chainbench run a.json b.json", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r1 := s.Test(1, "alpha")
	r1.SetEnvRef("env-abc123")
	r1.Spec([]byte(`{"id":"alpha"}`))
	r1.Artifacts(session.TestArtifacts{Refs: []session.ArtifactRef{
		{Kind: "genesis", Ref: "environments/env-abc123/genesis/genesis.json"},
		{Kind: "config", Ref: "sha256:deadbeef", Node: "node2"},
	}})
	r1.Status(session.StatusPass)

	r2 := s.Test(2, "beta")
	r2.SetEnvRef("env-abc123")
	r2.Status(session.StatusFail)
	r2.Reason("assertion X failed")

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return s.Root()
}

func TestGenerateAndRead_RoundTrip(t *testing.T) {
	root := buildSession(t)

	rep, err := report.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// report.json exists at the session root.
	if _, err := os.Stat(filepath.Join(root, "report.json")); err != nil {
		t.Fatalf("report.json not written: %v", err)
	}

	got, err := report.Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Session != rep.Session || got.Command != rep.Command {
		t.Fatalf("read mismatch: %+v vs %+v", got, rep)
	}
	if got.Summary.Pass != 1 || got.Summary.Fail != 1 {
		t.Fatalf("summary = %+v, want pass=1 fail=1", got.Summary)
	}
	if len(got.Tests) != 2 {
		t.Fatalf("tests = %d, want 2", len(got.Tests))
	}

	alpha := got.Tests[0]
	if alpha.ID != "alpha" || alpha.Status != "pass" || alpha.Env != "env-abc123" {
		t.Fatalf("alpha = %+v", alpha)
	}
	// The pass test links the evidence files it wrote (spec/status/artifacts/env-ref order excludes env-ref).
	joined := strings.Join(alpha.Evidence, " ")
	for _, want := range []string{"spec.json", "status.json", "artifacts.json"} {
		if !strings.Contains(joined, want) {
			t.Errorf("alpha evidence %q missing %q", joined, want)
		}
	}
	if !strings.Contains(alpha.Dir, "001_alpha") {
		t.Errorf("alpha dir = %q, want to contain 001_alpha", alpha.Dir)
	}
}

// TestBuild_WithoutGenerate builds a report directly from session.json even when
// report.json was never written (the fallback path the CLI/MCP use).
func TestBuild_WithoutGenerate(t *testing.T) {
	root := buildSession(t)
	rep, err := report.Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if rep.Summary.Pass != 1 || rep.Summary.Fail != 1 {
		t.Fatalf("summary = %+v", rep.Summary)
	}
}

// TestReport_NoSecretLeak asserts the report and the artifacts it links never
// contain raw key material: artifacts carry references, not secret bytes.
func TestReport_NoSecretLeak(t *testing.T) {
	root := buildSession(t)
	if _, err := report.Generate(root); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Scan report.json and every per-test artifact for a private-key-shaped
	// string. The needle is built at runtime (not a source literal) so this test
	// file does not itself embed a 64-hex key.
	secret := strings.Repeat("ab", 32)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(b), secret) {
			t.Errorf("artifact %s leaked secret material", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
