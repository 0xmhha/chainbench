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
	// The keys layer shows the public key (raw keypair), not the address.
	if !strings.Contains(out, "privateKey: 0x") || !strings.Contains(out, "publicKey:  0x") {
		t.Fatalf("keys new output: %s", out)
	}
	if strings.Contains(out, "address:") {
		t.Fatalf("keys new must not print an address: %s", out)
	}

	out, err = run(t, "keys", "new", "--json")
	if err != nil {
		t.Fatalf("keys new --json: %v\n%s", err, out)
	}
	var kp struct {
		PrivateKey string `json:"privateKey"`
		PublicKey  string `json:"publicKey"`
		Address    string `json:"address"`
	}
	if err := json.Unmarshal([]byte(out), &kp); err != nil {
		t.Fatalf("keys new --json not valid JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(kp.PrivateKey, "0x") || len(kp.PrivateKey) != 66 {
		t.Fatalf("private key = %q (want 0x + 64 hex)", kp.PrivateKey)
	}
	if !strings.HasPrefix(kp.PublicKey, "0x") || len(kp.PublicKey) != 130 {
		t.Fatalf("public key = %q (want 0x + 128 hex)", kp.PublicKey)
	}
	if kp.Address != "" {
		t.Fatalf("keys new must not emit an address, got %q", kp.Address)
	}
}

func TestValidatorSet_RequiresFlags(t *testing.T) {
	// Missing required --nodes/--out.
	if _, err := run(t, "validator", "set"); err == nil {
		t.Fatal("expected error without required flags")
	}
	// Bogus bootnode path -> generation error (no node binary needed anymore).
	if _, err := run(t, "validator", "set", "--nodes", "1", "--out", t.TempDir(), "--bootnode", "/nonexistent/bootnode"); err == nil {
		t.Fatal("expected error for missing bootnode")
	}
}
