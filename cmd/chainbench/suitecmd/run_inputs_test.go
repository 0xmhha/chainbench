package suitecmd_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// repoKeys is the repository's key preset, named absolutely so a run in this
// package's directory reaches the step under test instead of a missing key set.
func repoKeys(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "presets", "keys"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// TestRun_BinaryForReachesTheEngine: --binary-for binds a declared binary name
// to an executable the caller chose, and the engine judges it. A name the case
// does not declare is refused by the engine, which proves the flag got there.
func TestRun_BinaryForReachesTheEngine(t *testing.T) {
	stdout, _, err := runSplit(t, "run", "--workspace-dir", t.TempDir(), "--artifact-root", t.TempDir(), "--json",
		"--keys", repoKeys(t), "--binary-for", "undeclared=/bin/true", tcPath("07-basic-wbft-consensus.json"))
	if err == nil {
		t.Fatal("a run with an undeclared binary name succeeded")
	}
	var doc struct {
		Error string `json:"error"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &doc); jerr != nil {
		t.Fatalf("stdout is not a document: %v\n%s", jerr, stdout)
	}
	if !strings.Contains(doc.Error, `binary override "undeclared" is not declared by the test`) {
		t.Fatalf("error = %q, want the engine's refusal of the undeclared name", doc.Error)
	}
}

// TestRun_JSONCarriesTheSetupSteps: under --json the setup narration goes to
// stderr, so a caller reading only the document needs the steps in it to say
// what the run changed before it failed.
func TestRun_JSONCarriesTheSetupSteps(t *testing.T) {
	stdout, _, err := runSplit(t, "run", "--workspace-dir", t.TempDir(), "--artifact-root", t.TempDir(), "--json",
		"--keys", repoKeys(t), "--binary", "/nonexistent/chainbench-test-binary", tcPath("07-basic-wbft-consensus.json"))
	if err == nil {
		t.Fatal("a run on a missing binary succeeded")
	}
	var doc struct {
		Error      string   `json:"error"`
		SetupSteps []string `json:"setupSteps"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &doc); jerr != nil {
		t.Fatalf("stdout is not a document: %v\n%s", jerr, stdout)
	}
	if len(doc.SetupSteps) == 0 {
		t.Fatalf("the document lists no setup steps although the run got as far as %q", doc.Error)
	}
}

// TestRun_SuiteRunsSeveralDefinitionsAsOneRun: with --suite, several
// definitions are one run on one network with one session, so the document has
// the single-run shape; without it they are a sequence of runs.
func TestRun_SuiteRunsSeveralDefinitionsAsOneRun(t *testing.T) {
	specs := []string{tcPath("07-basic-wbft-consensus.json"), tcPath("07-basic-wbft-consensus.json")}
	for _, tc := range []struct {
		suite bool
		want  string
	}{{true, "error"}, {false, "runs"}} {
		args := []string{"run", "--workspace-dir", t.TempDir(), "--artifact-root", t.TempDir(), "--json",
			"--keys", repoKeys(t), "--binary", "/nonexistent/chainbench-test-binary"}
		if tc.suite {
			args = append(args, "--suite")
		}
		stdout, _, err := runSplit(t, append(args, specs...)...)
		if err == nil {
			t.Fatal("a run on a missing binary succeeded")
		}
		var doc map[string]any
		if jerr := json.Unmarshal([]byte(stdout), &doc); jerr != nil {
			t.Fatalf("suite=%v: stdout is not a document: %v\n%s", tc.suite, jerr, stdout)
		}
		if _, ok := doc[tc.want]; !ok {
			t.Fatalf("suite=%v: document %s has no %q", tc.suite, stdout, tc.want)
		}
	}
}

// TestRun_NoNodeControlIsAnAttachFlag: --no-node-control reaches the attach
// run, which then reports the workspace it was pointed at rather than the flag.
func TestRun_NoNodeControlIsAnAttachFlag(t *testing.T) {
	_, _, err := runSplit(t, "run", "--attach", "--workspace-dir", t.TempDir(), "--no-node-control", "--read-only-keys",
		"--artifact-root", t.TempDir(), tcPath("08-attached-chain-produces.json"))
	if err == nil {
		t.Fatal("attaching to a workspace with no network succeeded")
	}
	if strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("the attach flags were not accepted: %v", err)
	}
}
