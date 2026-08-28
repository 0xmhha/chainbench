package testengine_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/genesis"

	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testengine"
	"github.com/0xmhha/chainbench/internal/testspec"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin
)

// TestBuildEnv_Live_Stablenet proves the bring-up half of the walking skeleton:
// testengine.NewBuildEnv, wired with a real allocator, PresetGenesisSource, the
// launcher.Direct, and a block-advance health gate, brings a real 4-node
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

	sup := launcher.New(launcher.Deps{
		Launch: launcher.Direct{Plugin: plugin, Binary: bin, KeysDir: presetDir}.Launch,
		HealthGate: func(c context.Context, ns node.NodeSet) (launcher.Diagnosis, error) {
			if len(ns.Nodes) == 0 {
				return launcher.Diagnosis{Mode: launcher.RPCUnready}, fmt.Errorf("no nodes launched")
			}
			if err := waitForHead(c, ns.Nodes[0].RPCURL, 1, 90*time.Second); err != nil {
				return launcher.Diagnosis{Mode: launcher.RPCUnready, Detail: err.Error()}, err
			}
			return launcher.Diagnosis{OK: true}, nil
		},
		Procman: process.New(),
	})

	build := testengine.NewBuildEnv(testengine.BuildDeps{
		Plugin: plugin,
		Pool: resource.Pool{
			Hosts: []resource.Host{{Name: "local", Addr: "127.0.0.1"}},
			Slots: 8,
			Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
		},
		Genesis:    genesis.PresetSource{KeysDir: presetDir},
		Supervisor: sup,
		Caps:       []string{"ws"},
		Reqs: func(testspec.Spec) []node.LaunchReq {
			reqs := make([]node.LaunchReq, 4)
			for i := range reqs {
				reqs[i] = node.LaunchReq{Role: node.RoleValidator, Binary: "go-stablenet"}
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
