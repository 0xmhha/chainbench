package chainsetup_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// failingLaunchDriver launches every node but one, which it refuses the way a
// full disk refused node13's log directory.
type failingLaunchDriver struct {
	inspectingDriver
	failAt int
}

func (d *failingLaunchDriver) Launch(ctx context.Context, spec process.NodeSpec) (process.Handle, error) {
	if spec.Index == d.failAt {
		return process.Handle{}, errors.New("mkdir: cannot create directory: No space left on device")
	}
	return d.inspectingDriver.Launch(ctx, spec)
}

// unstartedWorkspace seeds a workspace composed up to the launch: every step
// but start done, three nodes, none running.
func unstartedWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	nodes := []node.Record{record(dir, 1, 8600, 0), record(dir, 2, 8610, 0), record(dir, 3, 8620, 0)}
	// What the launch checks for before it starts anything: the binary, and
	// each node's config and datadir.
	bin := filepath.Join(dir, "gstable")
	must(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	for _, ns := range nodes {
		must(t, os.MkdirAll(ns.DataDir, 0o755))
		must(t, os.WriteFile(ns.ConfigPath, []byte("# config\n"), 0o644))
	}
	seedWorkspace(t, dir, "stablenet", bin, nodes)
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := ws.State()
	keys, err := filepath.Abs("../../presets/keys")
	must(t, err)
	st.KeysDir = keys
	for _, name := range []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init"} {
		st.Steps[name] = chainsetup.Step{Done: true, Detail: "seeded"}
	}
	st.Request = &chainsetup.ChainUpIn{Chain: "stablenet", Binary: bin, BPCount: 3, Stage: chainsetup.UpStart}
	if err := comp.Save(st); err != nil {
		t.Fatal(err)
	}
	return dir
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// TestLaunch_AFailedLaunchStopsTheNodesItStarted: a launch that dies at the
// third node takes the first two down again and leaves no pid in the record.
// Before, they kept running with their pids recorded, the run that failed
// handed back nothing to stop them with, and the next run on the same servers
// found its ports taken.
func TestLaunch_AFailedLaunchStopsTheNodesItStarted(t *testing.T) {
	dir := unstartedWorkspace(t)
	d := &failingLaunchDriver{failAt: 3}
	deps := chainsetup.Deps{Driver: func() (process.Driver, error) { return d, nil }}

	out, err := verb.ChainResume(context.Background(), deps, verb.ChainResumeIn{DataDir: dir})
	if err == nil {
		t.Fatal("a launch that refused node3 reported success")
	}
	if !strings.Contains(err.Error(), "No space left on device") {
		t.Errorf("the launch's own error was lost: %v", err)
	}
	slices.Sort(d.stopped)
	if !slices.Equal(d.launched, []int{1, 2}) || !slices.Equal(d.stopped, []int{1, 2}) {
		t.Errorf("launched %v, stopped %v — want node1 and node2 started and then stopped", d.launched, d.stopped)
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ns := range ws.State().Nodes {
		if ns.PID != 0 {
			t.Errorf("node%d still has pid %d recorded after the rollback", ns.Index, ns.PID)
		}
	}
	if !slices.ContainsFunc(out.Steps, func(s string) bool { return strings.HasPrefix(s, "rollback: 2 node(s)") }) {
		t.Errorf("the rollback is not in the steps a run prints: %v", out.Steps)
	}
}

// TestRollBackLaunch_LeavesWhatWasAlreadyRunning: a node that was up before the
// launch began is not the launch's to stop.
func TestRollBackLaunch_LeavesWhatWasAlreadyRunning(t *testing.T) {
	dir, d, deps := launchedNetwork(t)
	detail, err := chainsetup.InWorkspace(deps, dir, func(ws *chainsetup.Workspace) (string, error) {
		return ws.RollBackLaunch(context.Background(), map[int]bool{1: true})
	})
	if err != nil {
		t.Fatalf("RollBackLaunch: %v", err)
	}
	if !slices.Equal(d.stopped, []int{2}) {
		t.Errorf("stopped %v, want only node2", d.stopped)
	}
	if !strings.HasPrefix(detail, "1 node(s)") {
		t.Errorf("detail = %q", detail)
	}
}
