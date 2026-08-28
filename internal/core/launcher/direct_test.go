package launcher_test

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/launcher"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// fakeStore records materialized files without touching disk.
type fakeStore struct {
	written map[string]int
	content map[string][]byte
}

func (s *fakeStore) Exists(context.Context, string) (bool, error) { return false, nil }
func (s *fakeStore) Read(_ context.Context, path string) ([]byte, error) {
	b, ok := s.content[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return b, nil
}

func (s *fakeStore) Write(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	s.written[path]++
	if s.content == nil {
		s.content = map[string][]byte{}
	}
	s.content[path] = content
	return nil
}

// fakeDriver records provision, init, launch, and stop calls; it implements
// driver.Driver and driver.Initializer, standing in for a local or remote
// driver.
type fakeDriver struct {
	provisioned []int
	inited      []int
	launched    []int
	stopped     []int
}

func (d *fakeDriver) Provision(_ context.Context, spec driver.NodeSpec) error {
	d.provisioned = append(d.provisioned, spec.Index)
	return nil
}
func (d *fakeDriver) Stop(_ context.Context, h driver.Handle) error {
	d.stopped = append(d.stopped, h.Index)
	return nil
}
func (d *fakeDriver) InitDatadir(_ context.Context, spec driver.NodeSpec, _ []byte) error {
	d.inited = append(d.inited, spec.Index)
	return nil
}
func (d *fakeDriver) Launch(_ context.Context, spec driver.NodeSpec) (driver.Handle, error) {
	d.launched = append(d.launched, spec.Index)
	return driver.Handle{Index: spec.Index, PID: 1000 + spec.Index}, nil
}

// fakeRemoteDriver is a fakeDriver that also ships files (FileProvisioner), so
// the launcher treats it as a remote host: identity paths root at the remote
// keys dir and Materialize ships the preset identities there.
type fakeRemoteDriver struct {
	fakeDriver
	shipped map[string]int
}

func (d *fakeRemoteDriver) ProvisionFile(_ context.Context, path string, _ []byte, _ fs.FileMode) error {
	if d.shipped == nil {
		d.shipped = map[string]int{}
	}
	d.shipped[path]++
	return nil
}

// launcherTestPlugin is the stablenet-family plugin used across launcher tests.
func launcherTestPlugin() registry.ChainPlugin {
	return registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", ChainID: 1, MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
}

func TestLocalLauncher_ComposesMaterializeInitLaunch(t *testing.T) {
	plugin := launcherTestPlugin()
	presetDir := filepath.Join(repoRoot(t), "keys", "preset")

	placed := []chainsetup.PlacedNode{
		{Req: node.LaunchReq{Role: node.RoleValidator}, Placement: node.Placement{Host: "127.0.0.1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}, DataDir: "/d/node1"}},
		{Req: node.LaunchReq{Role: node.RoleValidator}, Placement: node.Placement{Host: "127.0.0.1", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}, DataDir: "/d/node2"}},
	}
	plan, err := chainsetup.AssemblePlan(plugin, placed, []byte(`{"g":1}`), "/d", []string{"ws"})
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}

	store := &fakeStore{written: map[string]int{}}
	drv := &fakeDriver{}
	l := launcher.Direct{Plugin: plugin, Binary: "go-stablenet", KeysDir: presetDir, Driver: drv, Files: store}

	res, err := l.Launch(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Materialized: genesis + one config per node.
	if store.written[filepath.Join("/d", "genesis.json")] != 1 {
		t.Fatalf("genesis not materialized: %v", store.written)
	}
	if store.written["/d/config_node1.toml"] != 1 || store.written["/d/config_node2.toml"] != 1 {
		t.Fatalf("configs not materialized: %v", store.written)
	}
	// Init routed through the driver's Initializer, then launched.
	sort.Ints(drv.inited)
	sort.Ints(drv.launched)
	if len(drv.inited) != 2 || drv.inited[0] != 1 || drv.inited[1] != 2 {
		t.Fatalf("inited = %v, want [1 2]", drv.inited)
	}
	if len(drv.launched) != 2 {
		t.Fatalf("launched = %v, want 2", drv.launched)
	}
	if len(res.Nodes.Nodes) != 2 || len(res.Procs) != 2 {
		t.Fatalf("result nodes=%d procs=%d, want 2/2", len(res.Nodes.Nodes), len(res.Procs))
	}
	if res.Nodes.Nodes[0].PID == 0 {
		t.Fatal("node PID not set from launch handle")
	}
}

func TestLocalLauncher_ProvisionOnlyDoesNotLaunch(t *testing.T) {
	plugin := launcherTestPlugin()
	presetDir := filepath.Join(repoRoot(t), "keys", "preset")
	placed := []chainsetup.PlacedNode{
		{Req: node.LaunchReq{Role: node.RoleValidator}, Placement: node.Placement{Host: "127.0.0.1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}, DataDir: "/d/node1"}},
		{Req: node.LaunchReq{Role: node.RoleValidator}, Placement: node.Placement{Host: "127.0.0.1", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}, DataDir: "/d/node2"}},
	}
	plan, err := chainsetup.AssemblePlan(plugin, placed, []byte(`{"g":1}`), "/d", []string{"ws"})
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}

	store := &fakeStore{written: map[string]int{}}
	drv := &fakeDriver{}
	l := launcher.Direct{Plugin: plugin, Binary: "go-stablenet", KeysDir: presetDir, Driver: drv, Files: store}

	specs, err := l.Provision(context.Background(), plan)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	// Genesis + configs are materialized...
	if store.written[filepath.Join("/d", "genesis.json")] != 1 {
		t.Fatalf("genesis not materialized: %v", store.written)
	}
	if store.written["/d/config_node1.toml"] != 1 || store.written["/d/config_node2.toml"] != 1 {
		t.Fatalf("configs not materialized: %v", store.written)
	}
	// ...but nothing is initialized or launched.
	if len(drv.inited) != 0 || len(drv.launched) != 0 {
		t.Fatalf("provision-only must not init/launch: inited=%v launched=%v", drv.inited, drv.launched)
	}
	if len(specs) != 2 {
		t.Fatalf("armed specs = %d, want 2", len(specs))
	}
}

func TestLocalLauncher_RemoteShipsIdentities(t *testing.T) {
	plugin := launcherTestPlugin()
	presetDir := filepath.Join(repoRoot(t), "keys", "preset")
	placed := []chainsetup.PlacedNode{
		{Req: node.LaunchReq{Role: node.RoleValidator}, Placement: node.Placement{Host: "127.0.0.1", Ports: node.Endpoints{P2P: 31000, HTTP: 8600}, DataDir: "/d/node1"}},
		{Req: node.LaunchReq{Role: node.RoleValidator}, Placement: node.Placement{Host: "127.0.0.1", Ports: node.Endpoints{P2P: 31010, HTTP: 8610}, DataDir: "/d/node2"}},
	}
	plan, err := chainsetup.AssemblePlan(plugin, placed, []byte(`{"g":1}`), "/d", []string{"ws"})
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}

	store := &fakeStore{written: map[string]int{}}
	drv := &fakeRemoteDriver{}
	l := launcher.Direct{Plugin: plugin, Binary: "go-stablenet", KeysDir: presetDir, Driver: drv, Files: store}

	// Materialize (via Provision) must ship each node's preset identity to the
	// remote keys dir under the data root — the shared password and both nodekeys.
	if _, err := l.Provision(context.Background(), plan); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	keyBase := filepath.Join("/d", "keys")
	for _, want := range []string{
		filepath.Join(keyBase, "password"),
		filepath.Join(keyBase, "node1", "nodekey"),
		filepath.Join(keyBase, "node2", "nodekey"),
	} {
		if drv.shipped[want] != 1 {
			t.Fatalf("identity %q not shipped once: %v", want, drv.shipped)
		}
	}
}

// repoRoot walks up from the test's working directory to the module root (the
// directory holding go.mod), so the shipped preset can be read wherever the
// test runs.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}
