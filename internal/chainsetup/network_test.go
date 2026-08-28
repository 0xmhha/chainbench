package chainsetup_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// stubDriver stands in for node processes: it records what it was asked to do
// and never touches the OS.
type stubDriver struct {
	stopped  []int
	launched []int
	stopErr  error
}

func (s *stubDriver) Provision(context.Context, driver.NodeSpec) error { return nil }

func (s *stubDriver) Launch(_ context.Context, spec driver.NodeSpec) (driver.Handle, error) {
	s.launched = append(s.launched, spec.Index)
	return driver.Handle{Index: spec.Index, PID: 2000 + spec.Index}, nil
}

func (s *stubDriver) Stop(_ context.Context, h driver.Handle) error {
	if s.stopErr != nil {
		return s.stopErr
	}
	s.stopped = append(s.stopped, h.Index)
	return nil
}

// launchedNetwork seeds a workspace with the record a started network leaves
// behind: two validators on this machine, armed and running.
func launchedNetwork(t *testing.T) (dir string, d *stubDriver, deps chainsetup.Deps) {
	t.Helper()
	dir = t.TempDir()
	seedWorkspace(t, dir, "stablenet", "/opt/gstable", []node.Record{
		record(dir, 1, 8600, 1001), record(dir, 2, 8610, 1002),
	})
	d = &stubDriver{}
	deps = chainsetup.Deps{Driver: func() (driver.Driver, error) { return d, nil }}
	return dir, d, deps
}

// record is one armed validator's row: paths under the data root, an argv,
// and a pid when running.
func record(root string, index, http, pid int) node.Record {
	label := node.LabelFor(index)
	layout := node.Layout{Root: root}
	return node.Record{
		Index: index, Label: string(label), Role: string(node.RoleValidator), Host: "127.0.0.1",
		DataDir: layout.DataDir(label), ConfigPath: layout.ConfigPath(label), LogPath: layout.LogPath(label),
		Endpoints: node.Endpoints{P2P: 31000 + (index-1)*10, HTTP: http},
		Args:      []string{"--datadir", layout.DataDir(label)},
		PID:       pid,
	}
}

// seedWorkspace writes a workspace record the way the composition steps
// leave one, without running them.
func seedWorkspace(t *testing.T, dir, chain, binary string, nodes []node.Record) {
	t.Helper()
	comp, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatalf("open composition: %v", err)
	}
	st := chainsetup.State{
		Chain: chain, Binary: binary, Validators: len(nodes),
		Target: machine.Spec{DataRoot: dir},
		Nodes:  nodes,
		Steps:  map[string]chainsetup.Step{},
	}
	if err := comp.Save(st); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
}

func TestNetworkStatus_ReadsTheLaunchedSet(t *testing.T) {
	dir, _, deps := launchedNetwork(t)

	out, err := chainsetup.NetworkStatus(context.Background(), deps, chainsetup.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStatus: %v", err)
	}
	if out.Nodes.Chain != "stablenet" || len(out.Nodes.Nodes) != 2 {
		t.Errorf("node set = %+v", out.Nodes)
	}
	if out.Nodes.Nodes[1].RPCURL != "http://127.0.0.1:8610" || out.Nodes.Nodes[1].PID != 1002 {
		t.Errorf("node2 = %+v", out.Nodes.Nodes[1])
	}
}

func TestNetworkStatus_RequiresAWorkspace(t *testing.T) {
	if _, err := chainsetup.NetworkStatus(context.Background(), chainsetup.Deps{}, chainsetup.NetworkStatusIn{}); err == nil {
		t.Error("want an error without a workspace dir")
	}
	if _, err := chainsetup.NetworkStatus(context.Background(), chainsetup.Deps{}, chainsetup.NetworkStatusIn{DataDir: t.TempDir()}); err == nil {
		t.Error("want an error for a directory holding no workspace")
	}
}

func TestNetworkStop_StopsEveryLaunchedNode(t *testing.T) {
	dir, d, deps := launchedNetwork(t)

	out, err := chainsetup.NetworkStop(context.Background(), deps, chainsetup.NetworkStopIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStop: %v", err)
	}
	if out.Stopped != 2 {
		t.Errorf("stopped=%d, want 2", out.Stopped)
	}
	if len(d.stopped) != 2 {
		t.Errorf("driver saw %v", d.stopped)
	}
	// The cleared pids are what make a later status accurate.
	st, err := chainsetup.NetworkStatus(context.Background(), deps, chainsetup.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range st.Nodes.Nodes {
		if n.PID != 0 {
			t.Errorf("node%d pid = %d after stop", n.Index, n.PID)
		}
	}
}

func TestNetworkStop_NamesTheNodeItCouldNotStop(t *testing.T) {
	dir, d, deps := launchedNetwork(t)
	d.stopErr = errors.New("no such process")

	_, err := chainsetup.NetworkStop(context.Background(), deps, chainsetup.NetworkStopIn{DataDir: dir})
	if err == nil {
		t.Fatal("want an error when a node cannot be stopped")
	}
	if !strings.Contains(err.Error(), "node1") || !strings.Contains(err.Error(), "no such process") {
		t.Errorf("error should name the node and the cause, got: %v", err)
	}
}

func TestNodeStop_ClearsThePID(t *testing.T) {
	dir, d, deps := launchedNetwork(t)

	if err := chainsetup.NodeStop(context.Background(), deps, chainsetup.NodeStopIn{DataDir: dir, Index: 2}); err != nil {
		t.Fatalf("NodeStop: %v", err)
	}
	if len(d.stopped) != 1 || d.stopped[0] != 2 {
		t.Errorf("stopped the wrong node: %v", d.stopped)
	}
	st, err := chainsetup.NetworkStatus(context.Background(), deps, chainsetup.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, n := range st.Nodes.Nodes {
		if n.Index == 2 && n.PID != 0 {
			t.Errorf("node2 PID = %d, want cleared", n.PID)
		}
		if n.Index == 1 && n.PID != 1001 {
			t.Errorf("node1 PID = %d, want untouched", n.PID)
		}
	}
}

func TestNodeStop_RequiresAWorkspaceAndIndex(t *testing.T) {
	dir, _, deps := launchedNetwork(t)
	if err := chainsetup.NodeStop(context.Background(), deps, chainsetup.NodeStopIn{DataDir: dir}); err == nil {
		t.Error("want an error without an index")
	}
	if err := chainsetup.NodeStop(context.Background(), deps, chainsetup.NodeStopIn{Index: 1}); err == nil {
		t.Error("want an error without a workspace dir")
	}
}

func TestNodeStart_RelaunchesWithTheRecordedArgv(t *testing.T) {
	dir, d, deps := launchedNetwork(t)
	if err := chainsetup.NodeStop(context.Background(), deps, chainsetup.NodeStopIn{DataDir: dir, Index: 2}); err != nil {
		t.Fatalf("NodeStop: %v", err)
	}

	out, err := chainsetup.NodeStart(context.Background(), deps, chainsetup.NodeStartIn{DataDir: dir, Index: 2})
	if err != nil {
		t.Fatalf("NodeStart: %v", err)
	}
	if len(d.launched) != 1 || d.launched[0] != 2 {
		t.Errorf("launched %v, want node2", d.launched)
	}
	if out.Node.PID != 2002 || out.Node.RPCURL != "http://127.0.0.1:8610" {
		t.Errorf("refreshed node = %+v", out.Node)
	}
	// The new PID must be persisted, or a later stop has nothing to reach.
	st, err := chainsetup.NetworkStatus(context.Background(), deps, chainsetup.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, n := range st.Nodes.Nodes {
		if n.Index == 2 && n.PID != 2002 {
			t.Errorf("node2 PID = %d, want 2002", n.PID)
		}
	}
}

func TestNodeStart_RefusesARunningNode(t *testing.T) {
	dir, d, deps := launchedNetwork(t)
	_, err := chainsetup.NodeStart(context.Background(), deps, chainsetup.NodeStartIn{DataDir: dir, Index: 1})
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("a running node must not be doubled, got: %v", err)
	}
	if len(d.launched) != 0 {
		t.Errorf("driver launched %v", d.launched)
	}
}

func TestNodeStart_UnknownNodeIsNamed(t *testing.T) {
	dir, _, deps := launchedNetwork(t)

	_, err := chainsetup.NodeStart(context.Background(), deps, chainsetup.NodeStartIn{DataDir: dir, Index: 9})
	if err == nil {
		t.Fatal("want an error for a node not in the table")
	}
	if !strings.Contains(err.Error(), "node 9") {
		t.Errorf("error should name the node, got: %v", err)
	}
}

func TestNetworkRemove_StopsThenDeletes(t *testing.T) {
	dir, d, deps := launchedNetwork(t)

	out, err := chainsetup.NetworkRemove(context.Background(), deps, chainsetup.NetworkRemoveIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkRemove: %v", err)
	}
	if out.Stopped != 2 || len(d.stopped) != 2 {
		t.Errorf("stopped=%d driver=%v, want both nodes stopped first", out.Stopped, d.stopped)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("workspace still present: %v", err)
	}
}

func TestNetworkRemove_RefusesADirectoryWithNoWorkspace(t *testing.T) {
	// The guard is what stops a mistyped path from deleting something
	// unrelated, so it must hold even for a directory that clearly has files.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "important.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := chainsetup.NetworkRemove(context.Background(), chainsetup.Deps{}, chainsetup.NetworkRemoveIn{DataDir: dir})
	if err == nil {
		t.Fatal("want a refusal for a directory holding no workspace")
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("refused remove still deleted the directory: %v", statErr)
	}
}

func TestNetworkRemove_ComposedButNeverStartedIsRemovable(t *testing.T) {
	// A workspace whose nodes never started has nothing to stop, and that
	// workspace still needs to be removable.
	dir := t.TempDir()
	seedWorkspace(t, dir, "stablenet", "", []node.Record{record(dir, 1, 8600, 0)})

	out, err := chainsetup.NetworkRemove(context.Background(), chainsetup.Deps{}, chainsetup.NetworkRemoveIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkRemove: %v", err)
	}
	if out.Stopped != 0 {
		t.Errorf("stopped = %d, want 0 (nothing was started)", out.Stopped)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("workspace still present: %v", err)
	}
}
