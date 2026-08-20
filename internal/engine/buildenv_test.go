package engine_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/portplan"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
	"github.com/0xmhha/chainbench/internal/engine"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// fakeAllocator returns one placement per request with stepped fake ports.
type fakeAllocator struct {
	gotReqs []place.NodeReq
	err     error
}

func (a *fakeAllocator) Allocate(reqs []place.NodeReq, _ place.Mode, _ place.Capacity) ([]place.NodePlacement, error) {
	a.gotReqs = reqs
	if a.err != nil {
		return nil, a.err
	}
	out := make([]place.NodePlacement, len(reqs))
	for i := range reqs {
		p2p, http := 30301+i, 8501+i
		out[i] = place.NodePlacement{
			Name:  reqs[i].Name,
			Host:  "127.0.0.1",
			Ports: portplan.Ports{P2P: p2p, Etcd: p2p + 1, HTTP: http, WS: http + 1, Auth: http + 2},
		}
	}
	return out, nil
}

// fakeGenesis records the requested validator count and returns fixed bytes.
type fakeGenesis struct {
	bytes         []byte
	gotValidators int
}

func (g *fakeGenesis) Genesis(_ context.Context, _ registry.ChainPlugin, validators int) ([]byte, error) {
	g.gotValidators = validators
	return g.bytes, nil
}

// fakeSupervisor is a real supervisor whose launch/health seams are fakes: it
// synthesizes a node set from the plan and always reports healthy.
func fakeSupervisor() supervisor.Supervisor {
	return supervisor.New(supervisor.Deps{
		Launch: func(_ context.Context, plan driver.Plan) (supervisor.LaunchResult, error) {
			var ns node.NodeSet
			for _, s := range plan.Nodes {
				ns.Nodes = append(ns.Nodes, node.Node{
					Index: s.Index, Role: s.Role, Host: s.Host,
					RPCURL: fmt.Sprintf("http://%s:%d", s.Host, s.Ports.HTTP), Ports: s.Ports,
				})
			}
			return supervisor.LaunchResult{Nodes: ns}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (supervisor.Diagnosis, error) {
			return supervisor.Diagnosis{OK: true}, nil
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

func fourNodeReqs() []place.NodeReq {
	return []place.NodeReq{
		{Name: "node1", Role: node.RoleValidator, Binary: "go-wbft"},
		{Name: "node2", Role: node.RoleValidator, Binary: "go-wbft"},
		{Name: "node3", Role: node.RoleValidator, Binary: "go-wbft"},
		{Name: "node4", Role: node.RoleEndpoint, Binary: "go-wbft"},
	}
}

func TestNewBuildEnv_ComposesAndBringsUp(t *testing.T) {
	alloc := &fakeAllocator{}
	gen := &fakeGenesis{bytes: []byte(`{"genesis":true}`)}
	var gotPlan driver.Plan
	provisionCalled := false

	build := engine.NewBuildEnv(engine.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Allocator:  alloc,
		Genesis:    gen,
		Supervisor: fakeSupervisor(),
		Caps:       []string{"ws"},
		Reqs:       func(testspec.Spec) []place.NodeReq { return fourNodeReqs() },
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
	build := engine.NewBuildEnv(engine.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Allocator:  &fakeAllocator{},
		Genesis:    &fakeGenesis{},
		Supervisor: fakeSupervisor(),
		Reqs:       func(testspec.Spec) []place.NodeReq { return nil },
	})
	if _, _, err := build(context.Background(), buildEnvSession(t), testspec.Spec{}); err == nil {
		t.Fatal("expected error when a spec resolves to no nodes")
	}
}

func TestNewBuildEnv_AllocatorError(t *testing.T) {
	build := engine.NewBuildEnv(engine.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Allocator:  &fakeAllocator{err: errors.New("no capacity")},
		Genesis:    &fakeGenesis{},
		Supervisor: fakeSupervisor(),
		Reqs:       func(testspec.Spec) []place.NodeReq { return fourNodeReqs() },
	})
	if _, _, err := build(context.Background(), buildEnvSession(t), testspec.Spec{}); err == nil {
		t.Fatal("expected allocator error to propagate")
	}
}

func TestNewBuildEnv_ProvisionError(t *testing.T) {
	build := engine.NewBuildEnv(engine.BuildDeps{
		Plugin:     wbftTestPlugin(),
		Allocator:  &fakeAllocator{},
		Genesis:    &fakeGenesis{bytes: []byte("{}")},
		Supervisor: fakeSupervisor(),
		Reqs:       func(testspec.Spec) []place.NodeReq { return fourNodeReqs() },
		Provision:  func(context.Context, driver.Plan) error { return errors.New("disk full") },
	})
	if _, _, err := build(context.Background(), buildEnvSession(t), testspec.Spec{}); err == nil {
		t.Fatal("expected provision error to propagate")
	}
}
