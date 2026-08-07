package engine_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/portplan"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/engine"
)

// fakeSink records materialized files without touching disk.
type fakeSink struct{ written map[string]int }

func (s *fakeSink) Exists(context.Context, string) (bool, error) { return false, nil }
func (s *fakeSink) Write(_ context.Context, path string, _ []byte, _ fs.FileMode) error {
	s.written[path]++
	return nil
}

// fakeDriver records init and launch calls; it implements driver.Driver and
// driver.Initializer, standing in for a local or remote driver.
type fakeDriver struct {
	inited   []int
	launched []int
}

func (d *fakeDriver) Provision(context.Context, driver.NodeSpec) error { return nil }
func (d *fakeDriver) Stop(context.Context, driver.Handle) error        { return nil }
func (d *fakeDriver) InitDatadir(_ context.Context, spec driver.NodeSpec, _ []byte) error {
	d.inited = append(d.inited, spec.Index)
	return nil
}
func (d *fakeDriver) Launch(_ context.Context, spec driver.NodeSpec) (driver.Handle, error) {
	d.launched = append(d.launched, spec.Index)
	return driver.Handle{Index: spec.Index, PID: 1000 + spec.Index}, nil
}

func TestLocalLauncher_ComposesMaterializeInitLaunch(t *testing.T) {
	plugin := registry.StaticPlugin{
		M: registry.Manifest{
			ID: "stablenet", Binary: "go-stablenet", ChainID: 1, MinerRecommit: "duration",
			Consensus: registry.ConsensusSpec{RPCNamespace: "istanbul"},
		},
		Fam: wbftfam.New(),
	}
	presetDir := filepath.Join(repoRoot(t), "keys", "preset")

	placed := []engine.PlacedNode{
		{Req: place.NodeReq{Role: node.RoleValidator}, Placement: place.NodePlacement{Host: "127.0.0.1", Ports: portplan.Ports{P2P: 31000, HTTP: 8600}, DataPath: "/d/node1"}},
		{Req: place.NodeReq{Role: node.RoleValidator}, Placement: place.NodePlacement{Host: "127.0.0.1", Ports: portplan.Ports{P2P: 31010, HTTP: 8610}, DataPath: "/d/node2"}},
	}
	plan, err := engine.AssemblePlan(plugin, placed, []byte(`{"g":1}`), "/d", []string{"ws"})
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}

	sink := &fakeSink{written: map[string]int{}}
	drv := &fakeDriver{}
	l := engine.LocalLauncher{Plugin: plugin, Binary: "go-stablenet", KeysDir: presetDir, Driver: drv, Sink: sink}

	res, err := l.Launch(context.Background(), plan)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Materialized: genesis + one config per node.
	if sink.written[filepath.Join("/d", "genesis.json")] != 1 {
		t.Fatalf("genesis not materialized: %v", sink.written)
	}
	if sink.written["/d/config_node1.toml"] != 1 || sink.written["/d/config_node2.toml"] != 1 {
		t.Fatalf("configs not materialized: %v", sink.written)
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
