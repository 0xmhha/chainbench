package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCmd_ValidAndInvalid(t *testing.T) {
	valid := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "good",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 1337}},
	})
	// Missing assertions -> invalid.
	invalid := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "bad",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
	})

	// All-valid run exits 0.
	out, err := run(t, "validate", valid)
	if err != nil {
		t.Fatalf("validate valid: %v\n%s", err, out)
	}
	if !strings.Contains(out, "good") || !strings.Contains(out, "OK") {
		t.Fatalf("expected OK row for valid spec:\n%s", out)
	}

	// A run with an invalid spec exits non-zero.
	out, err = run(t, "validate", valid, invalid)
	if exitCode(err) != 1 {
		t.Fatalf("validate invalid: exit = %d, want 1\n%s", exitCode(err), out)
	}
	if !strings.Contains(out, "INVALID") {
		t.Fatalf("expected INVALID row:\n%s", out)
	}
}

func TestValidateCmd_MissingFile(t *testing.T) {
	if _, err := run(t, "validate", "/no/such/spec.json"); exitCode(err) != 1 {
		t.Fatalf("missing file should exit 1, got %d", exitCode(err))
	}
}

func TestValidateCmd_ChainApplicability(t *testing.T) {
	base := map[string]any{
		"schemaVersion": "1",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 1337}},
	}
	spec := func(extra map[string]any) string {
		m := map[string]any{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return writeSpec(t, m)
	}

	// requires a capability stablenet has -> OK.
	okPath := spec(map[string]any{"id": "needs-rpc", "requires": []string{"rpc"}})
	if out, err := run(t, "validate", "--chain", "stablenet", okPath); err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("expected OK for satisfied caps:\n%s (err=%v)", out, err)
	}

	// requires a capability stablenet lacks -> SKIP (needs caps), still exit 0.
	needsPath := spec(map[string]any{"id": "needs-archive", "requires": []string{"archive"}})
	out, err := run(t, "validate", "--chain", "stablenet", needsPath)
	if err != nil {
		t.Fatalf("unsatisfied caps must not fail validate: %v", err)
	}
	if !strings.Contains(out, "needs caps: archive") {
		t.Fatalf("expected needs-caps note:\n%s", out)
	}

	// applicableChains excludes stablenet -> SKIP (not applicable).
	otherPath := spec(map[string]any{"id": "wbft-only", "applicableChains": "wbft"})
	if out, err := run(t, "validate", "--chain", "stablenet", otherPath); err != nil || !strings.Contains(out, "not applicable") {
		t.Fatalf("expected not-applicable note:\n%s (err=%v)", out, err)
	}
}

func TestValidateCmd_UnknownChain(t *testing.T) {
	p := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "x",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "chainId", "expected": 1}},
	})
	if _, err := run(t, "validate", "--chain", "nope", p); err == nil {
		t.Fatal("unknown --chain must error")
	}
}

// TestValidateCmd_ExampleSpecs guards the shipped example specs: they must all
// parse and apply to stablenet, so the DSL docs never drift from the parser.
func TestValidateCmd_ExampleSpecs(t *testing.T) {
	paths, err := filepath.Glob("../../examples/specs/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no example specs found")
	}
	args := append([]string{"validate", "--chain", "stablenet"}, paths...)
	out, err := run(t, args...)
	if err != nil {
		t.Fatalf("example specs failed validation: %v\n%s", err, out)
	}
	if strings.Contains(out, "INVALID") || strings.Contains(out, "ERROR") {
		t.Fatalf("example specs must be valid:\n%s", out)
	}
}

func TestValidateCmd_UnresolvedNames(t *testing.T) {
	// Parses fine, but references an unknown assertion and action.
	bad := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "typo",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"steps":         []map[string]any{{"teleport": map[string]any{}}},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 1}, {"assert": "Nonexistent"}},
	})
	out, err := run(t, "validate", bad)
	if exitCode(err) != 1 {
		t.Fatalf("unresolved spec should exit 1, got %d\n%s", exitCode(err), out)
	}
	if !strings.Contains(out, "UNRESOLVED") || !strings.Contains(out, "assert:Nonexistent") || !strings.Contains(out, "action:teleport") {
		t.Fatalf("expected unresolved names in output:\n%s", out)
	}
}

func TestValidateCmd_JSONOutput(t *testing.T) {
	good := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "good",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "chainId", "expected": 1337}},
	})
	bad := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "bad",
		"chain": map[string]any{"name": "stablenet", "binary": "go-stablenet"},
	}) // missing assertions

	out, err := run(t, "validate", "--json", good, bad)
	if exitCode(err) != 1 {
		t.Fatalf("expected exit 1 with an invalid spec, got %d\n%s", exitCode(err), out)
	}
	var results []struct {
		Spec   string `json:"spec"`
		ID     string `json:"id"`
		OK     bool   `json:"ok"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if !results[0].OK || results[0].ID != "good" {
		t.Fatalf("first result should be OK good: %+v", results[0])
	}
	if results[1].OK || !strings.Contains(results[1].Result, "INVALID") {
		t.Fatalf("second result should be invalid: %+v", results[1])
	}
}

// TestValidateCmd_PortedSpecs guards the specs ported from the legacy Go-func
// suites: every one must parse and resolve. A ported case that only fails once
// a network is up would defeat the point of porting it.
func TestValidateCmd_PortedSpecs(t *testing.T) {
	paths, err := filepath.Glob("../../tests/specs/*/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no ported specs found under tests/specs")
	}
	out, err := run(t, append([]string{"validate"}, paths...)...)
	if err != nil {
		t.Fatalf("ported specs failed validation: %v\n%s", err, out)
	}
	if strings.Contains(out, "INVALID") || strings.Contains(out, "UNRESOLVED") {
		t.Fatalf("ported specs must parse and resolve:\n%s", out)
	}
}

// TestPortedSpecs_IDsAreUnique keeps a ported id from colliding with another,
// since the session records a test by id and a duplicate would overwrite it.
func TestPortedSpecs_IDsAreUnique(t *testing.T) {
	paths, err := filepath.Glob("../../tests/specs/*/*.json")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(b, &spec); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if spec.ID == "" {
			t.Errorf("%s has no id", p)
			continue
		}
		if prev, dup := seen[spec.ID]; dup {
			t.Errorf("duplicate spec id %q in %s and %s", spec.ID, prev, p)
		}
		seen[spec.ID] = p
	}
}

// TestValidateCmd_ChainCases: the four chain-setup declarations and their
// cases validate offline — env references resolve from the shared env/
// directory, and an env file validates as a declaration.
func TestValidateCmd_ChainCases(t *testing.T) {
	paths, err := filepath.Glob("../../tests/cases/*/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 8 {
		t.Fatalf("expected the four envs and four cases under tests/cases, found %d files", len(paths))
	}
	var out bytes.Buffer
	if err := validateSpecs(&out, paths, "", false); err != nil {
		t.Fatalf("validate tests/cases: %v\n%s", err, out.String())
	}
	for _, want := range []string{"env declaration for chain wemix", "wemix-wbft-handoff", "stablenet-chain-up"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should mention %q:\n%s", want, out.String())
		}
	}
}
