package suitecmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	// A binary that cannot exist: valid inputs reach binary provisioning.
	const noSuchBinary = "/nonexistent/chainbench-test-binary"
	definitions := setupFailureDefinitions(t, 2)

	cases := []struct {
		name  string
		specs []string
	}{
		{"one definition", definitions[:1]},
		{"several definitions", definitions},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The artifact root is named, so the test's own failure records do
			// not land in the operator's ~/.chainbench.
			args := append([]string{"run", "--workspace-dir", t.TempDir(), "--artifact-root", t.TempDir(), "--json", "--binary", noSuchBinary, "--keys", repoKeys(t), "--read-only-keys"}, tc.specs...)
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
			// Each definition must reach the intended binary setup failure.
			if !carriesACause(doc) {
				t.Fatalf("the document does not say what failed: %s", stdout)
			}
			requireBinarySetupFailure(t, doc, noSuchBinary)
			if len(tc.specs) > 1 {
				if runs, _ := doc["runs"].([]any); len(runs) != len(tc.specs) {
					t.Fatalf("only %d of %d definitions reached setup", len(runs), len(tc.specs))
				}
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

func requireBinarySetupFailure(t *testing.T, doc map[string]any, binary string) {
	t.Helper()
	if runs, ok := doc["runs"].([]any); ok {
		if len(runs) == 0 {
			t.Fatal("no definitions reached setup")
		}
		for _, run := range runs {
			cause, _ := run.(map[string]any)["error"].(string)
			if !strings.Contains(cause, binary) {
				t.Fatalf("definition did not reach binary setup: %s", cause)
			}
		}
		return
	}
	cause, _ := doc["error"].(string)
	if !strings.Contains(cause, binary) {
		t.Fatalf("did not reach binary setup: %s", cause)
	}
	steps, _ := doc["setupSteps"].([]any)
	if len(steps) == 0 || doc["failedAt"] == nil {
		t.Fatalf("missing setup execution or failure state: %#v", doc)
	}
}

// TestRun_SetupFailureIsCodeTwoWithoutJSONToo: the code is about what happened,
// not about how it is printed. Only the document is conditional on --json.
func TestRun_SetupFailureIsCodeTwoWithoutJSONToo(t *testing.T) {
	_, _, err := runSplit(t, "run", "--workspace-dir", t.TempDir(), "--artifact-root", t.TempDir(),
		"--keys", repoKeys(t), "--read-only-keys", "--binary", "/nonexistent/chainbench-test-binary", setupFailureDefinitions(t, 1)[0])
	if got := codeOf(t, err); got != 2 {
		t.Fatalf("exit code = %d, want 2 without --json as well: %v", got, err)
	}
	if !strings.Contains(err.Error(), "/nonexistent/chainbench-test-binary") {
		t.Fatalf("did not reach binary setup: %v", err)
	}
}

// setupFailureDefinitions validates each distinct definition before setup. A missing or
// malformed fixture must fail here, never masquerade as the requested failure.
func setupFailureDefinitions(t *testing.T, count int) []string {
	t.Helper()
	source, err := os.ReadFile(tcPath("07-basic-wbft-consensus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var definition map[string]any
	if err := json.Unmarshal(source, &definition); err != nil {
		t.Fatal(err)
	}
	presetPath, err := filepath.Abs(filepath.Join("..", "..", "..", "presets", "chain", "stablenet-bp4-en1.json"))
	if err != nil {
		t.Fatal(err)
	}
	presetData, err := os.ReadFile(presetPath)
	if err != nil {
		t.Fatal(err)
	}
	var preset map[string]any
	if err := json.Unmarshal(presetData, &preset); err != nil {
		t.Fatal(err)
	}
	preset["keys"] = map[string]any{"nodekeys": map[string]any{"ref": repoKeys(t), "source": "keyPreset"}}
	definition["chainPreset"] = preset
	directory := t.TempDir()
	var paths []string
	for n := 0; n < count; n++ {
		definition["id"] = fmt.Sprintf("setup-failure-%d", n)
		encoded, err := json.Marshal(definition)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, fmt.Sprintf("case-%d.json", n))
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		if output, err := run(t, "validate", path); err != nil {
			t.Fatalf("invalid setup fixture: %v\n%s", err, output)
		}
		paths = append(paths, path)
	}
	return paths
}

// TestRun_SetupFailureDocumentCarriesNoKeyMaterial: the failure document is a
// place an error string gets published, and setup errors are carried into it
// verbatim. A node key named inline is refused during compose (MON-001), and
// that refusal must not become the thing this prints.
//
// The first version of this test proved none of that. Its fixture wrote a
// top-level "tests" field where a v2 case has "steps", the parser rejects
// unknown fields, and the run therefore died at "unknown field" long before
// anything looked at a key. The assertions accepted any error at all and only
// searched the streams, and a syntax message has no key in it — so the test
// passed even with the refusal deliberately changed to quote the key.
//
// Hence the shape below: prove the fixture is a valid spec BEFORE running it, so
// a parse failure cannot masquerade as the thing under test, then require the
// failure to be the key refusal by name.
func TestRun_SetupFailureDocumentCarriesNoKeyMaterial(t *testing.T) {
	// Not a real key: 32 bytes of 0x44, recognisable in any stream it leaks to.
	const inlineHex = "0x4444444444444444444444444444444444444444444444444444444444444444"
	bare := strings.TrimPrefix(inlineHex, "0x")
	dir := t.TempDir()
	spec := filepath.Join(dir, "case.json")
	if err := writeCaseWithInlineKey(spec, inlineHex); err != nil {
		t.Fatal(err)
	}

	// The fixture has to be a spec this project accepts, and validate is the
	// verb that says so. Asserting it here is the guard the old test lacked: a
	// fixture that stops being parseable fails loudly instead of quietly
	// exercising the parser instead of the key check.
	if out, err := run(t, "validate", spec); err != nil {
		t.Fatalf("the fixture must parse and pre-check cleanly, or the run below tests the parser: %v\n%s", err, out)
	}

	// No --binary: the inline key is then the only thing wrong with this run, so
	// a refusal for any other reason is a real signal rather than noise.
	stdout, stderr, err := runSplit(t, "run", "--workspace-dir", dir, "--artifact-root", t.TempDir(), "--json", spec)
	if err == nil {
		t.Fatal("an inline node key must fail the run")
	}
	if got := codeOf(t, err); got != 2 {
		t.Fatalf("exit code = %d, want 2 for a setup failure: %v", got, err)
	}

	var doc map[string]any
	if jerr := json.Unmarshal([]byte(stdout), &doc); jerr != nil {
		t.Fatalf("stdout does not parse as one JSON document: %v\n%s", jerr, stdout)
	}
	cause, _ := doc["error"].(string)
	if cause == "" {
		t.Fatalf("the document states no cause: %s", stdout)
	}
	// It has to be THE refusal. A syntax error, a missing binary and a missing
	// key set all fail the run too, and none of them would be evidence about
	// how a key is reported. Matching two words rather than the whole sentence
	// keeps this from breaking every time the message is reworded, and the
	// validate call above already rules out the parse-failure case.
	for _, want := range []string{"inline", "node1"} {
		if !strings.Contains(cause, want) {
			t.Fatalf("the run failed for something other than the inline key (no %q in the cause): %s", want, cause)
		}
	}

	// Every surface the failure reaches, not just the two streams: the returned
	// error is what the process exits with, and the document's error field is
	// what a machine reads.
	surfaces := map[string]string{
		"stdout":         stdout,
		"stderr":         stderr,
		"returned error": err.Error(),
		"document error": cause,
	}
	for name, text := range surfaces {
		for _, secret := range []string{inlineHex, bare, strings.ToUpper(bare)} {
			if strings.Contains(text, secret) {
				t.Fatalf("the %s carries the private key:\n%s", name, text)
			}
		}
	}
}

// writeCaseWithInlineKey writes the smallest valid v2 case whose env declares a
// node table with one node's key written inline.
//
// "steps" is the field name, and what goes in it has to be a registered action:
// the point of the fixture is to reach the composition, so anything the grammar
// or the interpreter rejects defeats it before it starts.
func writeCaseWithInlineKey(path, key string) error {
	doc := map[string]any{
		"schemaVersion": "2",
		"kind":          "case",
		"id":            "inline-key-case",
		"chainPreset": map[string]any{
			"schemaVersion": "2",
			"kind":          "chain-preset",
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
		// One assertion, of the kind every shipped case opens with. The network
		// never comes up here, so nothing runs it — it exists so the document is
		// a case rather than a fragment.
		"steps": []any{
			map[string]any{
				"expect":  "blockNumber",
				"on":      "node1",
				"compare": "GreaterOrEqual",
				"is":      "0",
			},
		},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
