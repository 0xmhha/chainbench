package engine_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
	"github.com/0xmhha/chainbench/internal/engine"
	"github.com/0xmhha/chainbench/internal/testspec"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin
)

// TestBuildEnv_Live_Stablenet proves the bring-up half of the walking skeleton:
// engine.NewBuildEnv, wired with a real allocator, PresetGenesisSource, the
// LocalLauncher, and a block-advance health gate, brings a real 4-node
// stablenet network up on the allocator-assigned ports, and the teardown stops
// it cleanly.
//
// Gated on GSTABLE_BIN. CI has no chain binary, so it skips and the suite stays
// green.
func TestBuildEnv_Live_Stablenet(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the live build-env e2e")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}

	plugin, err := registry.Get("stablenet")
	if err != nil {
		t.Fatalf("registry.Get(stablenet): %v", err)
	}
	presetDir := filepath.Join(repoRoot(t), "keys", "preset")

	// A short session root keeps each node's IPC unix socket path under the
	// ~104-char limit (env datadirs nest under this root).
	sessRoot, err := os.MkdirTemp("/tmp", "cbs")
	if err != nil {
		t.Fatalf("mkdir session root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sessRoot) })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	sup := supervisor.New(supervisor.Deps{
		Launch: engine.LocalLauncher{Plugin: plugin, Binary: bin, KeysDir: presetDir}.Launch,
		HealthGate: func(c context.Context, ns node.NodeSet) (supervisor.Diagnosis, error) {
			if len(ns.Nodes) == 0 {
				return supervisor.Diagnosis{Mode: supervisor.RPCUnready}, fmt.Errorf("no nodes launched")
			}
			if err := waitForHead(c, ns.Nodes[0].RPCURL, 1, 90*time.Second); err != nil {
				return supervisor.Diagnosis{Mode: supervisor.RPCUnready, Detail: err.Error()}, err
			}
			return supervisor.Diagnosis{OK: true}, nil
		},
		Procman: procman.New(),
	})

	build := engine.NewBuildEnv(engine.BuildDeps{
		Plugin:     plugin,
		Allocator:  place.New(place.Config{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10}),
		Genesis:    engine.PresetGenesisSource{KeysDir: presetDir},
		Supervisor: sup,
		Mode:       place.LocalStepped,
		Capacity:   place.Capacity{MinValidators: 1, PortBandSize: 100},
		Caps:       []string{"ws"},
		Reqs: func(testspec.Spec) []place.NodeReq {
			reqs := make([]place.NodeReq, 4)
			for i := range reqs {
				reqs[i] = place.NodeReq{Name: fmt.Sprintf("node%d", i+1), Role: node.RoleValidator, Binary: "go-stablenet"}
			}
			return reqs
		},
	})

	sess, err := session.New(sessRoot, "live", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("live00000000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}

	ns, teardown, err := build(ctx, env, testspec.Spec{})
	if teardown != nil {
		t.Cleanup(func() { _ = teardown(context.Background()) })
	}
	if err != nil {
		t.Fatalf("build env: %v", err)
	}
	if len(ns.Nodes) != 4 {
		t.Fatalf("node set = %d, want 4", len(ns.Nodes))
	}
	// The gate already waited for the head; confirm the RPC still answers.
	if err := waitForHead(ctx, ns.Nodes[0].RPCURL, 1, 10*time.Second); err != nil {
		t.Fatalf("head not advancing after bring-up: %v", err)
	}
	t.Logf("BuildEnv brought up %d nodes on allocator ports (node1 %s)", len(ns.Nodes), ns.Nodes[0].RPCURL)
}
