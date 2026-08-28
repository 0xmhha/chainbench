package chainsetup_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// testPool is the resource the build allocates from: one host with room for a
// development-sized network. Allocation is no longer injectable — resource.Assign
// is deterministic, so a fake would only restate it less accurately.
func testPool() resource.Pool {
	return resource.Pool{
		Hosts: []resource.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots: 8,
		Ports: resource.Bands{P2P: resource.Band{Base: 30301, Step: 10}, RPC: resource.Band{Base: 8501, Step: 10}},
	}
}

// fakeGenesis records the requested validator count and returns fixed bytes.
type fakeGenesis struct {
	bytes         []byte
	extra         map[string][]byte
	gotValidators int
}

func (g *fakeGenesis) Genesis(_ context.Context, _ registry.ChainPlugin, req genesis.Request) (genesis.Artifacts, error) {
	g.gotValidators = req.Validators
	return genesis.Artifacts{Genesis: g.bytes, Extra: g.extra}, nil
}

// fakeSupervisor is a real supervisor whose launch and health hooks are fakes: it
// synthesizes a node set from the plan and always reports healthy.
func fakeSupervisor() launcher.Launcher {
	return launcher.New(launcher.Deps{
		Launch: func(_ context.Context, plan driver.Plan, _ []int) (launcher.Result, error) {
			var ns node.NodeSet
			for _, s := range plan.Nodes {
				ns.Nodes = append(ns.Nodes, node.Node{
					Index: s.Index, Role: s.Role, Host: s.Host,
					RPCURL: fmt.Sprintf("http://%s:%d", s.Host, s.Ports.HTTP), Ports: s.Ports,
				})
			}
			return launcher.Result{Nodes: ns}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (launcher.Diagnosis, error) {
			return launcher.Diagnosis{OK: true}, nil
		},
		Sleep: func(time.Duration) {},
	})
}

func buildEnvSession(t *testing.T) session.Environment {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("aaaaaaaaaaaa0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	return env
}

func fourNodeReqs() []node.LaunchReq {
	return []node.LaunchReq{
		{Role: node.RoleValidator, Binary: "go-wbft"},
		{Role: node.RoleValidator, Binary: "go-wbft"},
		{Role: node.RoleValidator, Binary: "go-wbft"},
		{Role: node.RoleEndpoint, Binary: "go-wbft"},
	}
}

func TestNewBuildEnv_ComposesAndBringsUp(t *testing.T) {
	gen := &fakeGenesis{bytes: []byte(`{"genesis":true}`)}
	var gotPlan driver.Plan
	provisionCalled := false

	build := chainsetup.NewBuildEnv(chainsetup.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Pool:       testPool(),
		Genesis:    gen,
		Supervisor: fakeSupervisor(),
		Caps:       []string{"ws"},
		Reqs:       func(testspec.Spec) []node.LaunchReq { return fourNodeReqs() },
		Provision: func(_ context.Context, plan driver.Plan) error {
			provisionCalled = true
			gotPlan = plan
			return nil
		},
	})

	ns, teardown, err := build(context.Background(), buildEnvSession(t), testspec.Spec{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(ns.Nodes) != 4 {
		t.Fatalf("node set = %d, want 4", len(ns.Nodes))
	}
	if gen.gotValidators != 3 {
		t.Fatalf("genesis validators = %d, want 3", gen.gotValidators)
	}
	if !provisionCalled {
		t.Fatal("provision was not called")
	}
	if len(gotPlan.Nodes) != 4 || string(gotPlan.Genesis) != `{"genesis":true}` {
		t.Fatalf("plan not threaded: nodes=%d genesis=%q", len(gotPlan.Nodes), gotPlan.Genesis)
	}
	if gotPlan.Nodes[0].Ports.HTTP != 8501 {
		t.Fatalf("plan node1 http = %d, want allocator port 8501", gotPlan.Nodes[0].Ports.HTTP)
	}
	if teardown == nil {
		t.Fatal("teardown is nil")
	}
	if err := teardown(context.Background()); err != nil {
		t.Fatalf("teardown: %v", err)
	}
}

func TestNewBuildEnv_NoNodes(t *testing.T) {
	build := chainsetup.NewBuildEnv(chainsetup.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Pool:       testPool(),
		Genesis:    &fakeGenesis{},
		Supervisor: fakeSupervisor(),
		Reqs:       func(testspec.Spec) []node.LaunchReq { return nil },
	})
	if _, _, err := build(context.Background(), buildEnvSession(t), testspec.Spec{}); err == nil {
		t.Fatal("expected error when a spec resolves to no nodes")
	}
}

// TestNewBuildEnv_PoolTooSmall: a network that does not fit must fail before
// anything is provisioned, naming the shortfall — the alternative is two nodes
// handed the same port and a bind failure much later.
func TestNewBuildEnv_PoolTooSmall(t *testing.T) {
	small := testPool()
	small.Slots = 2
	build := chainsetup.NewBuildEnv(chainsetup.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Pool:       small,
		Genesis:    &fakeGenesis{},
		Supervisor: fakeSupervisor(),
		Reqs:       func(testspec.Spec) []node.LaunchReq { return fourNodeReqs() },
	})
	_, _, err := build(context.Background(), buildEnvSession(t), testspec.Spec{})
	if err == nil {
		t.Fatal("expected the over-capacity assignment to fail")
	}
	if !strings.Contains(err.Error(), "short") {
		t.Fatalf("error %q should name the shortfall", err)
	}
}

func TestNewBuildEnv_ProvisionError(t *testing.T) {
	build := chainsetup.NewBuildEnv(chainsetup.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Pool:       testPool(),
		Genesis:    &fakeGenesis{bytes: []byte("{}")},
		Supervisor: fakeSupervisor(),
		Reqs:       func(testspec.Spec) []node.LaunchReq { return fourNodeReqs() },
		Provision:  func(context.Context, driver.Plan) error { return errors.New("disk full") },
	})
	if _, _, err := build(context.Background(), buildEnvSession(t), testspec.Spec{}); err == nil {
		t.Fatal("expected provision error to propagate")
	}
}
