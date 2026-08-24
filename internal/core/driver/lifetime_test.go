package driver_test

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
)

// TestLaunch_NodeOutlivesTheCallThatStartedIt: a node is not a subprocess of
// the request that started it. The caller owns it from Launch onward — through
// Stop and the pid the workspace records — so cancelling the context that
// started it must not end it.
//
// This is not hypothetical. While every command ran on a context that was never
// cancelled, binding the node to it was invisible; the moment an interrupt could
// cancel the root context, `net up` reported four nodes started and left three
// running, because cancelling on the way out killed them.
func TestLaunch_NodeOutlivesTheCallThatStartedIt(t *testing.T) {
	dir := t.TempDir()
	d := driver.NewLocalDriver()
	ctx, cancel := context.WithCancel(context.Background())

	h, err := d.Launch(ctx, driver.NodeSpec{
		Index:   1,
		Binary:  "/bin/sleep",
		Args:    []string{"30"},
		DataDir: dir,
		LogPath: filepath.Join(dir, "node1.log"),
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	t.Cleanup(func() { _ = d.Stop(context.Background(), h) })

	cancel()
	// Give a context-bound child time to die, so a pass is not a race won.
	time.Sleep(500 * time.Millisecond)

	if err := syscall.Kill(h.PID, 0); err != nil {
		t.Fatalf("node %d died when its launch context was cancelled: %v", h.PID, err)
	}

	// And Stop still ends it, because that is who owns the lifetime.
	if err := d.Stop(context.Background(), h); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(h.PID, 0) != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("node %d survived Stop", h.PID)
}

// TestLaunch_ShortLivedCommandsStillFollowTheContext: the separation is between
// a node and everything else, not a blanket detach. Datadir init is a command
// the caller waits for, so cancelling must stop it.
func TestLaunch_ShortLivedCommandsStillFollowTheContext(t *testing.T) {
	dir := t.TempDir()
	genesis := filepath.Join(dir, "genesis.json")
	if err := os.WriteFile(genesis, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := driver.InitDatadir(ctx, "/bin/sleep", dir, genesis)
	if err == nil {
		t.Fatal("init ran to completion on a cancelled context")
	}
}
