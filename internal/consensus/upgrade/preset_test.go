package upgrade_test

import (
	"path/filepath"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// goldenPresetPath resolves the repo's golden upgrade profile from this test's
// package dir (internal/consensus/upgrade -> repo root).
func goldenPresetPath() string {
	return filepath.Join("..", "..", "..", "presets", "hardfork", "wemix-upgrade.yaml")
}

// TestGoldenPreset_SaysWhichForkAndHowManyOfEachSide.
//
// What a hardfork preset still decides, now that the composition is the
// ordinary one: which chain hands over to which, at which fork, and how many
// nodes stand on each side. Everything else it used to drive — the plan, the
// ports, the launch — belongs to the composition steps.
func TestGoldenPreset_SaysWhichForkAndHowManyOfEachSide(t *testing.T) {
	p, err := upgrade.LoadChainPreset(goldenPresetPath())
	if err != nil {
		t.Fatal(err)
	}
	if p.Upgrade.From != "wemix" || p.Upgrade.To != "wbft" || p.Upgrade.AtFork != "croissant" {
		t.Fatalf("golden profile upgrade section unexpected: %+v", p.Upgrade)
	}
	if p.Roles.Validators < 4 || p.Roles.Producers < 1 {
		t.Fatalf("golden profile roles below verified minimum: %+v", p.Roles)
	}
	if _, err := registry.Get(p.Upgrade.From); err != nil {
		t.Errorf("the from-chain is not a registered chain: %v", err)
	}
	if _, err := registry.Get(p.Upgrade.To); err != nil {
		t.Errorf("the to-chain is not a registered chain: %v", err)
	}
}
