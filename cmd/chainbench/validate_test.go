package main

import (
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
