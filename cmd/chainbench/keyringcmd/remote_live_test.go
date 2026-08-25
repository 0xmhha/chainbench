package keyringcmd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/machine"
	netmapmod "github.com/0xmhha/chainbench/internal/netmap"
)

// The remote half of the keyring verification matrix, live against the local
// docker fleet posing as servers. Gated so CI (which has no fleet) skips it:
//
//	cd env/docker && ./gen-env.sh && docker compose -f build/docker-compose.yml up -d
//	CHAINBENCH_DOCKER_FLEET=$PWD/env/docker/build go test ./cmd/chainbench/ -run Live_Keyring -v
//
// The local half (new/add/list/show/export, hex and file imports, env
// selection) runs unconditionally in keyring_test.go.
func fleetBuildDir(t *testing.T) string {
	t.Helper()
	build := os.Getenv("CHAINBENCH_DOCKER_FLEET")
	if build == "" {
		t.Skip("set CHAINBENCH_DOCKER_FLEET=<repo>/env/docker/build with the fleet running (env/docker/gen-env.sh)")
	}
	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1")
	// Access mirrors the real fleet: user + password. The srv:// path reads
	// them from the server set; the direct user@host form reads the password
	// from the environment, so it is exported here from the same file.
	if cfg, err := netmapmod.LoadSet(filepath.Join(build, "server-set.yaml")); err == nil {
		if srv, err := cfg.ByName("server1"); err == nil {
			t.Setenv("CHAINBENCH_REMOTE_PASS", srv.SSH.Password)
		}
	}
	return build
}

// plantOnServer1 writes content to path on server1 through the harness's own
// remote stack — the user@host target form resolved with the address map — so
// planting the fixture itself exercises the direct-host form and the remote
// write direction.
func plantOnServer1(t *testing.T, build, remotePath string, content []byte) {
	t.Helper()
	inv := filepath.Join(build, "server-set.yaml")
	cfg, err := netmapmod.LoadSet(inv)
	if err != nil {
		t.Fatalf("load server set: %v", err)
	}
	srv, err := cfg.ByName("server1")
	if err != nil {
		t.Fatal(err)
	}
	spec := machine.Spec{
		Kind: machine.KindRemote, User: srv.SSH.User, Host: srv.Host, DataRoot: "/data/chainbench",
	}
	// The direct user@host form authenticates from the environment; the dial
	// still translates through the module's one address map.
	o := netmapmod.Opener{ServerSet: inv, Docker: true, Env: os.Getenv}
	tgt, err := o.Open(spec)
	if err != nil {
		t.Fatalf("resolve user@host target: %v", err)
	}
	if err := tgt.Files.Write(context.Background(), remotePath, content, 0o600); err != nil {
		t.Fatalf("write fixture to server1: %v", err)
	}
}

// TestLive_KeyringImportsARawKeyFromAServer covers srv:// + --docker end to
// end: a key that exists only on the server comes back as a derived identity,
// and the translation is reported. The fixture is a key exported from a local
// ring, so the imported address has a known right answer.
func TestLive_KeyringImportsARawKeyFromAServer(t *testing.T) {
	build := fleetBuildDir(t)
	inv := filepath.Join(build, "server-set.yaml")

	local := newRing(t)
	want := addressOf(t, local, "node1")
	plantOnServer1(t, build, "/data/chainbench/live-test/rawkey",
		[]byte(strings.TrimPrefix(exportKey(t, local, "node1"), "0x")))

	dir := filepath.Join(t.TempDir(), "ring")
	out, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "fromsrv",
		"--from", "srv://server1/data/chainbench/live-test/rawkey",
		"--server-set", inv, "--docker")
	if err != nil {
		t.Fatalf("srv:// import: %v\n%s", err, out)
	}
	if !strings.Contains(out, "docker: dialing") {
		t.Errorf("the applied translation was not reported:\n%s", out)
	}
	if got := addressOf(t, dir, "fromsrv"); got != want {
		t.Errorf("imported address %s, want %s — the key changed in transit", got, want)
	}
}

// TestLive_KeyringImportsAKeystoreFromAServer covers the encrypted case: a
// keystore JSON on the server, decrypted with --password on import.
func TestLive_KeyringImportsAKeystoreFromAServer(t *testing.T) {
	build := fleetBuildDir(t)
	inv := filepath.Join(build, "server-set.yaml")

	local := newRing(t) // keystores encrypted with the default password "1"
	want := addressOf(t, local, "node1")
	ksDir := filepath.Join(local, "node1", "keystore")
	entries, err := os.ReadDir(ksDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("local ring has no keystore: %v", err)
	}
	ks, err := os.ReadFile(filepath.Join(ksDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	plantOnServer1(t, build, "/data/chainbench/live-test/keystore.json", ks)

	dir := filepath.Join(t.TempDir(), "ring")
	out, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "fromks",
		"--from", "srv://server1/data/chainbench/live-test/keystore.json",
		"--password", "1", "--server-set", inv, "--docker")
	if err != nil {
		t.Fatalf("keystore import: %v\n%s", err, out)
	}
	if got := addressOf(t, dir, "fromks"); got != want {
		t.Errorf("imported address %s, want %s", got, want)
	}
}

func addressOf(t *testing.T, ring, name string) string {
	t.Helper()
	out, err := run(t, "keyring", "show", "--keyring-dir", ring, "--name", name, "--json")
	if err != nil {
		t.Fatalf("keyring show: %v\n%s", err, out)
	}
	var e app.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &e); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	return e.Address
}

// TestLive_KeyringCreatesARingOnAServer pins the remote ring end to end: a
// ring named srv://server1/... is created ON the server through the file
// seam, its files land there (checked through the same remote stack), and it
// reads back and verifies remotely — the same contracts the local ring
// holds, at a different location. The path is per-process so reruns never
// collide with a leftover index.
func TestLive_KeyringCreatesARingOnAServer(t *testing.T) {
	build := fleetBuildDir(t)
	inv := filepath.Join(build, "server-set.yaml")
	onServer := fmt.Sprintf("/data/chainbench/live-ring-%d", os.Getpid())
	ring := "srv://server1" + onServer

	out, err := run(t, "keyring", "new", "--keyring-dir", ring, "--server-set", inv, "--docker",
		"--count", "2", "--with-bls")
	if err != nil {
		t.Fatalf("remote ring new: %v\n%s", err, out)
	}
	if !strings.Contains(out, "docker: dialing") {
		t.Errorf("the applied translation was not reported:\n%s", out)
	}

	// The key material exists ON the server, checked through the same stack.
	cfg, err := netmapmod.LoadSet(inv)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := cfg.ByName("server1")
	if err != nil {
		t.Fatal(err)
	}
	spec := machine.Spec{Kind: machine.KindRemote, User: srv.SSH.User, Host: srv.Host, DataRoot: "/data/chainbench"}
	tgt, err := netmapmod.Opener{ServerSet: inv, Docker: true, Env: os.Getenv}.Open(spec)
	if err != nil {
		t.Fatal(err)
	}
	exists, err := tgt.Files.Exists(context.Background(), onServer+"/node1/nodekey")
	if err != nil || !exists {
		t.Fatalf("node1 nodekey not on the server (exists=%v err=%v)", exists, err)
	}

	// Read back and verify remotely: derivation must match what was written.
	if out, err := run(t, "keyring", "list", "--keyring-dir", ring, "--server-set", inv, "--docker", "--verify"); err != nil {
		t.Fatalf("remote list --verify: %v\n%s", err, out)
	}
}

// TestLive_KeyringClonesARingFromAServer pins the whole-ring pull: a ring is
// created ON the server, then --from-ring clones it here in one command —
// labels intact, validator declaration carried, and every entry verified
// against the server's index before anything lands locally.
func TestLive_KeyringClonesARingFromAServer(t *testing.T) {
	build := fleetBuildDir(t)
	inv := filepath.Join(build, "server-set.yaml")
	onServer := fmt.Sprintf("/data/chainbench/live-clone-src-%d", os.Getpid())
	remote := "srv://server1" + onServer

	out, err := run(t, "keyring", "new", "--keyring-dir", remote, "--server-set", inv, "--docker",
		"--count", "3", "--validators", "2", "--with-bls")
	if err != nil {
		t.Fatalf("seed remote ring: %v\n%s", err, out)
	}

	local := filepath.Join(t.TempDir(), "pulled")
	out, err = run(t, "keyring", "import", "--keyring-dir", local,
		"--from-ring", remote, "--server-set", inv, "--docker")
	if err != nil {
		t.Fatalf("clone from server: %v\n%s", err, out)
	}
	if !strings.Contains(out, "docker: dialing") {
		t.Errorf("the applied translation was not reported:\n%s", out)
	}
	if !strings.Contains(out, "2 validators") {
		t.Errorf("validator declaration did not travel:\n%s", out)
	}

	// Same identities on both sides, and the local copy self-verifies.
	for _, name := range []string{"node1", "node2", "node3"} {
		want := addressOfRemote(t, remote, inv, name)
		if got := addressOf(t, local, name); !strings.EqualFold(got, want) {
			t.Errorf("%s: local %s != remote %s", name, got, want)
		}
	}
	if out, err := run(t, "keyring", "list", "--keyring-dir", local, "--verify"); err != nil {
		t.Fatalf("local verify after clone: %v\n%s", err, out)
	}
}

func addressOfRemote(t *testing.T, ring, inv, name string) string {
	t.Helper()
	out, err := run(t, "keyring", "show", "--keyring-dir", ring, "--server-set", inv, "--docker",
		"--name", name, "--json")
	if err != nil {
		t.Fatalf("remote keyring show: %v\n%s", err, out)
	}
	var e app.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &e); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	return e.Address
}
