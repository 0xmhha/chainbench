package genesis

import (
	"encoding/json"
	"fmt"
)

// CroissantSpec is the post-fork wbft consensus configuration injected into a
// wemix genesis so a go-wbft node can (a) validate the pre-croissant wemix/PoA
// blocks via its wpoa engine and (b) produce blocks after the croissant block.
// Validators/BLSKeys are the post-fork wbft validator set (croissant.init).
type CroissantSpec struct {
	Validators []string // 0x-hex validator addresses
	BLSKeys    []string // 0x-hex BLS public keys, aligned with Validators
}

// govContract addresses are the fixed system-contract slots the croissant
// engine reads (params/config_wbft.go). They are consensus constants, not
// operator inputs.
const (
	govConfigAddr      = "0x0000000000000000000000000000000000001000"
	govStakingAddr     = "0x0000000000000000000000000000000000001001"
	govRewardeeImpAddr = "0x0000000000000000000000000000000000001002"
	govNCPAddr         = "0x0000000000000000000000000000000000001003"
)

// InjectCroissant adds the config.croissant section (WBFT + init + govContracts)
// to a wemix genesis and returns the new genesis bytes. Without this section
// go-wbft's ChainConfig.CroissantEnabled() is false, so it falls back to ethash
// validation and rejects the wemix diff=1 blocks ("invalid difficulty: have 1,
// want 131072"); with it, go-wbft selects the wpoa+wbft CroissantEngine. It
// also requires petersburgBlock to be present, without which go-wbft refuses
// the genesis on fork ordering.
func InjectCroissant(genesisJSON []byte, spec CroissantSpec) ([]byte, error) {
	if len(spec.Validators) == 0 || len(spec.Validators) != len(spec.BLSKeys) {
		return nil, fmt.Errorf("genesis: croissant needs matching validators/bls (%d/%d)", len(spec.Validators), len(spec.BLSKeys))
	}
	var g map[string]json.RawMessage
	if err := json.Unmarshal(genesisJSON, &g); err != nil {
		return nil, fmt.Errorf("genesis: parse for croissant: %w", err)
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(g["config"], &cfg); err != nil {
		return nil, fmt.Errorf("genesis: parse config for croissant: %w", err)
	}
	if _, ok := cfg["petersburgBlock"]; !ok {
		return nil, fmt.Errorf("genesis: petersburgBlock must be set before croissant (go-wbft rejects the fork ordering otherwise)")
	}

	ncps := ""
	for i, v := range spec.Validators {
		if i > 0 {
			ncps += ","
		}
		ncps += v
	}
	croissant := map[string]any{
		"wBFT": map[string]any{
			"requestTimeoutSeconds": 1, "blockPeriodSeconds": 1, "epochLength": 100,
			"proposerPolicy": 0, "maxRequestTimeoutSeconds": nil, "stabilizingStakersThreshold": 1,
			"targetValidators": len(spec.Validators),
		},
		"init": map[string]any{"validators": spec.Validators, "blsPublicKeys": spec.BLSKeys},
		"govContracts": map[string]any{
			"govConfig": map[string]any{"address": govConfigAddr, "version": "v1", "params": map[string]string{
				"minimumStaking": "10000000000000000000000000", "maximumStaking": "100000000000000000000000000",
				"unbondingPeriodStaker": "604800", "unbondingPeriodDelegator": "259200",
				"feePrecision": "10000", "changeFeeDelay": "604800", "govCouncil": govNCPAddr,
			}},
			"govStaking":     map[string]any{"address": govStakingAddr, "version": "v1"},
			"govRewardeeImp": map[string]any{"address": govRewardeeImpAddr, "version": "v1"},
			"govNCP":         map[string]any{"address": govNCPAddr, "version": "v1", "params": map[string]string{"ncps": ncps}},
		},
	}
	raw, err := json.Marshal(croissant)
	if err != nil {
		return nil, err
	}
	cfg["croissant"] = raw
	if g["config"], err = json.Marshal(cfg); err != nil {
		return nil, err
	}
	return json.Marshal(g)
}

// ValidateForks checks a genesis' fork configuration for the two consensus-
// critical conditions found the hard way: petersburgBlock must be present (else
// go-wbft rejects the genesis), and if a croissantBlock is set then a croissant
// config section must be present (else go-wbft falls back to ethash).
func ValidateForks(genesisJSON []byte) error {
	var g struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(genesisJSON, &g); err != nil {
		return fmt.Errorf("genesis: parse for fork check: %w", err)
	}
	if _, ok := g.Config["petersburgBlock"]; !ok {
		return fmt.Errorf("genesis: petersburgBlock must be set (go-wbft rejects the fork ordering otherwise)")
	}
	_, hasBlock := g.Config["croissantBlock"]
	_, hasSection := g.Config["croissant"]
	if hasBlock != hasSection {
		return fmt.Errorf("genesis: croissantBlock and the croissant config section must be set together (block=%v section=%v)", hasBlock, hasSection)
	}
	return nil
}
