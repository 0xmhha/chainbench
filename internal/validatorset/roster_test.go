package validatorset_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/validatorset"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

func preset(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		p := filepath.Join(dir, "keys", "preset")
		if _, err := os.Stat(filepath.Join(p, "metadata.json")); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("keys/preset not found")
		}
		dir = parent
	}
}

func TestRoster_WbftFamilyHasValidators(t *testing.T) {
	r, err := validatorset.Load("stablenet", preset(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Family != "wbft" {
		t.Fatalf("family = %q, want wbft", r.Family)
	}
	var validators, nodes, gov int
	for _, a := range r.Accounts {
		switch a.Role {
		case validatorset.RoleValidator:
			validators++
			if a.Detail != "BLS present" {
				t.Fatalf("stablenet validator should have BLS: %+v", a)
			}
		case validatorset.RoleNode:
			nodes++
		case validatorset.RoleGovernance:
			gov++
		}
	}
	if validators != 4 || nodes < 4 {
		t.Fatalf("roster counts: validators=%d nodes=%d gov=%d", validators, nodes, gov)
	}
}

func TestRoster_PoaFamilyNoGenesisValidators(t *testing.T) {
	r, err := validatorset.Load("wemix", preset(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Family != "poa" {
		t.Fatalf("family = %q, want poa", r.Family)
	}
	for _, a := range r.Accounts {
		if a.Role == validatorset.RoleValidator {
			t.Fatalf("poa chain should not list genesis validators: %+v", a)
		}
	}
	if r.Note == "" {
		t.Fatal("poa roster should note validators are set at bootstrap")
	}
}

func TestRoster_UnknownChain(t *testing.T) {
	if _, err := validatorset.Load("nope", preset(t)); err == nil {
		t.Fatal("expected error for unknown chain")
	}
}
