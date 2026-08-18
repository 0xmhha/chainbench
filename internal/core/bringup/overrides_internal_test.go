package bringup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// configOverrides must collect only the genesis.overrides.* namespace, stripping
// the prefix, and ignore every other config key (and a bare prefix with no key).
func TestConfigOverrides(t *testing.T) {
	cfg := config.Values{
		"nodes.validators":            "4",
		"genesis.overrides.bohoBlock": "10",
		"genesis.overrides.foo":       "bar",
		"genesis.overrides.":          "ignored", // empty suffix
	}
	ov := configOverrides(cfg)
	if len(ov) != 2 {
		t.Fatalf("collected %d overrides, want 2: %v", len(ov), ov)
	}
	if ov["bohoBlock"] != "10" || ov["foo"] != "bar" {
		t.Errorf("wrong overrides: %v", ov)
	}
	if _, ok := ov["nodes.validators"]; ok {
		t.Error("non-override key leaked in")
	}

	if ov := configOverrides(config.Values{"nodes.validators": "4"}); ov != nil {
		t.Errorf("no overrides should return nil, got %v", ov)
	}
}

// syncModeFor: validators are always full; endpoints follow
// nodes.endpoint_syncmode (default full).
func TestSyncModeFor(t *testing.T) {
	snap := config.Values{"nodes.endpoint_syncmode": "snap"}
	if got := syncModeFor(snap, node.RoleValidator); got != "full" {
		t.Errorf("validator sync mode = %q, want full (must hold full state to seal)", got)
	}
	if got := syncModeFor(snap, node.RoleEndpoint); got != "snap" {
		t.Errorf("endpoint sync mode = %q, want snap", got)
	}
	if got := syncModeFor(config.Values{}, node.RoleEndpoint); got != "full" {
		t.Errorf("endpoint default sync mode = %q, want full", got)
	}
}
