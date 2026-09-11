package wbft

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// The derivation of targetValidators/epochLength lives in the consensus family
// and is tested there. What can only be tested here is whether the SHIPPED
// template actually asks for it: a template that goes back to a literal leaves
// the derivation correct and unreached, which is exactly how a genesis declaring
// fifteen validators came to ask go-wbft for a set of one.

// builtFor renders the embedded template for n validators, the way a chain
// creation does.
func builtFor(t *testing.T, n int) []byte {
	t.Helper()
	vals := make([]string, n)
	keys := make([]string, n)
	for i := range vals {
		// Distinct addresses and keys, because extraData is derived from them.
		vals[i] = "0x" + strings.Repeat("0", 39) + string(rune('1'+i%9))
		keys[i] = "0x" + strings.Repeat("a", 95) + string(rune('1'+i%9))
	}
	out, err := plugin{}.Family().BuildGenesis(genesisTmpl, registry.GenesisParams{
		ChainID: 8285, Validators: vals, BLSKeys: keys, Members: vals,
	})
	if err != nil {
		t.Fatalf("BuildGenesis for %d validators: %v", n, err)
	}
	return out
}

func sizing(t *testing.T, out []byte) (target, epoch int) {
	t.Helper()
	var g struct {
		Config struct {
			Croissant struct {
				WBFT struct {
					EpochLength      int `json:"epochLength"`
					TargetValidators int `json:"targetValidators"`
				} `json:"wBFT"`
			} `json:"croissant"`
		} `json:"config"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("built genesis is not valid JSON: %v", err)
	}
	return g.Config.Croissant.WBFT.TargetValidators, g.Config.Croissant.WBFT.EpochLength
}

// TestShippedTemplate_SizesItselfToTheValidatorSet is the coupling test: the
// template must carry the placeholders, not a number.
func TestShippedTemplate_SizesItselfToTheValidatorSet(t *testing.T) {
	for _, n := range []int{4, 15} {
		target, epoch := sizing(t, builtFor(t, n))
		if target != n {
			t.Errorf("%d validators: targetValidators is %d — the template is carrying a literal again", n, target)
		}
		if epoch < target {
			t.Errorf("%d validators: epochLength %d < targetValidators %d, which go-wbft rejects", n, epoch, target)
		}
	}
}

// TestShippedTemplate_LeavesNoPlaceholder: a token the builder does not know
// would survive into the genesis. It cannot survive as valid JSON in value
// position, but it can inside a string, where a node would read it as data.
func TestShippedTemplate_LeavesNoPlaceholder(t *testing.T) {
	if out := builtFor(t, 4); strings.Contains(string(out), "__") {
		t.Errorf("unsubstituted placeholder remains in the built genesis:\n%s", out)
	}
}
