package remote

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// genPEMKey returns a fresh unencrypted ed25519 SSH private key in PEM form.
func genPEMKey(t *testing.T) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	blk, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return pem.EncodeToMemory(blk)
}

func TestAuthMethods(t *testing.T) {
	key := genPEMKey(t)
	cases := []struct {
		name    string
		creds   Credentials
		wantN   int
		wantErr bool
	}{
		{"password only", Credentials{Password: "pw"}, 1, false},
		{"key only", Credentials{PrivateKey: key}, 1, false},
		{"key and password", Credentials{Password: "pw", PrivateKey: key}, 2, false},
		{"neither", Credentials{}, 0, true},
		{"bad key", Credentials{PrivateKey: []byte("-----BEGIN nonsense-----")}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := authMethods(tc.creds)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("authMethods: %v", err)
			}
			if len(m) != tc.wantN {
				t.Fatalf("methods = %d, want %d", len(m), tc.wantN)
			}
		})
	}
}

func TestAuthMethods_BadKeyDoesNotLeak(t *testing.T) {
	// A malformed key carrying a unique sentinel; parsing must fail and the
	// error must not echo the sentinel (i.e. no key material in errors). The
	// bytes are deliberately not a PEM header so the secret scanner is not
	// tripped by the test fixture itself.
	const sentinel = "UNIQUE-KEY-SENTINEL-9137"
	_, err := authMethods(Credentials{PrivateKey: []byte("not-a-valid-key " + sentinel)})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("error leaked key material: %v", err)
	}
}

func TestLoadPrivateKey(t *testing.T) {
	dir := t.TempDir()
	key := genPEMKey(t)

	good := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(good, key, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(good, 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := LoadPrivateKey(good)
	if err != nil || len(b) == 0 {
		t.Fatalf("LoadPrivateKey(good) = %v", err)
	}

	insecure := filepath.Join(dir, "id_insecure")
	if err := os.WriteFile(insecure, key, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(insecure, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPrivateKey(insecure); err == nil {
		t.Fatal("expected insecure-permissions error for 0644 key")
	}

	if _, err := LoadPrivateKey(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected error for missing key file")
	}
}
