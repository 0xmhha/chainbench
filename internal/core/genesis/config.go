package genesis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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

// ApplyConfigOverrides sets each key→value pair under the genesis `config`
// object, returning the new genesis bytes. Each value is used as raw JSON when
// it parses as a JSON value (a number, boolean, or object — e.g. "10" sets a
// numeric block) and is otherwise quoted as a JSON string (e.g. "v2"). Keys are
// applied in sorted order so the result is deterministic. Empty overrides return
// the input unchanged.
//
// This is the delayed-fork boundary: the setup phase supplies {"bohoBlock":"10"} to
// move a fork off genesis so a network can be launched with a fork activating at
// block N. It is engine-agnostic — no fork name is baked in here; the caller
// (from config `genesis.overrides.*`) names the config keys.
func ApplyConfigOverrides(genesisJSON []byte, overrides map[string]string) ([]byte, error) {
	if len(overrides) == 0 {
		return genesisJSON, nil
	}
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := genesisJSON
	for _, k := range keys {
		next, err := SetConfigSection(out, k, jsonScalar(overrides[k]))
		if err != nil {
			return nil, fmt.Errorf("genesis: override %q: %w", k, err)
		}
		out = next
	}
	return out, nil
}

// MergeOverride deep-merges overlay into genesisJSON and returns the result.
// Objects merge key-by-key recursively; a non-object overlay value (scalar,
// array, or a new key) replaces the base value at that key. This adds genesis
// fragments — e.g. extra alloc accounts or system-contract params — without
// disturbing the rest of the genesis. Empty overlay returns the input unchanged.
// It is engine-agnostic: the overlay is caller-supplied data (a launch overlay
// file), so no chain-specific structure is baked in here.
func MergeOverride(genesisJSON, overlay []byte) ([]byte, error) {
	if len(bytes.TrimSpace(overlay)) == 0 {
		return genesisJSON, nil
	}
	var base, over map[string]json.RawMessage
	if err := json.Unmarshal(genesisJSON, &base); err != nil {
		return nil, fmt.Errorf("genesis: parse for overlay: %w", err)
	}
	if err := json.Unmarshal(overlay, &over); err != nil {
		return nil, fmt.Errorf("genesis: parse overlay: %w", err)
	}
	merged, err := mergeObjects(base, over)
	if err != nil {
		return nil, err
	}
	return json.Marshal(merged)
}

// mergeObjects recursively merges over into base (base is mutated and returned).
func mergeObjects(base, over map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	if base == nil {
		base = map[string]json.RawMessage{}
	}
	for k, ov := range over {
		bv, exists := base[k]
		bo, bIsObj := asObject(bv)
		oo, oIsObj := asObject(ov)
		if exists && bIsObj && oIsObj {
			m, err := mergeObjects(bo, oo)
			if err != nil {
				return nil, err
			}
			raw, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			base[k] = raw
			continue
		}
		base[k] = ov // replace: scalar, array, new key, or type change
	}
	return base, nil
}

// asObject reports whether raw is a JSON object and, if so, its decoded form.
func asObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || t[0] != '{' {
		return nil, false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(t, &m); err != nil {
		return nil, false
	}
	return m, true
}

// jsonScalar renders a flat config value as raw JSON: a value that is already
// valid JSON (number/bool/object/quoted string) is used verbatim, so "10" stays
// the number 10; anything else is quoted as a JSON string, so a bare "v2"
// becomes "v2".
func jsonScalar(v string) json.RawMessage {
	if json.Valid([]byte(v)) {
		return json.RawMessage(v)
	}
	return json.RawMessage(strconv.Quote(v))
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
