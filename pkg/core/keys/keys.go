// Package keys loads a preset key set — the validator addresses, BLS public
// keys, and RLP extra-data a wbft-family genesis needs — from a metadata.json
// (the reproducible "static" key mode). It turns on-disk preset material into
// the inputs the genesis builder consumes, so a setup produces a real genesis
// rather than placeholders (docs/CHAINBENCH_GO_REDESIGN.md §8).
package keys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Preset is a decoded preset key set.
type Preset struct {
	// Validators are the validator addresses (0x-hex), in genesis order.
	Validators []string
	// BLSKeys are the validators' BLS public keys (0x-hex), aligned with
	// Validators.
	BLSKeys []string
	// ExtraData is the RLP-encoded validator extra-data (0x-hex).
	ExtraData string
	// Password unlocks the preset keystores.
	Password string
}

type metadata struct {
	Password      string   `json:"password"`
	Validators    []string `json:"validators"`
	BLSPublicKeys []string `json:"blsPublicKeys"`
	ExtraData     string   `json:"extraData"`
}

// LoadPreset reads <dir>/metadata.json and returns the decoded Preset.
func LoadPreset(dir string) (Preset, error) {
	path := filepath.Join(dir, "metadata.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, fmt.Errorf("keys: read preset metadata: %w", err)
	}
	var m metadata
	if err := json.Unmarshal(b, &m); err != nil {
		return Preset{}, fmt.Errorf("keys: parse %s: %w", path, err)
	}
	if len(m.Validators) == 0 {
		return Preset{}, fmt.Errorf("keys: %s has no validators", path)
	}
	if len(m.Validators) != len(m.BLSPublicKeys) {
		return Preset{}, fmt.Errorf("keys: %s has %d validators but %d BLS keys",
			path, len(m.Validators), len(m.BLSPublicKeys))
	}
	return Preset{
		Validators: m.Validators,
		BLSKeys:    m.BLSPublicKeys,
		ExtraData:  m.ExtraData,
		Password:   m.Password,
	}, nil
}

// Take returns the first n validators/BLS keys, for networks smaller than the
// preset. n<=0 or n>=len returns the full set.
func (p Preset) Take(n int) Preset {
	if n <= 0 || n >= len(p.Validators) {
		return p
	}
	return Preset{
		Validators: p.Validators[:n],
		BLSKeys:    p.BLSKeys[:n],
		ExtraData:  p.ExtraData,
		Password:   p.Password,
	}
}
