package process_test

// A restart on a remote target used to erase the log it was restarting from. This
// runs one live, because the truncation lived in a shell redirection that no unit
// test executes:
//
//	cd env/docker && ./gen-env.sh && docker compose -f build/docker-compose.yml up -d
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/core/process -run Live_RemoteRestart -v

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestLive_RemoteRestartKeepsThePreviousAttemptsLog launches a node twice on a
// server and requires the second launch to leave the first one's output in place.
//
// The "node" is /bin/sh printing a line and exiting, because what is under test
// is the launch's redirection and not any chain binary.
func TestLive_RemoteRestartKeepsThePreviousAttemptsLog(t *testing.T) {
	acc := server1Access(t)
	d := acc.Driver
	ctx := context.Background()
	logPath := "/data/chainbench/restart-live-test.log"
	t.Cleanup(func() { _ = acc.Files.Remove(ctx, logPath) })
	_ = acc.Files.Remove(ctx, logPath)

	launch := func(marker string) {
		t.Helper()
		if _, err := d.Launch(ctx, process.NodeSpec{
			Index:   1,
			Binary:  "/bin/sh",
			Args:    []string{"-c", "echo " + marker},
			LogPath: logPath,
		}); err != nil {
			t.Fatalf("launch %s: %v", marker, err)
		}
		// The launch returns as soon as the shell is backgrounded; give the one
		// echo time to land before reading.
		time.Sleep(500 * time.Millisecond)
	}

	launch("attempt-one")
	first, err := acc.Files.Read(ctx, logPath)
	if err != nil {
		t.Fatalf("read after the first launch: %v", err)
	}
	if !strings.Contains(string(first), "attempt-one") {
		t.Fatalf("the first launch wrote nothing recognisable: %q", first)
	}

	launch("attempt-two")
	both, err := acc.Files.Read(ctx, logPath)
	if err != nil {
		t.Fatalf("read after the restart: %v", err)
	}
	if !strings.Contains(string(both), "attempt-one") {
		t.Fatalf("the restart erased the previous attempt; the log is now:\n%s", both)
	}
	if !strings.Contains(string(both), "attempt-two") {
		t.Fatalf("the restart's own output is missing:\n%s", both)
	}
	// Two attempts in one file need something that says where one ends.
	if n := strings.Count(string(both), "chainbench: launch node1"); n != 2 {
		t.Fatalf("expected one attempt marker per launch, found %d:\n%s", n, both)
	}
	if strings.Index(string(both), "attempt-one") > strings.Index(string(both), "attempt-two") {
		t.Fatalf("the attempts are out of order, so the log is not an append:\n%s", both)
	}
}
