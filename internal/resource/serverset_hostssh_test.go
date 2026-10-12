package resource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServerSet_HostSSHBlockReplacesTheSetBlock: a host can carry its own ssh
// block, so two servers of one set can log in as different users with
// different secrets. A host's block replaces the set's block whole, so a
// password from one never meets a password file from the other.
func TestServerSet_HostSSHBlockReplacesTheSetBlock(t *testing.T) {
	dir := t.TempDir()
	for name, value := range map[string]string{"alice.pass": "alice-secret", "bob.pass": "bob-secret"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "server-set.yaml")
	if err := os.WriteFile(path, []byte(`version: 2
pool:
  hosts:
    - {name: one, addr: 10.0.0.1}
    - name: two
      addr: 10.0.0.2
      ssh: {user: bob, port: 2202, password_file: bob.pass, insecure_host_key: true}
ssh: {user: alice, password_file: alice.pass, insecure_host_key: true}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := LoadSet(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		server, user, pass string
		port               int
	}{
		{"one", "alice", "alice-secret", 22},
		{"two", "bob", "bob-secret", 2202},
	} {
		s, err := set.ByName(want.server)
		if err != nil {
			t.Fatal(err)
		}
		c, err := s.Credentials()
		if err != nil {
			t.Fatalf("%s: %v", want.server, err)
		}
		if c.User != want.user || c.Password != want.pass || c.Port != want.port {
			t.Fatalf("%s logs in as %s:%s on port %d, want %s:%s on %d", want.server, c.User, c.Password, c.Port, want.user, want.pass, want.port)
		}
	}
}

// TestServerSet_HostSSHBlockIsValidatedLikeTheSetBlock: a host block breaking
// the ssh rules is refused when the file is read, naming the host.
func TestServerSet_HostSSHBlockIsValidatedLikeTheSetBlock(t *testing.T) {
	_, err := ParseSet([]byte(`version: 2
pool:
  hosts:
    - name: two
      addr: 10.0.0.2
      ssh: {user: bob, password: a, password_file: b}
`))
	if err == nil || !strings.Contains(err.Error(), "two") || !strings.Contains(err.Error(), "password_file") {
		t.Fatalf("a host block with both password forms gave %v, want a refusal naming the host", err)
	}
}

// TestServerSet_HostSSHBlockInheritsNothing: a set-level password must not
// reach a host whose own block uses a password file or a key, or the host would
// carry two ways to log in and the credentials would be refused.
func TestServerSet_HostSSHBlockInheritsNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bob.pass"), []byte("bob-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "server-set.yaml")
	if err := os.WriteFile(path, []byte(`version: 2
pool:
  hosts:
    - name: two
      addr: 10.0.0.2
      ssh: {user: bob, password_file: bob.pass, insecure_host_key: true}
ssh: {user: alice, password: alice-secret, key_file: alice.key, insecure_host_key: true}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := LoadSet(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := set.ByName("two")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Credentials()
	if err != nil {
		t.Fatalf("host block with its own password file: %v", err)
	}
	if c.User != "bob" || c.Password != "bob-secret" || len(c.PrivateKey) != 0 {
		t.Fatalf("host two logs in as %s:%s with a %d-byte key, want bob's password alone", c.User, c.Password, len(c.PrivateKey))
	}
}
