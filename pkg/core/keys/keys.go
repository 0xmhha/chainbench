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
	"strings"
)

// NodeKey is one preset node's identity for building static-node enodes.
type NodeKey struct {
	Index     int    // 1-based node index
	PublicKey string // 128-hex devp2p public key (no 0x prefix)
	Address   string // 0x-hex account address
	Nodekey   string // 64-hex devp2p private key (the file placed in the datadir)
}

// Preset is a decoded preset key set.
type Preset struct {
	// Validators are the validator addresses (0x-hex), in genesis order.
	Validators []string
	// BLSKeys are the validators' BLS public keys (0x-hex), aligned with
	// Validators.
	BLSKeys []string
	// ExtraData is the RLP-encoded validator extra-data (0x-hex).
	ExtraData string
	// Members are the governance council member addresses (0x-hex) that seed
	// the wbft-family system contracts (govValidator/govMinter/...). For the
	// stablenet preset these equal the validators; the field is empty for
	// presets whose family has no system contracts. Decoded from the
	// comma-separated "systemContractMembers" metadata field.
	Members []string
	// Alloc is the raw genesis pre-funded accounts object (address -> account)
	// exactly as it appears in the metadata, or nil when the preset funds no
	// accounts. Passed through verbatim into the genesis "alloc" field.
	Alloc json.RawMessage
	// Password unlocks the preset keystores.
	Password string
	// Nodes are the per-node devp2p identities (for static-node enodes).
	Nodes []NodeKey
}

type metadata struct {
	Password              string          `json:"password"`
	Validators            []string        `json:"validators"`
	BLSPublicKeys         []string        `json:"blsPublicKeys"`
	ExtraData             string          `json:"extraData"`
	SystemContractMembers string          `json:"systemContractMembers"`
	Alloc                 json.RawMessage `json:"alloc"`
	Nodes                 []struct {
		Index     int    `json:"index"`
		PublicKey string `json:"publicKey"`
		Address   string `json:"address"`
		Nodekey   string `json:"nodekey"`
	} `json:"nodes"`
}

// splitCSV splits a comma-separated metadata field into trimmed, non-empty
// entries. An empty or whitespace-only string yields nil.
func splitCSV(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
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
	nodes := make([]NodeKey, 0, len(m.Nodes))
	for _, n := range m.Nodes {
		nodes = append(nodes, NodeKey{Index: n.Index, PublicKey: n.PublicKey, Address: n.Address, Nodekey: n.Nodekey})
	}
	return Preset{
		Validators: m.Validators,
		BLSKeys:    m.BLSPublicKeys,
		ExtraData:  m.ExtraData,
		Members:    splitCSV(m.SystemContractMembers),
		Alloc:      m.Alloc,
		Password:   m.Password,
		Nodes:      nodes,
	}, nil
}

// Node returns the preset node with the given 1-based index and whether it was
// found.
func (p Preset) Node(index int) (NodeKey, bool) {
	for _, n := range p.Nodes {
		if n.Index == index {
			return n, true
		}
	}
	return NodeKey{}, false
}

// Take returns the first n validators/BLS keys, for networks smaller than the
// preset. n<=0 or n>=len returns the full set. The governance Members (system
// contract council) are independent of the active validator count and are
// preserved in full.
func (p Preset) Take(n int) Preset {
	if n <= 0 || n >= len(p.Validators) {
		return p
	}
	return Preset{
		Validators: p.Validators[:n],
		BLSKeys:    p.BLSKeys[:n],
		ExtraData:  p.ExtraData,
		Members:    p.Members,
		Alloc:      p.Alloc,
		Password:   p.Password,
	}
}
