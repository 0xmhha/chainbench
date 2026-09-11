package wbft

import (
	"encoding/json"
	"strings"
	"testing"
)

const tmpl = `{
  "config": { "chainId": __CHAIN_ID__, "anzeon": {
    "init": { "validators": "__VALIDATORS_JSON__", "blsPublicKeys": "__BLS_PUBLIC_KEYS_JSON__" }
  } },
  "extraData": "__EXTRA_DATA__"
}`

func TestBuildGenesis_SubstitutesAndValidates(t *testing.T) {
	out, err := BuildGenesis([]byte(tmpl), GenesisParams{
		ChainID:    8283,
		Validators: []string{"0xaaa", "0xbbb"},
		BLSKeys:    []string{"0x111", "0x222"},
		ExtraData:  "0xdeadbeef",
	})
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}

	var g struct {
		Config struct {
			ChainID int64 `json:"chainId"`
			Anzeon  struct {
				Init struct {
					Validators []string `json:"validators"`
					BLSKeys    []string `json:"blsPublicKeys"`
				} `json:"init"`
			} `json:"anzeon"`
		} `json:"config"`
		ExtraData string `json:"extraData"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	if g.Config.ChainID != 8283 {
		t.Errorf("chainId: %d", g.Config.ChainID)
	}
	if len(g.Config.Anzeon.Init.Validators) != 2 || g.Config.Anzeon.Init.Validators[0] != "0xaaa" {
		t.Errorf("validators: %v", g.Config.Anzeon.Init.Validators)
	}
	if len(g.Config.Anzeon.Init.BLSKeys) != 2 || g.Config.Anzeon.Init.BLSKeys[1] != "0x222" {
		t.Errorf("bls keys: %v", g.Config.Anzeon.Init.BLSKeys)
	}
	if g.ExtraData != "0xdeadbeef" {
		t.Errorf("extraData: %s", g.ExtraData)
	}
	// No placeholders should remain.
	if strings.Contains(string(out), "__") {
		t.Errorf("unsubstituted placeholder remains: %s", out)
	}
}

func TestBuildGenesis_Validation(t *testing.T) {
	if _, err := BuildGenesis([]byte(tmpl), GenesisParams{ChainID: 0}); err == nil {
		t.Error("expected error for chainId 0")
	}
	if _, err := BuildGenesis([]byte(tmpl), GenesisParams{
		ChainID: 1, Validators: []string{"0xa"}, BLSKeys: []string{},
	}); err == nil {
		t.Error("expected error for validator/bls length mismatch")
	}
}

// TestBuildGenesis_DerivesExtraDataForTheSetItIsGiven is the fix for a defect
// this move exposed: keys.Preset.Take narrowed the validator list but carried
// the full set's extra-data along, so a network run with fewer validators than
// its preset got a genesis whose extra-data named all of them. Genesis
// validation passes, and the chain then looks for producers that do not exist.
//
// It went unnoticed because the shipped preset has five validators and the
// usual run takes four, where BFT quorum happens to still be reachable.
func TestBuildGenesis_DerivesExtraDataForTheSetItIsGiven(t *testing.T) {
	const template = `{"config":{"chainId":__CHAIN_ID__},` +
		`"validators":"__VALIDATORS_JSON__","blsKeys":"__BLS_PUBLIC_KEYS_JSON__",` +
		`"extraData":"__EXTRA_DATA__"}`

	all := wbftGenesisParams()
	two := all
	two.Validators = all.Validators[:2]
	two.BLSKeys = all.BLSKeys[:2]

	full, err := BuildGenesis([]byte(template), all)
	if err != nil {
		t.Fatalf("BuildGenesis(all): %v", err)
	}
	narrowed, err := BuildGenesis([]byte(template), two)
	if err != nil {
		t.Fatalf("BuildGenesis(two): %v", err)
	}
	if string(full) == string(narrowed) {
		t.Fatal("a narrowed validator set produced the same genesis as the full one")
	}

	// And the derived value is exactly what ExtraData computes for that subset,
	// so the genesis and the validator list cannot disagree.
	want, err := ExtraData(two.Validators, two.BLSKeys)
	if err != nil {
		t.Fatalf("ExtraData: %v", err)
	}
	if !strings.Contains(string(narrowed), want) {
		t.Errorf("narrowed genesis does not carry the extra-data for its own set")
	}
}

// TestBuildGenesis_KeepsASuppliedExtraData covers reproducing a historical
// genesis, where the recorded extra-data is the point.
func TestBuildGenesis_KeepsASuppliedExtraData(t *testing.T) {
	p := wbftGenesisParams()
	p.ExtraData = "0xdeadbeef"
	out, err := BuildGenesis([]byte(`{"config":{"chainId":__CHAIN_ID__},`+
		`"validators":"__VALIDATORS_JSON__","blsKeys":"__BLS_PUBLIC_KEYS_JSON__",`+
		`"extraData":"__EXTRA_DATA__"}`), p)
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}
	if !strings.Contains(string(out), "0xdeadbeef") {
		t.Errorf("supplied extra-data was overwritten:\n%s", out)
	}
}

// wbftGenesisParams builds params with full-length values, so extra-data can
// actually be derived from them.
func wbftGenesisParams() GenesisParams {
	return GenesisParams{
		ChainID: 1337,
		Validators: []string{
			"0x1111000000000000000000000000000000000001",
			"0x2222000000000000000000000000000000000002",
			"0x3333000000000000000000000000000000000003",
		},
		BLSKeys: []string{
			"0x" + strings.Repeat("a", 96),
			"0x" + strings.Repeat("b", 96),
			"0x" + strings.Repeat("c", 96),
		},
	}
}

// sizingTmpl carries only the two sizing placeholders, so a failure points at the
// derivation rather than at anything else in a genesis.
const sizingTmpl = `{"config":{"chainId":__CHAIN_ID__,"croissant":{"wBFT":{` +
	`"epochLength":__EPOCH_LENGTH__,"targetValidators":__TARGET_VALIDATORS__},` +
	`"init":{"validators":"__VALIDATORS_JSON__","blsPublicKeys":"__BLS_PUBLIC_KEYS_JSON__"}}},` +
	`"extraData":"__EXTRA_DATA__"}`

// sizingOf builds sizingTmpl for n validators and returns (targetValidators,
// epochLength) as the genesis declares them.
func sizingOf(t *testing.T, n int) (int, int) {
	t.Helper()
	vals := make([]string, n)
	keys := make([]string, n)
	for i := range vals {
		vals[i] = "0xaaa"
		keys[i] = "0x111"
	}
	out, err := BuildGenesis([]byte(sizingTmpl), GenesisParams{
		ChainID: 8283, Validators: vals, BLSKeys: keys, ExtraData: "0xdeadbeef",
	})
	if err != nil {
		t.Fatalf("BuildGenesis with %d validators: %v", n, err)
	}
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
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	return g.Config.Croissant.WBFT.TargetValidators, g.Config.Croissant.WBFT.EpochLength
}

// TestBuildGenesis_TargetValidatorsIsTheSetItDeclares: the template used to carry
// a literal 1, so a genesis naming fifteen validators asked go-wbft for a set of
// one. go-wbft cuts the stake-sorted candidate list at targetValidators, so the
// first epoch that decided a set would have collapsed it to a single node.
func TestBuildGenesis_TargetValidatorsIsTheSetItDeclares(t *testing.T) {
	for _, n := range []int{1, 4, 15, 30} {
		target, _ := sizingOf(t, n)
		if target != n {
			t.Errorf("%d validators declared but targetValidators is %d", n, target)
		}
	}
}

// TestBuildGenesis_EpochIsNeverShorterThanTheTargetSet: go-wbft rejects a config
// whose epochLength is below targetValidators ("epochLength must be greater than
// or equal to targetValidators"), so raising one without the other produces a
// genesis the node refuses.
func TestBuildGenesis_EpochIsNeverShorterThanTheTargetSet(t *testing.T) {
	for _, n := range []int{1, 4, 10, 15, 30} {
		target, epoch := sizingOf(t, n)
		if epoch < target {
			t.Errorf("%d validators: epochLength %d is shorter than targetValidators %d, which go-wbft rejects", n, epoch, target)
		}
	}
}

// TestBuildGenesis_SmallSetsKeepTheTemplateEpoch: the derivation must not shorten
// the epoch a small set had. Ten was the template's literal, and an epoch of four
// blocks would change how often a four-validator chain re-decides its set for no
// reason connected to this fix.
func TestBuildGenesis_SmallSetsKeepTheTemplateEpoch(t *testing.T) {
	for _, n := range []int{1, 4, 10} {
		if _, epoch := sizingOf(t, n); epoch != minEpochLength {
			t.Errorf("%d validators: epochLength %d, want the floor %d", n, epoch, minEpochLength)
		}
	}
}
