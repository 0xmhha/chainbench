package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin

	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestRunSpec_Live_Stablenet is the walking-skeleton live proof: it brings up a
// real 4-node stablenet network with a real gstable binary, then runs a spec
// through testengine.NewRunSpec — the interpreter with the built-in tx action and
// RPC assertions — and asserts it passes against the running chain.
//
// It is gated on GSTABLE_BIN. CI has no chain binary, so the test skips and the
// suite stays green; a developer with the binary verifies the full DSL-execution
// vertical against a real chain.
func TestRunSpec_Live_Stablenet(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the live run-spec e2e")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}

	plugin, err := registry.Get("stablenet")
	if err != nil {
		t.Fatalf("registry.Get(stablenet): %v", err)
	}
	presetDir := filepath.Join(repoRoot(t), "keys", "preset")
	preset, err := store.LoadPreset(presetDir)
	if err != nil {
		t.Fatalf("load preset: %v", err)
	}

	// A short data root: the geth IPC unix socket path must stay under ~104
	// chars, which t.TempDir() (/var/folders/...) blows past on macOS.
	dataRoot, err := os.MkdirTemp("/tmp", "cbl")
	if err != nil {
		t.Fatalf("mkdir data root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataRoot) })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cfg := config.Resolve(nil, config.Values{"nodes.validators": "4"})
	plan, err := testengine.BuildLocalPlan(cfg, plugin, dataRoot, nil)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	ns, _, err := testengine.LocalSetup{
		Plugin: plugin, Config: cfg, Binary: bin, KeysDir: presetDir,
	}.Launch(ctx, plan)
	t.Cleanup(func() {
		_, errs := driver.StopNodeSet(context.Background(), driver.NewLocalDriver(), ns)
		for _, e := range errs {
			t.Logf("teardown: %v", e)
		}
	})
	if err != nil {
		t.Fatalf("launch stablenet: %v", err)
	}
	if len(ns.Nodes) == 0 {
		t.Fatal("no nodes launched")
	}

	// Wait for block production to warm up (wbft takes ~10-40s here).
	if err := waitForHead(ctx, ns.Nodes[0].RPCURL, 1, 90*time.Second); err != nil {
		t.Fatalf("chain did not produce blocks: %v", err)
	}

	// Build the session environment over the running nodes and run the spec.
	sess, err := session.New(t.TempDir(), "live", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("live00000000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	env.PopulateNodeTable(ns)

	deps := testspec.Deps{
		RPC:     func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions: testspec.NewRegistry(true),
	}
	run := testengine.NewRunSpec(deps)

	spec := liveSpec(t, plugin.Manifest().ChainID, preset)
	rec := sess.Test(1, spec.ID)
	status, err := run(ctx, spec, env, rec)
	if err != nil {
		t.Fatalf("run spec: %v", err)
	}
	if status != session.StatusPass {
		t.Fatalf("live spec status = %q, want pass", status)
	}
}

// liveSpec builds a smoke spec: send one node-signed tx, then assert the chain
// id and that the head has advanced.
func liveSpec(t *testing.T, chainID int64, preset keyring.Preset) testspec.Spec {
	t.Helper()
	from := preset.Network.Validators[0]
	to := from
	if len(preset.Network.Validators) > 1 {
		to = preset.Network.Validators[1]
	}
	raw, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "live-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"steps": []map[string]any{
			{"sendTx": map[string]any{"from": from, "to": to, "value": "1", "timeout": "40s", "pollInterval": "1s"}},
		},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": chainID},
			{"assert": "blockNumber", "expected": 1},
		},
	})
	spec, err := testspec.Parse(raw)
	if err != nil {
		t.Fatalf("parse live spec: %v", err)
	}
	return spec
}

// waitForHead polls until the head reaches at least target or the deadline.
func waitForHead(ctx context.Context, rpcURL string, target uint64, timeout time.Duration) error {
	c := rpc.Dial(rpcURL)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		if h, err := c.BlockNumber(ctx); err == nil && h >= target {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return context.DeadlineExceeded
		case <-tick.C:
		}
	}
}
