package suitecmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/exitcode"
	"github.com/0xmhha/chainbench/cmd/chainbench/suitecmd"
)

// runSplit is run() with the two streams kept apart. Under --json the promise is
// about stdout specifically — that the whole of it parses — so a helper that
// merges the narration into it cannot check the thing being promised.
func runSplit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(suitecmd.NewRun())
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	err = root.ExecuteContext(context.Background())
	return out.String(), errb.String(), err
}

func codeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var ec *exitcode.Error
	if errors.As(err, &ec) {
		return ec.Code
	}
	// What the CLI does with a plain error.
	return 1
}

func tcPath(name string) string {
	return filepath.Join("..", "..", "..", "tests", "tc", "basic", name)
}

// TestRun_SetupFailureAnswersTheSameWayForOneSpecAndSeveral is MON-015's
// remaining half.
//
// A chain that will not compose is the same news whether one definition was
// named or three, but the two paths disagreed: several definitions produced a
// document and exit 2, while one produced an empty stdout and exit 1. A CI job
// or an agent reading stdout could not tell a compose failure from a crash, and
// could not tell either from a test that ran and failed.
//
// This drives the command itself rather than the mapping function, because the
// disagreement was in which branch ran, not in how a total was scored.
func TestRun_SetupFailureAnswersTheSameWayForOneSpecAndSeveral(t *testing.T) {
	// A binary that cannot exist: the chain gets as far as init and stops.
	const noSuchBinary = "/nonexistent/chainbench-test-binary"

	cases := []struct {
		name  string
		specs []string
	}{
		{"one definition", []string{tcPath("01-basic-consensus.json")}},
		{"several definitions", []string{tcPath("01-basic-consensus.json"), tcPath("02-basic-peers.json")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"run", "--workspace-dir", t.TempDir(), "--json", "--binary", noSuchBinary}, tc.specs...)
			stdout, _, err := runSplit(t, args...)

			// A definition that could not run is code 2, not 1.
			if got := codeOf(t, err); got != 2 {
				t.Fatalf("exit code = %d, want 2 for a setup failure: %v", got, err)
			}
			// And the whole of stdout is still a document.
			if strings.TrimSpace(stdout) == "" {
				t.Fatal("stdout is empty — under --json a consumer gets nothing to read")
			}
			var doc map[string]any
			if jerr := json.Unmarshal([]byte(stdout), &doc); jerr != nil {
				t.Fatalf("stdout does not parse as one JSON document: %v\n%s", jerr, stdout)
			}
			// The cause has to be identifiable from the document, not only from
			// the words on stderr. Which step gave out first depends on the
			// environment the test runs in — a missing key set is reached before
			// a missing binary — so what is asserted is that the document says,
			// not which sentence it says.
			if !carriesACause(doc) {
				t.Fatalf("the document does not say what failed: %s", stdout)
			}
		})
	}
}

// carriesACause looks for a stated reason in either shape: the single run's
// top-level error, or one per definition in the sequence's runs.
func carriesACause(doc map[string]any) bool {
	if s, ok := doc["error"].(string); ok && strings.TrimSpace(s) != "" {
		return true
	}
	runs, ok := doc["runs"].([]any)
	if !ok || len(runs) == 0 {
		return false
	}
	for _, r := range runs {
		m, ok := r.(map[string]any)
		if !ok {
			return false
		}
		s, ok := m["error"].(string)
		if !ok || strings.TrimSpace(s) == "" {
			return false // a definition that failed and said nothing
		}
	}
	return true
}

// TestRun_SetupFailureIsCodeTwoWithoutJSONToo: the code is about what happened,
// not about how it is printed. Only the document is conditional on --json.
func TestRun_SetupFailureIsCodeTwoWithoutJSONToo(t *testing.T) {
	_, _, err := runSplit(t, "run", "--workspace-dir", t.TempDir(),
		"--binary", "/nonexistent/chainbench-test-binary", tcPath("01-basic-consensus.json"))
	if got := codeOf(t, err); got != 2 {
		t.Fatalf("exit code = %d, want 2 without --json as well: %v", got, err)
	}
}

// TestRun_SetupFailureDocumentCarriesNoKeyMaterial: the failure document is a
// new place for an error string to be published, and setup errors are carried
// into it verbatim. A node key named inline is refused during compose (MON-001),
// and that refusal must not become the thing this prints.
func TestRun_SetupFailureDocumentCarriesNoKeyMaterial(t *testing.T) {
	const inlineHex = "0x4444444444444444444444444444444444444444444444444444444444444444"
	bare := strings.TrimPrefix(inlineHex, "0x")
	dir := t.TempDir()
	spec := filepath.Join(dir, "case.json")
	if err := writeCaseWithInlineKey(spec, inlineHex); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runSplit(t, "run", "--workspace-dir", dir, "--json",
		"--binary", "/nonexistent/chainbench-test-binary", spec)
	if err == nil {
		t.Fatal("an inline node key must fail the run")
	}
	for _, stream := range map[string]string{"stdout": stdout, "stderr": stderr} {
		for _, secret := range []string{inlineHex, bare} {
			if strings.Contains(stream, secret) {
				t.Fatalf("the failure report carries the private key:\n%s", stream)
			}
		}
	}
}

// writeCaseWithInlineKey writes the smallest v2 case whose env declares a node
// table with one node's key written inline.
func writeCaseWithInlineKey(path, key string) error {
	doc := map[string]any{
		"schemaVersion": "2",
		"kind":          "case",
		"id":            "inline-key-case",
		"env": map[string]any{
			"schemaVersion": "2",
			"kind":          "env",
			"id":            "inline-key-env",
			"chain":         "stablenet",
			"binaries":      map[string]any{"default": "gstable"},
			"topology": map[string]any{
				"chain": "stablenet",
				"nodes": []any{
					map[string]any{"index": 1, "role": "bp", "key": key},
					map[string]any{"index": 2, "role": "bp"},
					map[string]any{"index": 3, "role": "bp"},
					map[string]any{"index": 4, "role": "bp"},
				},
			},
		},
		"tests": []any{
			map[string]any{
				"id": "t1",
				"do": []any{map[string]any{"rpc": "eth_blockNumber", "as": "h"}},
			},
		},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
