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

func TestNetAllocate_TopologyDrivesRolesAndSyncModes(t *testing.T) {
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, err := filepath.Abs(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}

	topo := filepath.Join(t.TempDir(), "topology.yaml")
	if err := os.WriteFile(topo, []byte(`chain: stablenet
nodes:
  - index: 1
    role: bp
    bootnode: true
  - index: 2
    role: en
    sync_mode: archive
  - index: 3
    role: bp
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := app.NetAllocate(ctx, d, app.NetAllocateIn{DataDir: dir, TopologyPath: topo})
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if !strings.Contains(out.Detail, "topology") {
		t.Errorf("detail should say the layout came from a topology, got %q", out.Detail)
	}

	st := stateOf(t, dir, d)
	want := []struct{ role, sync string }{
		{"validator", "full"},
		{"endpoint", "archive"},
		{"validator", "full"},
	}
	if len(st.Nodes) != len(want) {
		t.Fatalf("got %d nodes, want %d", len(st.Nodes), len(want))
	}
	for i, w := range want {
		if st.Nodes[i].Role != w.role || st.Nodes[i].SyncMode != w.sync {
			t.Errorf("node%d = %s/%s, want %s/%s", i+1, st.Nodes[i].Role, st.Nodes[i].SyncMode, w.role, w.sync)
		}
	}
	// The validator count comes from the resolved layout, not a requested
	// number, so the genesis step sizes its validator set correctly.
	if st.Validators != 2 {
		t.Errorf("validators = %d, want 2 from the topology", st.Validators)
	}
	if st.Bootnode != 1 {
		t.Errorf("bootnode = %d, want 1", st.Bootnode)
	}
}

func TestNetAllocate_TopologyCannotMakeAValidatorStateless(t *testing.T) {
	// A sealing node must hold full state; a topology asking otherwise is
	// overridden rather than silently producing a network that cannot seal.
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, _ := filepath.Abs(presetDir)
	ctx := context.Background()
	if _, err := app.NetNew(ctx, d, app.NetNewIn{DataDir: dir, Chain: "stablenet", KeysDir: keysAbs}); err != nil {
		t.Fatalf("new: %v", err)
	}
	topo := filepath.Join(t.TempDir(), "topology.yaml")
	if err := os.WriteFile(topo, []byte(`chain: stablenet
nodes:
  - index: 1
    role: bp
    sync_mode: snap
  - index: 2
    role: bp
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{DataDir: dir, TopologyPath: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	for _, n := range stateOf(t, dir, d).Nodes {
		if n.SyncMode != "full" {
			t.Errorf("validator node%d sync mode = %q, want full", n.Index, n.SyncMode)
		}
	}
}

func TestNetAllocate_MissingTopologyFileIsAnError(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 1})
	if _, err := app.NetAllocate(context.Background(), d, app.NetAllocateIn{
		DataDir: dir, TopologyPath: "/nonexistent/topology.yaml",
	}); err == nil {
		t.Error("want an error for a missing topology file")
	}
}

func TestNetNew_ExternalManifestChainSurvivesLaterSteps(t *testing.T) {
	// A project-supplied chain has to resolve on every later step, not just at
	// `new`: the workspace records the manifest, not only the id.
	dir := t.TempDir()
	d := app.Deps{Clock: fixedClock}
	keysAbs, _ := filepath.Abs(presetDir)
	ctx := context.Background()

	manifestDir := t.TempDir()
	manifest := filepath.Join(manifestDir, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{
		"id": "foonet", "binary": "gfoo", "chain_id": 9999, "network_id": 9999,
		"miner_recommit": "duration", "bootstrap": {"type": "static"},
		"consensus_family": "wbft", "protocol": "stablenet",
		"genesis": {"template": "foonet-genesis"},
		"consensus": {"rpc_namespace": "istanbul", "validators_method": "istanbul_getValidators"},
		"probe": {"method": "istanbul_getValidators"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(manifestDir, "genesis.json")
	if err := os.WriteFile(template, []byte(`{"config":{"chainId":9999},"extraData":"0x0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := app.NetNew(ctx, d, app.NetNewIn{
		DataDir: dir, KeysDir: keysAbs, ManifestPath: manifest, TemplatePath: template,
	}); err != nil {
		t.Fatalf("new: %v", err)
	}
	// The manifest's own id is recorded, so status reports a real chain.
	if got := stateOf(t, dir, d).Chain; got != "foonet" {
		t.Errorf("chain = %q, want foonet from the manifest", got)
	}

	// A later step must resolve the same plugin — registry.Get("foonet") would
	// fail, since it is not an embedded chain.
	if _, err := app.NetAllocate(ctx, d, app.NetAllocateIn{DataDir: dir, Validators: 2}); err != nil {
		t.Fatalf("allocate on an external chain: %v", err)
	}
	if _, err := app.NetGenesis(ctx, d, app.NetGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("genesis on an external chain: %v", err)
	}
	if got := genesisConfig(t, filepath.Join(dir, "genesis.json"))["chainId"]; got != float64(9999) {
		t.Errorf("genesis chainId = %v, want the manifest's 9999", got)
	}
}

func TestNetNew_NeedsAChainOrAManifest(t *testing.T) {
	if _, err := app.NetNew(context.Background(), app.Deps{Clock: fixedClock},
		app.NetNewIn{DataDir: t.TempDir()}); err == nil {
		t.Error("want an error with neither a chain nor a manifest")
	}
}

func TestNetworkStatus_ReadsAComposedWorkspace(t *testing.T) {
	// Every consumer downstream of a bring-up speaks NodeSet, so a composed
	// network has to be readable through the same call a setup one is.
	dir, d := composed(t, app.NetAllocateIn{Validators: 2, Endpoints: 1})
	if _, err := app.NetGenesis(context.Background(), d, app.NetGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("genesis: %v", err)
	}

	out, err := app.NetworkStatus(context.Background(), d, app.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStatus: %v", err)
	}
	if !out.Composed {
		t.Error("a workspace directory should report as composed")
	}
	if out.Nodes.Chain != "stablenet" || len(out.Nodes.Nodes) != 3 {
		t.Fatalf("node set = %+v", out.Nodes)
	}
	first := out.Nodes.Nodes[0]
	if first.RPCURL == "" || !strings.HasPrefix(first.RPCURL, "http://127.0.0.1:") {
		t.Errorf("node1 rpc url = %q", first.RPCURL)
	}
	if first.Ports.HTTP == 0 || first.Ports.P2P == 0 {
		t.Errorf("node1 ports not carried over: %+v", first.Ports)
	}
	// A network that has never been started reports no PIDs, the same
	// convention as an attached node chainbench did not launch.
	if first.PID != 0 {
		t.Errorf("node1 PID = %d, want 0 before start", first.PID)
	}
	if !hasCapability(out.Nodes.Capabilities, "ws") {
		t.Errorf("capabilities did not survive the bridge: %v", out.Nodes.Capabilities)
	}
}

func TestNetworkStatus_StillReadsASetupDataRoot(t *testing.T) {
	// The bridge must not shadow the stack it is meant to coexist with.
	dir, _, deps := launchedNetwork(t)
	out, err := app.NetworkStatus(context.Background(), deps, app.NetworkStatusIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStatus: %v", err)
	}
	if out.Composed {
		t.Error("a setup data root must not report as composed")
	}
	if len(out.Nodes.Nodes) != 2 {
		t.Errorf("node set = %+v", out.Nodes)
	}
}

func TestNetworkStop_OnAComposedWorkspaceWithNothingRunning(t *testing.T) {
	dir, d := composed(t, app.NetAllocateIn{Validators: 2})

	out, err := app.NetworkStop(context.Background(), d, app.NetworkStopIn{DataDir: dir})
	if err != nil {
		t.Fatalf("NetworkStop: %v", err)
	}
	if out.Stopped != 0 {
		t.Errorf("stopped = %d, want 0 (nothing was started)", out.Stopped)
	}
}
