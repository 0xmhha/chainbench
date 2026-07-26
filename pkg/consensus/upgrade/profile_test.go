package upgrade_test

import (
	"path/filepath"
	"testing"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	"github.com/0xmhha/chainbench/pkg/consensus/upgrade"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// goldenProfilePath resolves the repo's golden upgrade profile from this test's
// package dir (pkg/consensus/upgrade -> repo root).
func goldenProfilePath() string {
	return filepath.Join("..", "..", "..", "profiles", "wemix-upgrade.yaml")
}

// The golden profile must load and drive BuildPlan to a valid plan. This is the
// end-to-end contract of "golden profile, no code defaults": the file alone
// (plus a bootstrap from-genesis) fully specifies a launchable handoff.
func TestGoldenProfile_DrivesBuildPlan(t *testing.T) {
	p, err := upgrade.LoadProfile(goldenProfilePath())
	if err != nil {
		t.Fatal(err)
	}
	if p.Upgrade.From != "wemix" || p.Upgrade.To != "wbft" || p.Upgrade.AtFork != "croissant" {
		t.Fatalf("golden profile upgrade section unexpected: %+v", p.Upgrade)
	}
	if p.Roles.Validators < 4 || p.Roles.Producers < 1 {
		t.Fatalf("golden profile roles below verified minimum: %+v", p.Roles)
	}

	in, err := p.Inputs([]byte(fromGenesis))
	if err != nil {
		t.Fatal(err)
	}
	from, err := registry.Get(p.Upgrade.From)
	if err != nil {
		t.Fatal(err)
	}
	to, err := registry.Get(p.Upgrade.To)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := upgrade.BuildPlan(from, to, in)
	if err != nil {
		t.Fatalf("golden profile does not build a valid plan: %v", err)
	}
	if len(plan.Nodes) != p.Roles.Producers+p.Roles.Validators {
		t.Errorf("plan node count %d != profile roles %d+%d", len(plan.Nodes), p.Roles.Producers, p.Roles.Validators)
	}
}

func TestProfile_Inputs_Rejects(t *testing.T) {
	base, err := upgrade.LoadProfile(goldenProfilePath())
	if err != nil {
		t.Fatal(err)
	}
	// mismatched validator / bls counts must be caught, not silently defaulted.
	bad := base
	bad.Validators.BLSPublicKeys = bad.Validators.BLSPublicKeys[:1]
	if _, err := bad.Inputs([]byte(fromGenesis)); err == nil {
		t.Error("mismatched validator/bls counts should error")
	}
	// no producer members.
	bad = base
	bad.Producers.Members = nil
	if _, err := bad.Inputs([]byte(fromGenesis)); err == nil {
		t.Error("missing producer members should error")
	}
}
