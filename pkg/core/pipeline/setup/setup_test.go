package setup_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// fakeDriver records provisions/launches without touching the OS.
type fakeDriver struct {
	mu          sync.Mutex
	provisioned []int
	launched    []int
}

func (f *fakeDriver) Provision(_ context.Context, s driver.NodeSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.provisioned = append(f.provisioned, s.Index)
	return nil
}

func (f *fakeDriver) Launch(_ context.Context, s driver.NodeSpec) (driver.Handle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.launched = append(f.launched, s.Index)
	return driver.Handle{Index: s.Index, PID: 1000 + s.Index}, nil
}

func (f *fakeDriver) Stop(context.Context, driver.Handle) error { return nil }

func TestBuildPlan_RolesAndPorts(t *testing.T) {
	p, err := registry.Get("stablenet")
	if err != nil {
		t.Fatal(err)
	}
	// Defaults: 4 validators + 1 endpoint = 5 nodes.
	plan, err := setup.BuildPlan(config.Defaults(), p, "/tmp/data")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Nodes) != 5 {
		t.Fatalf("nodes: got %d, want 5", len(plan.Nodes))
	}
	// Roles: first 4 validators, last endpoint.
	for i, n := range plan.Nodes {
		wantRole := node.RoleValidator
		if i >= 4 {
			wantRole = node.RoleEndpoint
		}
		if n.Role != wantRole {
			t.Errorf("node%d role: got %s, want %s", n.Index, n.Role, wantRole)
		}
	}
	// Ports offset per node: node1 http=8501, node5 http=8505.
	if plan.Nodes[0].Ports.HTTP != 8501 || plan.Nodes[4].Ports.HTTP != 8505 {
		t.Errorf("http ports: node1=%d node5=%d", plan.Nodes[0].Ports.HTTP, plan.Nodes[4].Ports.HTTP)
	}
	// Binary from manifest, validator has --mine, endpoint does not.
	if plan.Nodes[0].Binary != "gstable" {
		t.Errorf("binary: %s", plan.Nodes[0].Binary)
	}
	if !hasFlag(plan.Nodes[0].Args, "--mine") {
		t.Error("validator should have --mine")
	}
	if hasFlag(plan.Nodes[4].Args, "--mine") {
		t.Error("endpoint should not have --mine")
	}
}

func TestRun_ProvisionsLaunchesWritesGenesisEmitsEvents(t *testing.T) {
	p, _ := registry.Get("wbft")
	dir := t.TempDir()
	plan, err := setup.BuildPlan(config.Resolve(config.Values{"nodes.validators": "2", "nodes.endpoints": "0"}, nil), p, dir)
	if err != nil {
		t.Fatal(err)
	}
	plan.Genesis = []byte(`{"config":{"chainId":1}}`)

	bus := obs.NewBus()
	sub := bus.Subscribe()
	var events []obs.Event
	done := make(chan struct{})
	go func() {
		for e := range sub {
			events = append(events, e)
		}
		close(done)
	}()

	fd := &fakeDriver{}
	ns, err := setup.Run(context.Background(), plan, fd, bus)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	bus.Close()
	<-done

	// NodeSet shape.
	if ns.Chain != "wbft" || len(ns.Nodes) != 2 {
		t.Fatalf("nodeset: chain=%s nodes=%d", ns.Chain, len(ns.Nodes))
	}
	if ns.Nodes[0].RPCURL != "http://127.0.0.1:8501" {
		t.Errorf("rpc url: %s", ns.Nodes[0].RPCURL)
	}
	// Driver saw both nodes provisioned then launched.
	if len(fd.provisioned) != 2 || len(fd.launched) != 2 {
		t.Errorf("driver calls: provisioned=%v launched=%v", fd.provisioned, fd.launched)
	}
	// Genesis written to disk.
	if b, err := os.ReadFile(plan.GenesisPath); err != nil || !strings.Contains(string(b), "chainId") {
		t.Errorf("genesis not written: %v", err)
	}
	// Events: genesis + 2 launched + complete.
	var launched, complete int
	for _, e := range events {
		if e.Message == "node launched" {
			launched++
		}
		if e.Message == "setup complete" {
			complete++
		}
	}
	if launched != 2 || complete != 1 {
		t.Errorf("events: launched=%d complete=%d (%d total)", launched, complete, len(events))
	}
}

func TestBuildPlan_ZeroNodes(t *testing.T) {
	p, _ := registry.Get("stablenet")
	_, err := setup.BuildPlan(config.Values{"nodes.validators": "0", "nodes.endpoints": "0"}, p, "/tmp")
	if err == nil {
		t.Error("expected error for zero nodes")
	}
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
