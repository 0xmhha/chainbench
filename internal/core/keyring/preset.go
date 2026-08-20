package keyring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PresetFile is the file a keyring's index lives in, inside the ring directory.
const PresetFile = "metadata.json"

// Entry is one node's material: the secret it is built from, and the public
// identity that follows from it.
//
// It replaces the three shapes this used to have — a read-side NodeKey without
// BLS, a write-side Node with it, and a registry Key with yet another field set
// — none of which converted to another.
type Entry struct {
	// Label names this entry within a ring. A preset entry is labelled by its
	// index ("node1"); a ring built by a command labels its own.
	Label Label
	// Index is the 1-based node number within the ring, or 0 when the entry did
	// not come from a numbered set.
	Index int
	// Nodekey is the secret. It redacts itself when formatted; reaching the
	// hex takes an explicit Hex call.
	Nodekey PrivateKey
	// Identity is everything public that derives from Nodekey.
	Identity
}

// Preset is a decoded key set: node identities, plus the network decisions the
// file has historically carried alongside them.
//
// The second half does not belong to a keyring — who validates and who holds a
// balance are properties of a network, not of a key — and moves to the network
// blueprint (worklist K5). The fields stay for now so existing presets keep
// working unchanged.
type Preset struct {
	// Nodes are the per-node identities. This is the keyring proper.
	Nodes []Entry

	// Validators are the validator addresses (0x-hex), in genesis order.
	Validators []string
	// BLSKeys are the validators' BLS public keys (0x-hex), aligned with
	// Validators.
	BLSKeys []string
	// ExtraData is the RLP-encoded validator extra-data (0x-hex) recorded in
	// the file, if any.
	//
	// It is derived from the validator set, so it is only valid for the whole
	// set: [Preset.Take] drops it, and the genesis builder recomputes one for
	// the set it is actually given.
	ExtraData string
	// Members are the governance council member addresses (0x-hex) that seed
	// the wbft-family system contracts. For the stablenet preset these equal
	// the validators; empty for families with no system contracts.
	Members []string
	// Alloc is the raw genesis pre-funded accounts object (address -> account)
	// exactly as it appears in the file, or nil when it funds no accounts.
	Alloc json.RawMessage
	// Password unlocks the keystores in this ring.
	Password string
}

// presetFile is the on-disk shape. It is unexported because the file format and
// the domain type are allowed to drift: the file keeps fields for
// compatibility that Preset no longer needs to expose.
type presetFile struct {
	Description           string          `json:"description,omitempty"`
	Warning               string          `json:"warning,omitempty"`
	Password              string          `json:"password"`
	Validators            []string        `json:"validators"`
	BLSPublicKeys         []string        `json:"blsPublicKeys"`
	ExtraData             string          `json:"extraData,omitempty"`
	SystemContractMembers string          `json:"systemContractMembers,omitempty"`
	SystemContractBLSKeys string          `json:"systemContractBlsKeys,omitempty"`
	Alloc                 json.RawMessage `json:"alloc,omitempty"`
	Nodes                 []presetNode    `json:"nodes"`
}

// presetNode is one node as the file records it.
type presetNode struct {
	Index        int    `json:"index"`
	Nodekey      string `json:"nodekey"`
	PublicKey    string `json:"publicKey"`
	Address      string `json:"address"`
	BLSPublicKey string `json:"blsPublicKey,omitempty"`
	BLSPoP       string `json:"blsPoP,omitempty"`
}

// LoadPreset reads <dir>/metadata.json and returns the decoded set.
//
// A node's public fields are read rather than re-derived: the file is the
// record of what a running network was given, and silently correcting it would
// hide a set whose identities and keys have come apart. Use [Entry.Verify] to
// check them on purpose.
func LoadPreset(dir string) (Preset, error) {
	path := filepath.Join(dir, PresetFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, fmt.Errorf("keyring: read preset: %w", err)
	}
	var f presetFile
	if err := json.Unmarshal(b, &f); err != nil {
		return Preset{}, fmt.Errorf("keyring: parse %s: %w", path, err)
	}
	if len(f.Validators) == 0 {
		return Preset{}, fmt.Errorf("keyring: %s has no validators", path)
	}
	if len(f.Validators) != len(f.BLSPublicKeys) {
		return Preset{}, fmt.Errorf("keyring: %s has %d validators but %d BLS keys",
			path, len(f.Validators), len(f.BLSPublicKeys))
	}

	nodes := make([]Entry, 0, len(f.Nodes))
	for _, n := range f.Nodes {
		key, err := ParsePrivateKey(n.Nodekey)
		if err != nil {
			return Preset{}, fmt.Errorf("keyring: %s node %d: %w", path, n.Index, err)
		}
		e := Entry{
			Index:   n.Index,
			Nodekey: key,
			Identity: Identity{
				PublicKey: n.PublicKey,
				Address:   n.Address,
			},
		}
		if n.BLSPublicKey != "" {
			e.BLS = &BLS{PublicKey: n.BLSPublicKey, PoP: n.BLSPoP}
		}
		nodes = append(nodes, e)
	}

	return Preset{
		Nodes:      nodes,
		Validators: f.Validators,
		BLSKeys:    f.BLSPublicKeys,
		ExtraData:  f.ExtraData,
		Members:    splitCSV(f.SystemContractMembers),
		Alloc:      f.Alloc,
		Password:   f.Password,
	}, nil
}

// Node returns the entry with the given 1-based index and whether it was found.
func (p Preset) Node(index int) (Entry, bool) {
	for _, n := range p.Nodes {
		if n.Index == index {
			return n, true
		}
	}
	return Entry{}, false
}

// Take returns the first n validators and their BLS keys, for a network smaller
// than the set. n<=0 or n>=len returns the whole set.
//
// ExtraData is dropped from a narrowed set on purpose: it encodes the validator
// set, so the file's copy describes all of them. Carrying it through would hand
// the genesis builder a validator set contradicting Validators, and the chain
// reads the extra-data, not the list.
//
// Node identities and the governance council are preserved in full: a
// two-validator network still runs on nodes drawn from the whole ring, and the
// council is independent of how many validators are active.
func (p Preset) Take(n int) Preset {
	if n <= 0 || n >= len(p.Validators) {
		return p
	}
	out := p
	out.Validators = p.Validators[:n]
	out.BLSKeys = p.BLSKeys[:n]
	out.ExtraData = ""
	return out
}

// Verify reports whether an entry's recorded public fields match what its
// nodekey actually derives. A mismatch means the file and the key have come
// apart: a node would launch with one identity while the genesis registers
// another, which shows up as a chain that produces no blocks.
func (e Entry) Verify() error {
	want, err := Derive(e.Nodekey, derivationFor(e.Identity))
	if err != nil {
		return err
	}
	if !strings.EqualFold(want.Address, e.Address) {
		return fmt.Errorf("keyring: node %d: key derives address %s but the file records %s",
			e.Index, want.Address, e.Address)
	}
	if want.PublicKey != e.PublicKey {
		return fmt.Errorf("keyring: node %d: key derives a different devp2p public key", e.Index)
	}
	if e.BLS != nil && (want.BLS == nil || want.BLS.PublicKey != e.BLS.PublicKey || want.BLS.PoP != e.BLS.PoP) {
		return fmt.Errorf("keyring: node %d: key derives different BLS material", e.Index)
	}
	return nil
}

// derivationFor asks for exactly as much as the identity claims to have, so
// verifying a poa entry does not compute BLS material it never had.
func derivationFor(id Identity) Derivation {
	if id.BLS != nil {
		return WithBLS
	}
	return AccountOnly
}

// splitCSV splits a comma-separated field into trimmed, non-empty entries.
func splitCSV(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
