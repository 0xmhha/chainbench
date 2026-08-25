package chainsetup

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
)

// recordingDriver counts launches and hands back incrementing pids, standing in
// for a chain binary.
type recordingDriver struct {
	launched []int
	nextPID  int
}

func (d *recordingDriver) Provision(context.Context, driver.NodeSpec) error { return nil }
func (d *recordingDriver) Launch(_ context.Context, spec driver.NodeSpec) (driver.Handle, error) {
	d.launched = append(d.launched, spec.Index)
	d.nextPID++
	return driver.Handle{PID: 10000 + d.nextPID}, nil
}
func (d *recordingDriver) Stop(context.Context, driver.Handle) error { return nil }

func TestNodeController_StartRelaunchesWithTheOriginalArming(t *testing.T) {
	drv := &recordingDriver{}
	c := NewNodeController(LocalLauncher{Driver: drv}, process.New())
	// Seed the controller as a launch would, without needing a preset on disk.
	// PID 0 is the "stopped" state, which is when a restart is legal.
	c.record(
		supervisor.LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 2, PID: 0}}}},
		[]driver.NodeSpec{{Index: 2, DataDir: "/tmp/n2", Args: []string{"--nodekey", "k2"}}},
	)

	if err := c.Start(context.Background(), node.Node{Index: 2}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(drv.launched) != 1 || drv.launched[0] != 2 {
		t.Fatalf("launched = %v, want [2]", drv.launched)
	}
}

func TestNodeController_StartUnknownNodeIsAnError(t *testing.T) {
	c := NewNodeController(LocalLauncher{Driver: &recordingDriver{}}, process.New())
	err := c.Start(context.Background(), node.Node{Index: 9})
	if err == nil {
		t.Fatal("expected an error for a node this run never launched")
	}
	if !strings.Contains(err.Error(), "node9") {
		t.Fatalf("error should name the node, got: %v", err)
	}
}

func TestNodeController_StopWithoutATrackedProcessIsAnError(t *testing.T) {
	c := NewNodeController(LocalLauncher{Driver: &recordingDriver{}}, process.New())
	if err := c.Stop(context.Background(), node.Node{Index: 1}); err == nil {
		t.Fatal("expected an error when there is no tracked process")
	}
}

func TestNodeController_StopThenStartRoundTrip(t *testing.T) {
	pid := startDetachedSleeper(t)
	procs := process.New()
	procs.Track(pid, "node1")

	drv := &recordingDriver{}
	c := NewNodeController(LocalLauncher{Driver: drv}, procs)
	c.record(
		supervisor.LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1, PID: pid}}}},
		[]driver.NodeSpec{{Index: 1, DataDir: "/tmp/n1"}},
	)

	n := node.Node{Index: 1, PID: pid}
	if err := c.Stop(context.Background(), n); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if process.Alive(pid) {
		t.Fatal("node process still alive after Stop")
	}
	// Start is allowed now that the process is gone, and re-arms from the record.
	if err := c.Start(context.Background(), n); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	if len(drv.launched) != 1 {
		t.Fatalf("relaunch count = %d, want 1", len(drv.launched))
	}
}

// startDetachedSleeper launches a detached long-lived process and returns its
// pid, mirroring how chainbench launches nodes.
func startDetachedSleeper(t *testing.T) int {
	t.Helper()
	out, err := exec.Command("sh", "-c", "sleep 30 >/dev/null 2>&1 & echo $!").Output()
	if err != nil {
		t.Fatalf("start sleeper: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse pid %q: %v", out, err)
	}
	t.Cleanup(func() {
		if process.Alive(pid) {
			m := process.New()
			m.Track(pid, "cleanup")
			_ = m.StopAll(time.Second)
		}
	})
	return pid
}
