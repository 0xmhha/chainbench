//go:build e2e

// This E2E is gated behind the `e2e` build tag and skips unless a remote SSH
// endpoint is provided, so `go test ./...` never runs it. It drives the SSH
// RemoteDriver against a real sshd (locally: the Docker stand-in in
// tests/remote/sshd) and asserts the provision -> launch -> stop path works over
// SSH.
//
// Run it with tests/remote/sshd/run.sh, or manually:
//
//	CHAINBENCH_REMOTE_HOST=127.0.0.1 CHAINBENCH_REMOTE_PORT=2222 \
//	CHAINBENCH_REMOTE_USER=chainbench CHAINBENCH_REMOTE_PASS=chainbench \
//	go test -tags e2e -run TestRemoteDriver_E2E -v ./pkg/core/driver/
package driver_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// The host and port this e2e dials. They are a test gate like
// testkit.EnvDockerFleet, not a product setting: the transport itself takes
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
	run := driver.SSHRunner(creds, hostKey)
	d := driver.NewRemoteDriver(run)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	base := "/home/" + user + "/chainbench-e2e"
	spec := driver.NodeSpec{
		Index:         1,
		Binary:        "/usr/local/bin/fakenode",
		DataDir:       base + "/node1",
		ConfigPath:    base + "/node1/config.toml",
		ConfigContent: []byte("[Node]\nname = \"e2e\"\n"),
		LogPath:       base + "/logs/node1.log",
		Args:          []string{"run"},
	}
	// clean any prior run
	_, _ = remote.Exec(ctx, creds, hostKey, "rm -rf "+base)

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
