package driver_test

// The remote half of the inspector/commander capabilities, verified against
// the local docker servers:
//
//	cd env/docker && ./gen-env.sh && docker compose -f build/docker-compose.yml up -d
//	CHAINBENCH_DOCKER_FLEET=$PWD/env/docker/build go test ./internal/core/driver -run Live_ -v

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/machine"
	netmapmod "github.com/0xmhha/chainbench/internal/netmap"
)

// server1Driver opens server1 through the netmap module — the same wiring
// production uses — and returns its driver.
func server1Driver(t *testing.T) driver.Driver {
	t.Helper()
	build := os.Getenv("CHAINBENCH_DOCKER_FLEET")
	if build == "" {
		t.Skip("set CHAINBENCH_DOCKER_FLEET=<repo>/env/docker/build with the fleet running (env/docker/gen-env.sh)")
	}
	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1")
	acc, err := netmapmod.Opener{
		ServerSet: filepath.Join(build, "server-set.yaml"), Docker: true, Env: os.Getenv,
	}.Open(machine.Spec{Kind: machine.KindServer, Server: "server1", DataRoot: "/data/chainbench"})
	if err != nil {
		t.Fatalf("open server1: %v", err)
	}
	return acc.Driver
}

// TestLive_RemoteInspectorAnswersOnTheMachine: sshd runs on every server, pid
// 1 exists, and a name nothing runs is an empty answer.
func TestLive_RemoteInspectorAnswersOnTheMachine(t *testing.T) {
	d := server1Driver(t)
	insp, ok := d.(driver.ProcessInspector)
	if !ok {
		t.Fatalf("remote driver lost the inspector capability: %T", d)
	}
	ctx := context.Background()

	if alive, err := insp.PIDAlive(ctx, 1); err != nil || !alive {
		t.Fatalf("pid 1 on the server reported dead (alive=%v err=%v)", alive, err)
	}
	pids, err := insp.FindBinary(ctx, "sshd")
	if err != nil || len(pids) == 0 {
		t.Fatalf("sshd not found on the server (pids=%v err=%v)", pids, err)
	}
	if none, err := insp.FindBinary(ctx, "no-such-binary"); err != nil || len(none) != 0 {
		t.Fatalf("no match must be empty, got %v / %v", none, err)
	}
}

// TestLive_RemoteCommanderReturnsOutput: the answer is the command's stdout
// from the machine itself.
func TestLive_RemoteCommanderReturnsOutput(t *testing.T) {
	d := server1Driver(t)
	cmd, ok := d.(driver.Commander)
	if !ok {
		t.Fatalf("remote driver lost the commander capability: %T", d)
	}
	out, err := cmd.Run(context.Background(), "hostname")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "server1") {
		t.Errorf("hostname = %q, want server1", strings.TrimSpace(out))
	}
}
