package chainsetup_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// inspectingDriver is the stub driver with the machine questions resume
// asks: which pids are alive, which run a binary, and what a pid's command
// line is.
type inspectingDriver struct {
	stubDriver
	alive    map[int]bool
	byBinary map[string][]int
	commands map[int]string
}

func (d *inspectingDriver) PIDAlive(_ context.Context, pid int) (bool, error) {
	return d.alive[pid], nil
}

func (d *inspectingDriver) FindBinary(_ context.Context, name string) ([]int, error) {
	return d.byBinary[name], nil
}

func (d *inspectingDriver) Run(_ context.Context, command string) (string, error) {
	var pid int
	if _, err := fmt.Sscanf(command, "ps -o command= -p %d", &pid); err != nil {
		return "", err
	}
	c, ok := d.commands[pid]
	if !ok {
		return "", errors.New("no such process")
	}
	return c + "\n", nil
}

func (d *inspectingDriver) Launch(ctx context.Context, spec process.NodeSpec) (process.Handle, error) {
	h, err := d.stubDriver.Launch(ctx, spec)
	if d.alive == nil {
		d.alive = map[int]bool{}
	}
	d.alive[h.PID] = true
	return h, err
}

// startedWorkspace seeds a workspace every step of which is done, with a
// recorded request and two validators recorded as running.
func startedWorkspace(t *testing.T) (dir string, d *inspectingDriver, deps chainsetup.Deps) {
	t.Helper()
	dir = t.TempDir()
	nodes := []node.Record{record(dir, 1, 8600, 1001), record(dir, 2, 8610, 1002)}
	seedWorkspace(t, dir, "stablenet", "/opt/gstable", nodes)
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := ws.State()
	for _, name := range []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init", "start"} {
		st.Steps[name] = chainsetup.Step{Done: true, Detail: "seeded"}
	}
	st.Request = &chainsetup.NetUpIn{Chain: "stablenet", Binary: "/opt/gstable", Validators: 2, Stage: chainsetup.UpStart}
	if err := comp.Save(st); err != nil {
		t.Fatal(err)
	}
	d = &inspectingDriver{alive: map[int]bool{1001: true, 1002: true}}
	deps = chainsetup.Deps{Driver: func() (process.Driver, error) { return d, nil }}
	return dir, d, deps
}

func TestNetResume_NothingToDoWhenEverythingIsRunning(t *testing.T) {
	dir, d, deps := startedWorkspace(t)
	out, err := chainsetup.NetResume(context.Background(), deps, chainsetup.NetResumeIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetResume: %v", err)
	}
	if out.Resumed != "" || len(out.Steps) != 0 || len(out.Started) != 0 {
		t.Errorf("a complete, running workspace must resume nothing: %+v", out)
	}
	if len(d.launched) != 0 || len(d.stopped) != 0 {
		t.Errorf("driver was asked to act: launched %v stopped %v", d.launched, d.stopped)
	}
	for _, line := range out.Reconciled {
		if !strings.Contains(line, "alive") {
			t.Errorf("reconcile line = %q, want alive", line)
		}
	}
}

func TestNetResume_ADeadNodeIsClearedAndBroughtBack(t *testing.T) {
	dir, d, deps := startedWorkspace(t)
	d.alive[1002] = false // node2's process died with the run

	out, err := chainsetup.NetResume(context.Background(), deps, chainsetup.NetResumeIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetResume: %v", err)
	}
	if len(out.Reconciled) != 2 || !strings.Contains(out.Reconciled[1], "pid 1002 dead, cleared") {
		t.Errorf("reconcile = %v", out.Reconciled)
	}
	if len(d.launched) != 1 || d.launched[0] != 2 {
		t.Errorf("launched %v, want only node2 brought back", d.launched)
	}
	if len(out.Started) != 1 || !strings.Contains(out.Started[0], "node2 started (pid 2002)") {
		t.Errorf("started = %v", out.Started)
	}
	// The record now says the truth: node1 as it was, node2 on its new pid.
	pids := map[int]int{}
	for _, n := range out.Nodes.Nodes.Nodes {
		pids[n.Index] = n.PID
	}
	if pids[1] != 1001 || pids[2] != 2002 {
		t.Errorf("pids after resume = %v", pids)
	}
}

func TestNetResume_AdoptsAnUnrecordedProcessOfOurs(t *testing.T) {
	// The run launched node2 and died before recording its pid. The process
	// is still there, running the argv this workspace armed it with.
	dir, d, deps := startedWorkspace(t)
	ws, _ := chainsetup.Open(dir, nil)
	st := ws.State()
	comp, _ := session.OpenComposition(dir, nil)
	st.Nodes[1].PID = 0
	if err := comp.Save(st); err != nil {
		t.Fatal(err)
	}
	// The ledger still knows node1 only; drop node2's stale entry if any.
	ours := strings.Join(append([]string{"/opt/gstable"}, st.Nodes[1].Args...), " ")
	d.byBinary = map[string][]int{"gstable": {1001, 4242, 4343}}
	d.commands = map[int]string{
		4242: "/usr/local/bin/gstable --datadir /somebody/else", // same binary, not ours
		4343: ours,
	}
	d.alive[4343] = true

	out, err := chainsetup.NetResume(context.Background(), deps, chainsetup.NetResumeIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetResume: %v", err)
	}
	if !strings.Contains(out.Reconciled[1], "pid 4343 running unrecorded, adopted") {
		t.Errorf("reconcile = %v", out.Reconciled)
	}
	if len(d.launched) != 0 {
		t.Errorf("an adopted node must not be launched again: %v", d.launched)
	}
	for _, n := range out.Nodes.Nodes.Nodes {
		if n.Index == 2 && n.PID != 4343 {
			t.Errorf("node2 pid = %d, want the adopted 4343", n.PID)
		}
	}
}

func TestNetResume_RefusesAWorkspaceWithNoRequest(t *testing.T) {
	dir, _, deps := launchedNetwork(t)
	_, err := chainsetup.NetResume(context.Background(), deps, chainsetup.NetResumeIn{DataDir: dir})
	if !errors.Is(err, chainsetup.ErrNoRequest) {
		t.Fatalf("want ErrNoRequest, got %v", err)
	}
}

func TestNetResume_ContinuesFromTheFirstUnfinishedStep(t *testing.T) {
	// A run that died after provision: the workspace holds every artifact
	// but no datadir was initialized and nothing started.
	dir := t.TempDir()
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	bin := fakeNodeBinary(t, dir)
	deps := chainsetup.Deps{Clock: fixedClock()}
	if _, err := chainsetup.NetUp(context.Background(), deps, chainsetup.NetUpIn{
		DataDir: dir, Stage: chainsetup.UpDeploy, Chain: "stablenet", KeysDir: keysAbs, Validators: 2, Binary: bin,
	}); err != nil {
		t.Fatalf("chain up --stage deploy: %v", err)
	}
	// What was asked is on the record, without the workspace's own location.
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := ws.State().Request
	if req == nil || req.Validators != 2 || req.Chain != "stablenet" || req.DataDir != "" {
		t.Fatalf("recorded request = %+v", req)
	}
	// The run was asked to start; the record says it stopped at provision.
	comp, _ := session.OpenComposition(dir, nil)
	st := ws.State()
	st.Request.Stage = chainsetup.UpStart
	if err := comp.Save(st); err != nil {
		t.Fatal(err)
	}

	out, err := chainsetup.NetResume(context.Background(), deps, chainsetup.NetResumeIn{DataDir: dir})
	t.Cleanup(func() {
		_, _ = chainsetup.NetworkStop(context.Background(), deps, chainsetup.NetworkStopIn{DataDir: dir})
	})
	if err != nil {
		t.Fatalf("NetResume: %v (steps %v)", err, out.Steps)
	}
	if out.Resumed != "init" {
		t.Errorf("resumed from %q, want init (the first step provision did not reach)", out.Resumed)
	}
	if len(out.Steps) != 2 || !strings.HasPrefix(out.Steps[0], "init: ") || !strings.HasPrefix(out.Steps[1], "start: ") {
		t.Errorf("steps = %v, want init then start", out.Steps)
	}
	for _, n := range out.Nodes.Nodes.Nodes {
		if n.PID == 0 {
			t.Errorf("node%d has no pid after resume", n.Index)
		}
	}
}

// fakeNodeBinary is a script that answers `init` and otherwise stays up.
func fakeNodeBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "fakegeth")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit 0; fi\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestSetLock_SerializesAllocation(t *testing.T) {
	// Two allocators on one set: the second sees the first's lock as live
	// (same host, this very process would nest — so it is a foreign pid).
	root := t.TempDir()
	t.Setenv("HOME", root)
	path := filepath.Join(root, ".chainbench", "local.lock")
	held, _, state, err := session.AcquireLock(path, "test", nil)
	if err != nil || state != session.LockFree {
		t.Fatalf("first acquire: state=%s err=%v", state, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock not written: %v", err)
	}
	// The same process nests rather than waits.
	again, _, state, err := session.AcquireLock(path, "test", nil)
	if err != nil || state != session.LockLive {
		t.Fatalf("nested acquire: state=%s err=%v", state, err)
	}
	_ = again.Release()
	if _, err := os.Stat(path); err != nil {
		t.Fatal("a nested release must not drop the outer lock")
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the outer release must remove the lock")
	}
}
