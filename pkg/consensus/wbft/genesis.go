package wbft

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GenesisParams are the wbft-family inputs a chain supplies to materialize a
// genesis.json from a template. The full stablenet system-contract/alloc
// substitution (ported from network/internal/adapters/stablenet with its golden
// vectors) attaches here as the network module is absorbed; G2 implements the
// consensus-critical placeholders (chain id, validator set, BLS keys,
// extraData) that define block production for the wbft family.
type GenesisParams struct {
	ChainID    int64
	Validators []string // validator addresses (0x-hex)
	BLSKeys    []string // BLS public keys (0x-hex), aligned with Validators
	ExtraData  string   // RLP-encoded validator extra-data (0x-hex)
	// Members are the governance council addresses (0x-hex) that seed the
	// anzeon system contracts (stablenet). Required only when the template
	// carries the system-contract placeholders; empty for templates (e.g.
	// wbft/croissant) that have no system-contract section.
	Members []string
}

// Placeholder tokens substituted in the genesis template.
const (
	phChainID    = "__CHAIN_ID__"
	phValidators = "__VALIDATORS_JSON__"
	phBLSKeys    = "__BLS_PUBLIC_KEYS_JSON__"
	phExtraData  = "__EXTRA_DATA__"

	// System-contract (anzeon) placeholders. Each sits inside a JSON string
	// value ("params" entries are string→string), so it is substituted in
	// bare form with a comma-separated address/key list.
	phSCValidators = "__SC_VALIDATORS_CSV__" // active validator addresses
	phSCBLSKeys    = "__SC_BLS_PUBLIC_KEYS_CSV__"
	phSCMembers    = "__SC_MEMBERS_CSV__" // governance council addresses
)

// BuildGenesis substitutes the wbft-family placeholders in template with params
// and returns the resulting JSON, erroring if the result is not valid JSON.
// Both quoted ("__X__") and bare (__X__) placeholder forms are handled so a
// token may sit in a JSON string or a JSON value position.
func BuildGenesis(template []byte, p GenesisParams) ([]byte, error) {
	if p.ChainID <= 0 {
		return nil, fmt.Errorf("wbft genesis: invalid chain id %d", p.ChainID)
	}
	if len(p.Validators) != len(p.BLSKeys) {
		return nil, fmt.Errorf("wbft genesis: %d validators but %d BLS keys", len(p.Validators), len(p.BLSKeys))
	}

	valsJSON, err := json.Marshal(p.Validators)
	if err != nil {
		return nil, fmt.Errorf("wbft genesis: marshal validators: %w", err)
	}
	blsJSON, err := json.Marshal(p.BLSKeys)
	if err != nil {
		return nil, fmt.Errorf("wbft genesis: marshal bls keys: %w", err)
	}

	out := string(template)

	// System-contract templates (anzeon) require a governance member set. Fail
	// early with a clear message rather than emitting an empty members list the
	// node would later reject.
	if strings.Contains(out, phSCMembers) && len(p.Members) == 0 {
		return nil, fmt.Errorf("wbft genesis: template requires system-contract members but none supplied")
	}

	// value-position (unquoted) forms first, then quoted forms.
	out = strings.ReplaceAll(out, phChainID, strconv.FormatInt(p.ChainID, 10))
	out = strings.ReplaceAll(out, `"`+phValidators+`"`, string(valsJSON))
	out = strings.ReplaceAll(out, phValidators, string(valsJSON))
	out = strings.ReplaceAll(out, `"`+phBLSKeys+`"`, string(blsJSON))
	out = strings.ReplaceAll(out, phBLSKeys, string(blsJSON))
	out = strings.ReplaceAll(out, `"`+phExtraData+`"`, strconv.Quote(p.ExtraData))
	out = strings.ReplaceAll(out, phExtraData, p.ExtraData)

	// System-contract list placeholders (bare, inside JSON string values).
	out = strings.ReplaceAll(out, phSCValidators, strings.Join(p.Validators, ","))
	out = strings.ReplaceAll(out, phSCBLSKeys, strings.Join(p.BLSKeys, ","))
	out = strings.ReplaceAll(out, phSCMembers, strings.Join(p.Members, ","))

	if !json.Valid([]byte(out)) {
		return nil, fmt.Errorf("wbft genesis: substitution produced invalid JSON")
	}
	return []byte(out), nil
}
