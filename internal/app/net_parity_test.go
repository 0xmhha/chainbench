package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/netcompose"
)

// The step stack has to be able to compose every network the setup stack can,
// or it cannot replace it. These cover the customizations setup supported and
// the steps did not: per-role sync mode, genesis config overrides, and the
// genesis overlay with its capability claims.

// composed runs new + allocate + keys on a fresh workspace and returns it.
func composed(t *testing.T, alloc app.NetAllocateIn) (dir string, d app.Deps) {
	t.Helper()
	dir = t.TempDir()
	d = app.Deps{Clock: fixedClock}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	alloc.DataDir = dir
	if _, err := app.NetAllocate(ctx, d, alloc); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if _, err := app.NetKeys(ctx, d, app.NetKeysIn{DataDir: dir}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	return dir, d
}

// stateOf reads the persisted composition state.
func stateOf(t *testing.T, dir string, d app.Deps) netcompose.State {
	t.Helper()
	out, err := app.NetStatus(context.Background(), d, app.NetStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	return out.State
}

func TestNetAllocate_EndpointSyncModeReachesTheConfig(t *testing.T) {
	// Every node rendered "full" before this: a snap-sync re-sync test composed
	// through the steps was silently running full sync instead.
	dir, d := composed(t, app.NetAllocateIn{Validators: 2, Endpoints: 1, EndpointSyncMode: "snap"})
	if _, err := app.NetGenesis(context.Background(), d, app.NetGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if _, err := app.NetConfig(context.Background(), d, app.NetConfigIn{DataDir: dir}); err != nil {
		t.Fatalf("config: %v", err)
	}

	for _, n := range stateOf(t, dir, d).Nodes {
		want := "full"
		if n.Role == "endpoint" {
			want = "snap"
		}
		if n.SyncMode != want {
			t.Errorf("node%d (%s) sync mode = %q, want %q", n.Index, n.Role, n.SyncMode, want)
		}
		body, err := os.ReadFile(n.ConfigPath)
		if err != nil {
			t.Fatalf("read node%d config: %v", n.Index, err)
		}
		if !strings.Contains(string(body), `SyncMode = "`+want+`"`) {
			t.Errorf("node%d config does not render SyncMode %q", n.Index, want)
		}
	}
}

func TestNetAllocate_ValidatorsIgnoreTheEndpointSyncMode(t *testing.T) {
	// A sealing node must hold full state, so the knob must not reach it even
	// when the caller sets it.
	dir, d := composed(t, app.NetAllocateIn{Validators: 2, EndpointSyncMode: "snap"})
	for _, n := range stateOf(t, dir, d).Nodes {
		if n.SyncMode != "full" {
			t.Errorf("validator node%d sync mode = %q, want full", n.Index, n.SyncMode)
		}
	}
}

func TestNetGenesis_ConfigOverrideDelaysTheFork(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 2})

	out, err := app.NetGenesis(context.Background(), d, app.NetGenesisIn{
		DataDir: dir, Set: []string{"bohoBlock=10"},
	})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if !strings.Contains(out.Detail, "override") {
		t.Errorf("detail should mention the overrides, got %q", out.Detail)
	}
	cfg := genesisConfig(t, filepath.Join(dir, "genesis.json"))
	if got := cfg["bohoBlock"]; got != float64(10) {
		t.Errorf("bohoBlock = %v, want 10", got)
	}
	// A fork moved off genesis is advertised so the fork-transition cases run.
	if !hasCapability(stateOf(t, dir, d).Capabilities, "delayed-boho") {
		t.Errorf("capabilities = %v, want delayed-boho", stateOf(t, dir, d).Capabilities)
	}
}

func TestNetGenesis_ForkAtGenesisIsNotAdvertisedAsDelayed(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 2})

	if _, err := app.NetGenesis(context.Background(), d, app.NetGenesisIn{
		DataDir: dir, Set: []string{"bohoBlock=0"},
	}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	caps := stateOf(t, dir, d).Capabilities
	if hasCapability(caps, "delayed-boho") {
		t.Errorf("block 0 is genesis, not a delay: %v", caps)
	}
	// The baseline capabilities are still there.
	if !hasCapability(caps, "ws") {
		t.Errorf("capabilities = %v, want ws", caps)
	}
}

func TestNetGenesis_OverlayMergesAndDeclaresCapabilities(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 2})
	overlay := filepath.Join(t.TempDir(), "overlay.json")
	if err := os.WriteFile(overlay, []byte(`{"capabilities":["account-extra"],"genesis":{"config":{"chainId":4242}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := app.NetGenesis(context.Background(), d, app.NetGenesisIn{DataDir: dir, OverlayPath: overlay})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if !strings.Contains(out.Detail, "overlay") {
		t.Errorf("detail should mention the overlay, got %q", out.Detail)
	}
	if got := genesisConfig(t, filepath.Join(dir, "genesis.json"))["chainId"]; got != float64(4242) {
		t.Errorf("overlay did not merge: chainId = %v", got)
	}
	if !hasCapability(stateOf(t, dir, d).Capabilities, "account-extra") {
		t.Errorf("overlay capability not advertised: %v", stateOf(t, dir, d).Capabilities)
	}
}

func TestNetGenesis_MalformedInputsAreRejected(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 2})
	ctx := context.Background()

	if _, err := app.NetGenesis(ctx, d, app.NetGenesisIn{DataDir: dir, Set: []string{"novalue"}}); err == nil {
		t.Error("want an error for an override without a value")
	}
	if _, err := app.NetGenesis(ctx, d, app.NetGenesisIn{DataDir: dir, OverlayPath: "/nonexistent/overlay.json"}); err == nil {
		t.Error("want an error for a missing overlay file")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := app.NetGenesis(ctx, d, app.NetGenesisIn{DataDir: dir, OverlayPath: bad}); err == nil {
		t.Error("want an error for an unparseable overlay")
	}
}

// genesisConfig reads the `config` object out of a written genesis.
func genesisConfig(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read genesis: %v", err)
	}
	var gen struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(raw, &gen); err != nil {
		t.Fatalf("parse genesis: %v", err)
	}
	return gen.Config
}
