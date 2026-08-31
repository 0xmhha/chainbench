package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// inSetDir runs the test in a directory holding a server set that declares
// the host-key policy, which is where the policy now comes from: no
// environment variable configures it, and a machine with no ~/.ssh (a CI
// runner) must still be able to build a process.
func inSetDir(t *testing.T, ssh string) {
	t.Helper()
	dir := t.TempDir()
	body := "version: 2\npool:\n  hosts: [10.0.0.5]\nssh: {" + ssh + "}\ndataRoot: /data/cb\n"
	if err := os.WriteFile(filepath.Join(dir, "server-set.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
}

func TestRemoteDriver_EmptyHostIsLocal(t *testing.T) {
	d, err := remoteDriver("", "", 0)
	if err != nil || d != nil {
		t.Errorf("empty host should yield (nil, nil), got (%v, %v)", d, err)
	}
}

func TestRemoteDriver_PasswordRequiredFromEnv(t *testing.T) {
	t.Setenv(remote.EnvPass, "")
	if _, err := remoteDriver("10.0.0.5", "cb", 22); err == nil {
		t.Error("missing CHAINBENCH_REMOTE_PASS should error (password must not be a flag)")
	}
}

func TestRemoteDriver_BuildsWithThePolicyFromTheSet(t *testing.T) {
	inSetDir(t, "user: cb, password: pw, insecure_host_key: true")
	t.Setenv(remote.EnvPass, "secret")
	d, err := remoteDriver("10.0.0.5", "cb", 2222)
	if err != nil || d == nil {
		t.Fatalf("remoteDriver should build with the set's policy: (%v, %v)", d, err)
	}
}

// TestRemoteDriver_VerifiesAgainstTheSetsKnownHosts: with a known_hosts file
// named in the set, an unreadable one is refused — proof the file, not the
// environment, decides.
func TestRemoteDriver_VerifiesAgainstTheSetsKnownHosts(t *testing.T) {
	inSetDir(t, "user: cb, password: pw, known_hosts_file: no-such-known-hosts")
	t.Setenv(remote.EnvPass, "secret")
	if _, err := remoteDriver("10.0.0.5", "cb", 2222); err == nil {
		t.Error("a missing known_hosts file named by the set should error")
	}
}
