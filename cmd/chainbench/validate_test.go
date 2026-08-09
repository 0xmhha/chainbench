package main

import (
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
