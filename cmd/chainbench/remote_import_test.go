package main

import (
	"strings"
	"testing"
)

func TestParseRemoteImport(t *testing.T) {
	cases := []struct {
		in               string
		user, host, path string
		wantErr          bool
	}{
		{"10.0.0.1:/keys/node1.json", "", "10.0.0.1", "/keys/node1.json", false},
		{"ubuntu@host:/k", "ubuntu", "host", "/k", false},
		{"nopath", "", "", "", true},
		{"host:", "", "", "", true},
		{":/path", "", "", "", true},
	}
	for _, c := range cases {
		u, h, p, err := parseRemoteImport(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", c.in)
			}
			continue
		}
		if err != nil || u != c.user || h != c.host || p != c.path {
			t.Errorf("%q -> (%q,%q,%q,%v), want (%q,%q,%q)", c.in, u, h, p, err, c.user, c.host, c.path)
		}
	}
}

func TestKeysImport_RemoteExactlyOneSource(t *testing.T) {
	// remote-import counts as a source: combining with --private-key is an error.
	if _, err := run(t, "keys", "import", "--remote-import", "h:/k", "--private-key", "0x01"); err == nil {
		t.Fatal("expected exactly-one-source error")
	}
	// malformed remote-import spec.
	if _, err := run(t, "keys", "import", "--remote-import", "bogus"); err == nil {
		t.Fatal("expected malformed remote-import error")
	}
	// well-formed but no SSH creds in env -> credential error (no dial).
	t.Setenv("CHAINBENCH_REMOTE_USER", "")
	t.Setenv("CHAINBENCH_REMOTE_PASS", "")
	t.Setenv("CHAINBENCH_REMOTE_KEY_FILE", "")
	_, err := run(t, "keys", "import", "--remote-import", "10.0.0.1:/k")
	if err == nil || !strings.Contains(err.Error()+" ", "SSH") {
		t.Fatalf("expected SSH credential error, got %v", err)
	}
}
