package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKeysNew_TextAndJSON(t *testing.T) {
	out, err := run(t, "keys", "new")
	if err != nil {
		t.Fatalf("keys new: %v\n%s", err, out)
	}
	if !strings.Contains(out, "privateKey: 0x") || !strings.Contains(out, "address:    0x") {
		t.Fatalf("keys new output: %s", out)
	}

	out, err = run(t, "keys", "new", "--json")
	if err != nil {
		t.Fatalf("keys new --json: %v\n%s", err, out)
	}
	var kp struct {
		PrivateKey string `json:"privateKey"`
		Address    string `json:"address"`
	}
	if err := json.Unmarshal([]byte(out), &kp); err != nil {
		t.Fatalf("keys new --json not valid JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(kp.PrivateKey, "0x") || len(kp.PrivateKey) != 66 {
		t.Fatalf("private key = %q (want 0x + 64 hex)", kp.PrivateKey)
	}
	if !strings.HasPrefix(kp.Address, "0x") || len(kp.Address) != 42 {
		t.Fatalf("address = %q (want 0x + 40 hex)", kp.Address)
	}
}

func TestValidatorSet_RequiresFlags(t *testing.T) {
	// Missing required --nodes/--binary/--out.
	if _, err := run(t, "validator", "set"); err == nil {
		t.Fatal("expected error without required flags")
	}
	// Bogus bootnode path -> generation error.
	if _, err := run(t, "validator", "set", "--nodes", "1", "--binary", "/nonexistent/bin", "--out", t.TempDir(), "--bootnode", "/nonexistent/bootnode"); err == nil {
		t.Fatal("expected error for missing bootnode/binary")
	}
}
