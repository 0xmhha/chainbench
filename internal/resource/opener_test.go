package resource_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

// fixture writes a server set and its localmap side by side, returning the
// set's path. The addresses are documentation examples, not live hosts —
// nothing here dials.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	set := filepath.Join(dir, "server-set.yaml")
	// The host-key policy is data in the set: an empty known_hosts of the
	// test's own keeps resolution off the machine's real ~/.ssh.
	kh := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(kh, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(set, []byte(
		"version: 2\n"+
			"pool:\n"+
			"  hosts: [{name: box1, addr: 192.0.2.11}]\n"+
			"ssh: {user: dev, password: pw, known_hosts_file: known_hosts}\n"+
			"dataRoot: /data/cb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "localmap.yaml"), []byte(
		"hosts:\n"+
			"  192.0.2.11:\n"+
			"    host: 127.0.0.1\n"+
			"    ports: { 22: 2201 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return set
}

// TestOpener_TranslatesAndReportsInOnePlace pins the wrapper's reason to
// exist: the server-set lookup, the docker translation, and the translation
// report all happen here, identically for every consumer.
func TestOpener_TranslatesAndReportsInOnePlace(t *testing.T) {
	set := fixture(t)
	var notes []string
	o := resource.Opener{
		ServerSet: set, Docker: true,
		Report: func(f string, a ...any) { notes = append(notes, fmt.Sprintf(f, a...)) },
	}
	acc, err := o.OpenPath("srv://box1/data/cb/ring")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if acc.DataRoot != "/data/cb/ring" {
		t.Errorf("data root = %q", acc.DataRoot)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "192.0.2.11:22") || !strings.Contains(notes[0], "127.0.0.1:2201") {
		t.Errorf("translation not reported from the wrapper: %v", notes)
	}
}

// TestOpener_DockerIsThePowerSwitch: with the flag, a missing localmap is an
// error; without it, a leftover localmap changes nothing and reports nothing.
func TestOpener_DockerIsThePowerSwitch(t *testing.T) {
	set := fixture(t)

	bare := filepath.Join(t.TempDir(), "server-set.yaml")
	raw, _ := os.ReadFile(set)
	if err := os.WriteFile(bare, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (resource.Opener{ServerSet: bare, Docker: true}).OpenPath("srv://box1/x"); err == nil {
		t.Fatal("docker mode accepted a missing localmap")
	}

	var notes []string
	o := resource.Opener{ServerSet: set, Report: func(f string, a ...any) { notes = append(notes, f) }}
	if _, err := o.OpenPath("srv://box1/x"); err != nil {
		t.Fatalf("open without docker: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("leftover localmap was applied without the flag: %v", notes)
	}
}

// TestOpener_CredentialsComeFromTheSet: an unknown name is refused by the
// lookup, and a local path resolves to local handles through the same call.
func TestOpener_CredentialsComeFromTheSet(t *testing.T) {
	set := fixture(t)
	if _, err := (resource.Opener{ServerSet: set}).OpenPath("srv://ghost/x"); err == nil {
		t.Fatal("unknown server name was accepted")
	}

	local := t.TempDir()
	acc, err := (resource.Opener{ServerSet: set}).OpenPath(local)
	if err != nil {
		t.Fatalf("local open: %v", err)
	}
	if _, ok := acc.Files.(filestore.Local); !ok {
		t.Errorf("local path did not resolve to local handles: %T", acc.Files)
	}
	if acc.Spec.IsRemote() {
		t.Errorf("a plain path resolved as remote: %+v", acc.Spec)
	}
}
