package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDepsWithLoginsReadsTheGivenSet: logins named apart from the recorded
// server set are what a dial uses, so a network can record a set with no
// secret in it while one run supplies its own logins.
func TestDepsWithLoginsReadsTheGivenSet(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pass"), []byte("run-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	logins := filepath.Join(dir, "logins.yaml")
	if err := os.WriteFile(logins, []byte("version: 2\npool:\n  hosts:\n    - {name: far, addr: 10.0.0.9, ssh: {user: ops, password_file: pass}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d := (Deps{}).WithLogins(""); d.ServerLookup != nil {
		t.Fatal("no logins file still replaced the recorded set's lookup")
	}
	c, err := (Deps{}).WithLogins(logins).ServerLookup("far")
	if err != nil {
		t.Fatal(err)
	}
	if c.User != "ops" || c.Password != "run-secret" {
		t.Fatalf("logins = %s:%s", c.User, c.Password)
	}
}
