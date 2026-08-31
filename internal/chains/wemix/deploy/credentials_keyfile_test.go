package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

func TestCredentials_For_KeyFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id")
	if err := os.WriteFile(keyPath, []byte("PEMKEYBYTES"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(keyPath, 0o600); err != nil {
		t.Fatal(err)
	}
	c := &Cluster{SSHPort: 22, Servers: []Server{{Index: 1, Host: "10.0.0.1", Role: RoleEndpoint}}}
	noEnv := func(string) string { return "" }

	// key_file only (no password) is now valid.
	cr := &Credentials{User: "ubuntu", KeyFile: keyPath}
	rc, err := cr.For(c, c.Servers[0], noEnv)
	if err != nil {
		t.Fatalf("For with key_file: %v", err)
	}
	if string(rc.PrivateKey) != "PEMKEYBYTES" {
		t.Fatalf("private key not loaded: %q", rc.PrivateKey)
	}
	if rc.Password != "" {
		t.Fatalf("password should be empty, got %q", rc.Password)
	}

	// key_file via env, with passphrase.
	env := map[string]string{
		remote.EnvKeyFile:       keyPath,
		remote.EnvKeyPassphrase: "secret",
	}
	crEnv := &Credentials{User: "ubuntu"}
	rcEnv, err := crEnv.For(c, c.Servers[0], func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("For with env key_file: %v", err)
	}
	if len(rcEnv.PrivateKey) == 0 || rcEnv.Passphrase != "secret" {
		t.Fatalf("env key/passphrase not applied: key=%d passphrase=%q", len(rcEnv.PrivateKey), rcEnv.Passphrase)
	}

	// insecure key file perms are rejected.
	badPath := filepath.Join(dir, "id_bad")
	if err := os.WriteFile(badPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(badPath, 0o644)
	if _, err := (&Credentials{User: "u", KeyFile: badPath}).For(c, c.Servers[0], noEnv); err == nil {
		t.Fatal("expected insecure-permissions error")
	}
}
