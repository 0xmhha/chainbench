package genesis

import (
	"encoding/json"
	"fmt"
)

// SetConfigSection merges section into a genesis' `config` object under key,
// returning the new genesis bytes. It is engine-agnostic: the section content
// (e.g. an "anzeon" or "croissant" consensus block) is data supplied by the
// caller from a template/manifest, never baked into this package. This is how a
// wemix genesis gains the "croissant" section (from the wbft chain's genesis
// template) so a go-wbft node can take over at the fork — without hardcoding any
// chain-specific consensus config here.
func SetConfigSection(genesisJSON []byte, key string, section json.RawMessage) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("genesis: empty config-section key")
	}
	if len(section) == 0 || !json.Valid(section) {
		return nil, fmt.Errorf("genesis: config section %q is empty or invalid JSON", key)
	}
	var g map[string]json.RawMessage
	if err := json.Unmarshal(genesisJSON, &g); err != nil {
		return nil, fmt.Errorf("genesis: parse: %w", err)
	}
	var cfg map[string]json.RawMessage
	if len(g["config"]) > 0 {
		if err := json.Unmarshal(g["config"], &cfg); err != nil {
			return nil, fmt.Errorf("genesis: parse config: %w", err)
		}
	} else {
		cfg = map[string]json.RawMessage{}
	}
	cfg[key] = section
	merged, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	g["config"] = merged
	return json.Marshal(g)
}

// ExtractConfigSection returns the raw JSON of a genesis' `config.<key>` (e.g.
// pull the fully-substituted "croissant" section out of a built wbft genesis so
// it can be merged into a wemix genesis). Missing key returns (nil, nil).
func ExtractConfigSection(genesisJSON []byte, key string) (json.RawMessage, error) {
	var g struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(genesisJSON, &g); err != nil {
		return nil, fmt.Errorf("genesis: parse: %w", err)
	}
	return g.Config[key], nil
}

// ValidateForks checks a genesis' fork configuration for two consensus-critical
// conditions found the hard way: petersburgBlock must be present (else go-wbft
// rejects the genesis on fork ordering), and if a croissantBlock is set then a
// croissant config section must be present, and vice versa (else go-wbft falls
// back to ethash and rejects the pre-fork blocks).
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
