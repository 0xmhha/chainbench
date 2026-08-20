package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestNetworkPlan_CountsDriveTheLayout(t *testing.T) {
	three, one := 3, 1
	out, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "wbft", DataDir: t.TempDir(), Validators: &three, Endpoints: &one,
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if len(out.Plan.Nodes) != 4 {
		t.Fatalf("got %d nodes, want 3 validators + 1 endpoint", len(out.Plan.Nodes))
	}
	for i, want := range []node.Role{node.RoleValidator, node.RoleValidator, node.RoleValidator, node.RoleEndpoint} {
		if got := out.Plan.Nodes[i].Role; got != want {
			t.Errorf("node%d role = %v, want %v", i+1, got, want)
		}
	}
	if out.Plugin.Manifest().ID != "wbft" {
		t.Errorf("plugin = %s, want wbft", out.Plugin.Manifest().ID)
	}
}

func TestNetworkPlan_ZeroEndpointsIsARequest(t *testing.T) {
	// Nil and zero must not mean the same thing: a caller asking for no
	// endpoints has to be able to say so, which is why the counts are pointers.
	one, zero := 1, 0
	withZero, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "wbft", DataDir: t.TempDir(), Validators: &one, Endpoints: &zero,
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if len(withZero.Plan.Nodes) != 1 {
		t.Fatalf("endpoints=0 planned %d nodes, want the validator only", len(withZero.Plan.Nodes))
	}
}

func TestNetworkPlan_TopologyChainLosesToAnExplicitChain(t *testing.T) {
	path := writeTopology(t, "wbft")

	// Without an explicit chain the topology's own chain is used...
	implicit, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "stablenet", TopologyPath: path, DataDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if implicit.Plugin.Manifest().ID != "wbft" {
		t.Errorf("chain = %s, want the topology's wbft", implicit.Plugin.Manifest().ID)
	}

	// ...but a chain the caller asked for is never silently overridden.
	explicit, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "stablenet", ChainExplicit: true, TopologyPath: path, DataDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if explicit.Plugin.Manifest().ID != "stablenet" {
		t.Errorf("chain = %s, want the requested stablenet", explicit.Plugin.Manifest().ID)
	}
}

func TestNetworkPlan_TopologyDrivesRolesAndBootnode(t *testing.T) {
	out, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		TopologyPath: writeTopology(t, "wbft"), DataDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if len(out.Plan.Nodes) != 3 {
		t.Fatalf("got %d nodes, want the topology's 3", len(out.Plan.Nodes))
	}
	if out.Plan.Nodes[1].Role != node.RoleEndpoint {
		t.Errorf("node2 role = %v, want endpoint", out.Plan.Nodes[1].Role)
	}
	if out.BootnodeIndex != 1 {
		t.Errorf("bootnode = %d, want 1", out.BootnodeIndex)
	}
}

func TestNetworkPlan_SetOverridesReachTheConfig(t *testing.T) {
	out, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "stablenet", DataDir: t.TempDir(),
		Set: []string{"genesis.overrides.bohoBlock=10"},
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if got := out.Config.String("genesis.overrides.bohoBlock", ""); got != "10" {
		t.Errorf("override = %q, want 10", got)
	}
	// A delayed fork is advertised so the fork-transition cases gate on it.
	if !hasCapability(out.Plan.Capabilities, "delayed-boho") {
		t.Errorf("capabilities = %v, want delayed-boho", out.Plan.Capabilities)
	}
}

func TestNetworkPlan_MalformedSetIsRejected(t *testing.T) {
	_, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "stablenet", DataDir: t.TempDir(), Set: []string{"novalue"},
	})
	if err == nil {
		t.Fatal("want an error for an override without a value")
	}
	if !strings.Contains(err.Error(), "key=value") {
		t.Errorf("error should say what was expected, got: %v", err)
	}
}

func TestNetworkPlan_GenesisOverlayAddsCapabilities(t *testing.T) {
	dir := t.TempDir()
	overlay := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlay, []byte(`{"capabilities":["account-extra"],"genesis":{"nonce":"0x1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "stablenet", DataDir: dir, GenesisOverlayPath: overlay,
	})
	if err != nil {
		t.Fatalf("NetworkPlan: %v", err)
	}
	if !hasCapability(out.Plan.Capabilities, "account-extra") {
		t.Errorf("capabilities = %v, want account-extra", out.Plan.Capabilities)
	}
	if out.Config.String("genesis.overlay", "") == "" {
		t.Error("the overlay's genesis fragment did not reach the config")
	}
}

func TestNetworkPlan_UnknownChainIsNamed(t *testing.T) {
	_, err := app.NetworkPlan(context.Background(), app.Deps{}, app.NetworkSpecIn{
		Chain: "nosuchchain", DataDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("want an error for an unregistered chain")
	}
}

func TestNetworkLaunch_RequiresAResolvedBinary(t *testing.T) {
	// Resolving the binary on PATH is a surface convenience; this layer refuses
	// to guess, because a remote launch's path is not a local one.
	_, err := app.NetworkLaunch(context.Background(), app.Deps{}, app.NetworkLaunchIn{
		Spec: app.NetworkSpecIn{Chain: "stablenet", DataDir: t.TempDir()},
	})
	if err == nil {
		t.Fatal("want an error without a binary")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("error should name the missing binary, got: %v", err)
	}
}

func TestNetworkProvision_WritesGenesisConfigsAndTopology(t *testing.T) {
	dir := t.TempDir()
	topo := writeTopology(t, "stablenet")

	res, err := app.NetworkProvision(context.Background(), app.Deps{}, app.NetworkProvisionIn{
		Spec:    app.NetworkSpecIn{TopologyPath: topo, DataDir: dir},
		KeysDir: presetKeysDir,
	})
	if err != nil {
		t.Fatalf("NetworkProvision: %v", err)
	}
	if len(res.Plan.Plan.Nodes) != 3 {
		t.Fatalf("planned %d nodes, want 3", len(res.Plan.Plan.Nodes))
	}
	for _, f := range []string{"genesis.json", "config_node1.toml", "topology.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
}

// presetKeysDir is the repository's shipped key set.
const presetKeysDir = "../../keys/preset"

// writeTopology writes a three-node topology (bp, en, bp) with node 1 as the
// bootnode.
func writeTopology(t *testing.T, chain string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "topology.yaml")
	body := "chain: " + chain + `
nodes:
  - index: 1
    role: bp
    bootnode: true
  - index: 2
    role: en
  - index: 3
    role: bp
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write topology: %v", err)
	}
	return path
}

// hasCapability reports whether caps advertises want.
func hasCapability(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}
