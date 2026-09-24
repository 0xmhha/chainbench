//go:build e2e

// This E2E is gated behind the `e2e` build tag and skips unless a remote SSH
// endpoint is provided, so `go test ./...` never runs it. It drives the SSH
// RemoteDriver against a real sshd and asserts the provision -> init -> launch
// -> stop path works over SSH.
//
// Any host it can log into will do, and it brings its own node binary (see
// standInNode), so nothing has to be installed on the far side. The env/docker
// fleet is the one at hand — its server1 publishes sshd on 2201, and the login
// is the first account in env/docker/accounts.env:
//
//	set -a; . env/docker/accounts.env; set +a
//	CHAINBENCH_REMOTE_HOST=127.0.0.1 CHAINBENCH_REMOTE_PORT=2201 \
//	CHAINBENCH_REMOTE_USER="${DEV_ACCOUNTS%%:*}" \
//	CHAINBENCH_REMOTE_PASS="$DEV_ACCOUNTS_PASSWORD" \
//	go test -tags e2e -run TestRemoteDriver_E2E -v ./internal/core/process/
package process_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// The host and port this e2e dials. They are a test gate like
// testkit.EnvDockerServers, not a product setting: the transport itself takes
// its address from the caller.
const (
	envRemoteHost = "CHAINBENCH_REMOTE_HOST"
	envRemotePort = "CHAINBENCH_REMOTE_PORT"
)

func TestRemoteDriver_E2E(t *testing.T) {
	host := os.Getenv(envRemoteHost)
	user := os.Getenv(remote.EnvUser)
	pass := os.Getenv(remote.EnvPass)
	if host == "" || user == "" || pass == "" {
		t.Skip("set " + envRemoteHost + "/" + remote.EnvUser + "/" + remote.EnvPass + " (and optionally " + envRemotePort + ") to run")
	}
	port, _ := strconv.Atoi(os.Getenv(envRemotePort))
	if port == 0 {
		port = 22
	}
	creds := remote.Credentials{User: user, Host: host, Port: port, Password: pass}

	// The container's host key is ephemeral; accept it insecurely for the test.
	hostKey, err := remote.HostKeyPolicy{InsecureHostKey: true}.Callback()
	if err != nil {
		t.Fatalf("host key: %v", err)
	}
	run := process.SSHRunner(creds, hostKey)
	d := process.NewRemoteDriver(run)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	base := "/home/" + user + "/chainbench-e2e"
	// clean any prior run
	_, _ = remote.Exec(ctx, creds, hostKey, "rm -rf "+base)

	spec := process.NodeSpec{
		Index:         1,
		Binary:        standInNode(ctx, t, creds, hostKey, base),
		DataDir:       base + "/node1",
		ConfigPath:    base + "/node1/config.toml",
		ConfigContent: []byte("[Node]\nname = \"e2e\"\n"),
		LogPath:       base + "/logs/node1.log",
		Args:          []string{"run"},
	}

	// Provision: datadir + config written on the remote host.
	if err := d.Provision(ctx, spec); err != nil {
		t.Fatalf("provision: %v", err)
	}
	res, err := remote.Exec(ctx, creds, hostKey, "cat "+spec.ConfigPath)
	if err != nil || !strings.Contains(res.Stdout, `name = "e2e"`) {
		t.Fatalf("config not written remotely: %q (%v)", res.Stdout, err)
	}

	// InitDatadir: the genesis is shipped and `init` runs on the remote host
	// (the fake node's `init` exits 0). Assert the genesis landed remotely.
	if err := d.InitDatadir(ctx, spec, []byte(`{"config":{"chainId":1}}`)); err != nil {
		t.Fatalf("init: %v", err)
	}
	gen, err := remote.Exec(ctx, creds, hostKey, "cat "+spec.DataDir+"/genesis.json")
	if err != nil || !strings.Contains(gen.Stdout, "chainId") {
		t.Fatalf("genesis not shipped remotely: %q (%v)", gen.Stdout, err)
	}

	// Launch: the fake node runs; its PID is returned.
	h, err := d.Launch(ctx, spec)
	if err != nil || h.PID <= 0 {
		t.Fatalf("launch: pid=%d err=%v", h.PID, err)
	}
	up, _ := remote.Exec(ctx, creds, hostKey, "kill -0 "+strconv.Itoa(h.PID)+" && echo UP")
	if !strings.Contains(up.Stdout, "UP") {
		t.Fatalf("launched process not running: pid %d", h.PID)
	}

	// Stop: the process is gone afterwards.
	if err := d.Stop(ctx, h); err != nil {
		t.Fatalf("stop: %v", err)
	}
	time.Sleep(time.Second)
	gone, _ := remote.Exec(ctx, creds, hostKey, "kill -0 "+strconv.Itoa(h.PID)+" 2>/dev/null && echo UP || echo GONE")
	if !strings.Contains(gone.Stdout, "GONE") {
		t.Errorf("process still running after stop: pid %d", h.PID)
	}

	_, _ = remote.Exec(ctx, creds, hostKey, "rm -rf "+base)
}

// standInNode writes the node binary this drives and answers its path.
//
// The driver is what is under test, not a chain: `init` has to exit 0 and a
// launch has to leave a process alive long enough for Stop to kill it, and a
// four-line script does both. A real binary would only add a build to the
// prerequisites.
//
// It used to come from a container of its own — tests/remote/sshd shipped an
// sshd AND a /usr/local/bin/fakenode, and #130 retired that suite in July.
// This test kept naming the path and failed with "exit 127: no such file" from
// then on, behind a gate that skips when the SSH variables are unset, so
// nothing said so. Writing it here needs no image and no root: any host this
// test can log into can hold it, including the env/docker fleet.
func standInNode(ctx context.Context, t *testing.T, creds remote.Credentials, hostKey ssh.HostKeyCallback, base string) string {
	t.Helper()
	path := base + "/stand-in-node"
	script := "#!/bin/sh\ncase \"$1\" in\n  init) exit 0 ;;\n  *) exec sleep 3600 ;;\nesac\n"
	cmd := fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s && chmod +x %s",
		base, shellQuote(script), path, path)
	if res, err := remote.Exec(ctx, creds, hostKey, cmd); err != nil {
		t.Fatalf("write the stand-in node on the remote host: %v (%s)", err, res.Stderr)
	}
	return path
}

// shellQuote wraps s for a POSIX shell in single quotes, which take everything
// literally except a single quote of their own.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
