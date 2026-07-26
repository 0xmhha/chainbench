package genesis_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/genesis"
)

const wemixBase = `{"config":{"chainId":8285,"petersburgBlock":0,"croissantBlock":20},"alloc":{}}`

func TestInjectCroissant(t *testing.T) {
	spec := genesis.CroissantSpec{
		Validators: []string{"0xaaa", "0xbbb"},
		BLSKeys:    []string{"0x111", "0x222"},
	}
	out, err := genesis.InjectCroissant([]byte(wemixBase), spec)
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		Config struct {
			Croissant struct {
				Init struct {
					Validators    []string `json:"validators"`
					BLSPublicKeys []string `json:"blsPublicKeys"`
				} `json:"init"`
				GovContracts map[string]struct {
					Address string `json:"address"`
				} `json:"govContracts"`
			} `json:"croissant"`
		} `json:"config"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("output not valid: %v\n%s", err, out)
	}
	c := g.Config.Croissant
	if len(c.Init.Validators) != 2 || c.Init.Validators[0] != "0xaaa" || c.Init.BLSPublicKeys[1] != "0x222" {
		t.Errorf("init: %+v", c.Init)
	}
	for _, k := range []string{"govConfig", "govStaking", "govRewardeeImp", "govNCP"} {
		if c.GovContracts[k].Address == "" {
			t.Errorf("missing govContract %s", k)
		}
	}
	if err := genesis.ValidateForks(out); err != nil {
		t.Errorf("injected genesis should pass fork validation: %v", err)
	}
}

func TestInjectCroissant_NeedsPetersburg(t *testing.T) {
	_, err := genesis.InjectCroissant([]byte(`{"config":{"chainId":8285,"croissantBlock":20}}`),
		genesis.CroissantSpec{Validators: []string{"0xa"}, BLSKeys: []string{"0x1"}})
	if err == nil || !strings.Contains(err.Error(), "petersburg") {
		t.Errorf("missing petersburg should error, got %v", err)
	}
}

func TestValidateForks(t *testing.T) {
	// petersburg missing
	if err := genesis.ValidateForks([]byte(`{"config":{"croissantBlock":0,"croissant":{}}}`)); err == nil {
		t.Error("missing petersburg should error")
	}
	// croissantBlock without section
	if err := genesis.ValidateForks([]byte(`{"config":{"petersburgBlock":0,"croissantBlock":20}}`)); err == nil {
		t.Error("croissantBlock without section should error")
	}
	// section without block
	if err := genesis.ValidateForks([]byte(`{"config":{"petersburgBlock":0,"croissant":{}}}`)); err == nil {
		t.Error("croissant section without block should error")
	}
	// pure wemix (no croissant at all) is valid
	if err := genesis.ValidateForks([]byte(`{"config":{"petersburgBlock":0}}`)); err != nil {
		t.Errorf("pure wemix genesis should pass: %v", err)
	}
}
