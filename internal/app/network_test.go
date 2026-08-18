package app_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/state"
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

// launchedNetwork seeds a data root with the state a launch leaves behind.
func launchedNetwork(t *testing.T) (dir string, d *stubDriver, deps app.Deps) {
	t.Helper()
	dir = t.TempDir()
	ns := node.NodeSet{
		Chain: "stablenet", Network: "local",
		Nodes: []node.Node{
			{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: "http://127.0.0.1:8600", PID: 1001},
			{Index: 2, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: "http://127.0.0.1:8610", PID: 1002},
		},
	}
	if err := state.SaveNodeSet(dir, ns); err != nil {
		t.Fatalf("seed node set: %v", err)
	}
	specs := []driver.NodeSpec{
		{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", Ports: node.Endpoints{HTTP: 8600}},
		{Index: 2, Role: node.RoleValidator, Host: "127.0.0.1", Ports: node.Endpoints{HTTP: 8610}},
	}
	if err := state.SaveNodeSpecs(dir, specs); err != nil {
		t.Fatalf("seed node specs: %v", err)
	}
	d = &stubDriver{}
	deps = app.Deps{Driver: func() (driver.Driver, error) { return d, nil }}
	return dir, d, deps
}

func TestNetworkStatus_ReadsTheLaunchedSet(t *testing.T) {
	dir, _, deps := launchedNetwork(t)

	out, err := app.NetworkStatus(context.Background(), deps, app.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStatus: %v", err)
	}
	if out.Nodes.Chain != "stablenet" || len(out.Nodes.Nodes) != 2 {
		t.Errorf("node set = %+v", out.Nodes)
	}
}

func TestNetworkStatus_RequiresADataDir(t *testing.T) {
	if _, err := app.NetworkStatus(context.Background(), app.Deps{}, app.NetworkStatusIn{}); err == nil {
		t.Error("want an error without a data dir")
	}
}

func TestNetworkStop_StopsEveryLaunchedNode(t *testing.T) {
	dir, d, deps := launchedNetwork(t)

	out, err := app.NetworkStop(context.Background(), deps, app.NetworkStopIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStop: %v", err)
	}
	if out.Stopped != 2 || len(out.Failed) != 0 {
		t.Errorf("stopped=%d failed=%v, want 2/none", out.Stopped, out.Failed)
	}
	if len(d.stopped) != 2 {
		t.Errorf("driver saw %v", d.stopped)
	}
}

func TestNetworkStop_ReportsFailuresWithoutFailingTheCall(t *testing.T) {
	// A stop that cannot reach a node is reported per node, not returned as the
	// call's error: the caller still needs the rest of the outcome.
	dir, d, deps := launchedNetwork(t)
	d.stopErr = errors.New("no such process")

	out, err := app.NetworkStop(context.Background(), deps, app.NetworkStopIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStop should not fail on a per-node error: %v", err)
	}
	if out.Stopped != 0 || len(out.Failed) != 2 {
		t.Errorf("stopped=%d failed=%v, want 0/two", out.Stopped, out.Failed)
	}
}

func TestNodeStop_ClearsThePID(t *testing.T) {
	dir, d, deps := launchedNetwork(t)

	if err := app.NodeStop(context.Background(), deps, app.NodeStopIn{DataDir: dir, Index: 2}); err != nil {
		t.Fatalf("NodeStop: %v", err)
	}
	if len(d.stopped) != 1 || d.stopped[0] != 2 {
		t.Errorf("stopped the wrong node: %v", d.stopped)
	}
	// The cleared PID is what makes a later status or start accurate.
	ns, err := state.LoadNodeSet(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, n := range ns.Nodes {
		if n.Index == 2 && n.PID != 0 {
			t.Errorf("node2 PID = %d, want cleared", n.PID)
		}
		if n.Index == 1 && n.PID != 1001 {
			t.Errorf("node1 PID = %d, want untouched", n.PID)
		}
	}
}

func TestNodeStop_RequiresADataDirAndIndex(t *testing.T) {
	dir, _, deps := launchedNetwork(t)
	if err := app.NodeStop(context.Background(), deps, app.NodeStopIn{DataDir: dir}); err == nil {
		t.Error("want an error without an index")
	}
	if err := app.NodeStop(context.Background(), deps, app.NodeStopIn{Index: 1}); err == nil {
		t.Error("want an error without a data dir")
	}
}

func TestNodeStart_RelaunchesFromTheSavedSpec(t *testing.T) {
	dir, d, deps := launchedNetwork(t)
	if err := app.NodeStop(context.Background(), deps, app.NodeStopIn{DataDir: dir, Index: 2}); err != nil {
		t.Fatalf("NodeStop: %v", err)
	}

	out, err := app.NodeStart(context.Background(), deps, app.NodeStartIn{DataDir: dir, Index: 2})
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
	ns, err := state.LoadNodeSet(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, n := range ns.Nodes {
		if n.Index == 2 && n.PID != 2002 {
			t.Errorf("node2 PID = %d, want 2002", n.PID)
		}
	}
}

func TestNodeStart_WithoutASavedSpecNamesTheNode(t *testing.T) {
	dir, _, deps := launchedNetwork(t)

	_, err := app.NodeStart(context.Background(), deps, app.NodeStartIn{DataDir: dir, Index: 9})
	if err == nil {
		t.Fatal("want an error for a node with no saved spec")
	}
	if !strings.Contains(err.Error(), "node9") {
		t.Errorf("error should name the node, got: %v", err)
	}
}

func TestNetworkRemove_StopsThenDeletes(t *testing.T) {
	dir, d, deps := launchedNetwork(t)

	out, err := app.NetworkRemove(context.Background(), deps, app.NetworkRemoveIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkRemove: %v", err)
	}
	if out.Stopped != 2 || len(d.stopped) != 2 {
		t.Errorf("stopped=%d driver=%v, want both nodes stopped first", out.Stopped, d.stopped)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("data dir still present: %v", err)
	}
}

func TestNetworkRemove_RefusesADirectoryWithNoSetupArtifact(t *testing.T) {
	// The guard is what stops a mistyped path from deleting something
	// unrelated, so it must hold even for a directory that clearly has files.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "important.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := app.NetworkRemove(context.Background(), app.Deps{}, app.NetworkRemoveIn{DataDir: dir})
	if err == nil {
		t.Fatal("want a refusal for a non-data-dir")
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("refused remove still deleted the directory: %v", statErr)
	}
}

func TestNetworkRemove_ProvisionedButNeverLaunchedIsRemovable(t *testing.T) {
	// A genesis alone marks a data root: provisioning without launching leaves
	// no node set, and that root still needs to be removable.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "genesis.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := app.NetworkRemove(context.Background(), app.Deps{}, app.NetworkRemoveIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkRemove: %v", err)
	}
	if out.Stopped != 0 {
		t.Errorf("stopped = %d, want 0 (nothing was launched)", out.Stopped)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("data dir still present: %v", err)
	}
}
