package upgrade_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// fakeDriver records the specs it is asked to provision and launch.
type fakeDriver struct {
	launched []driver.NodeSpec
	nextPID  int
}

func (f *fakeDriver) Provision(context.Context, driver.NodeSpec) error { return nil }
func (f *fakeDriver) Launch(_ context.Context, s driver.NodeSpec) (driver.Handle, error) {
	f.nextPID++
	f.launched = append(f.launched, s)
	return driver.Handle{Index: s.Index, PID: 1000 + f.nextPID}, nil
}
func (f *fakeDriver) Stop(context.Context, driver.Handle) error { return nil }

func launchOpts(t *testing.T) upgrade.LaunchOptions {
	t.Helper()
	from, _ := registry.Get("wemix")
	to, _ := registry.Get("wbft")
	return upgrade.LaunchOptions{
		DataRoot:   t.TempDir(),
		FromBinary: "gwemix", ToBinary: "gwbft",
		FromFamily: from.Family(), ToFamily: to.Family(),
		InitFn: func(_ context.Context, binary, dataDir, genesisPath string) error { return nil },
	}
}

func TestLaunch_MixedBinariesConcurrent(t *testing.T) {
	from, to := plugins(t)
	plan, err := upgrade.BuildPlan(from, to, goodInputs())
	if err != nil {
		t.Fatal(err)
	}
	d := &fakeDriver{}
	opts := launchOpts(t)
	ns, err := upgrade.Launch(context.Background(), d, plan, opts)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if len(ns.Nodes) != 5 || len(d.launched) != 5 {
		t.Fatalf("want 5 launched nodes, got set=%d driver=%d", len(ns.Nodes), len(d.launched))
	}
	// node0 is the producer on the from-binary; the rest are validators on the to-binary.
	if d.launched[0].Binary != "gwemix" {
		t.Errorf("producer should run gwemix, got %s", d.launched[0].Binary)
	}
	for _, s := range d.launched[1:] {
		if s.Binary != "gwbft" {
			t.Errorf("validator node%d should run gwbft, got %s", s.Index, s.Binary)
		}
	}
	// every launched node carries the uniform network id.
	for _, s := range d.launched {
		if !strings.Contains(strings.Join(s.Args, " "), "--networkid 8285") {
			t.Errorf("node%d missing uniform networkid: %v", s.Index, s.Args)
		}
	}
	// the returned set records a pid per node.
	for _, n := range ns.Nodes {
		if n.PID == 0 {
			t.Errorf("node%d has no pid", n.Index)
		}
	}
}

func TestBuildNodeSpecs_Rejects(t *testing.T) {
	from, to := plugins(t)
	plan, _ := upgrade.BuildPlan(from, to, goodInputs())
	if _, err := upgrade.BuildNodeSpecs(plan, upgrade.LaunchOptions{ToBinary: "gwbft"}); err == nil {
		t.Error("missing from binary should error")
	}
	if _, err := upgrade.BuildNodeSpecs(plan, upgrade.LaunchOptions{FromBinary: "a", ToBinary: "b"}); err == nil {
		t.Error("missing families should error")
	}
}
