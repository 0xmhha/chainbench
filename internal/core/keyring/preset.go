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

// Network is what a *network* decides about a ring's identities: which of them
// validate, which seed the governance council, and who starts with a balance.
//
// None of it is a property of a key. Two networks can run from one ring with
// different validator sets, and a ring generated for one network is usable by
// another only because these answers are not baked into it.
//
// A preset file may record them, because presets predate the blueprint that
// owns them. [Preset.NetworkFor] is how a caller asks the question without
// caring whether the file had an answer.
type Network struct {
	// Validators are the validator addresses (0x-hex), in genesis order.
	Validators []string
	// BLSKeys are the validators' BLS public keys (0x-hex), aligned with
	// Validators. Empty for a family that does not use BLS.
	BLSKeys []string
	// ExtraData is the RLP-encoded validator extra-data (0x-hex), when the file
	// recorded one for exactly this set. It is derived from the validator set,
	// so it is only ever carried for the set it was computed from; the genesis
	// builder recomputes it whenever it is empty.
	ExtraData string
	// Members are the governance council addresses (0x-hex) that seed the
	// wbft-family system contracts. Empty for families with no system contracts.
	Members []string
	// Alloc is the raw genesis pre-funded accounts object (address -> account),
	// or nil when the network funds no accounts.
	Alloc json.RawMessage
}

// Preset is a decoded ring: the identities it holds, and — for a file that
// still carries them — the network decisions recorded beside them.
//
// A ring that declares no validator set is the point: it is identities and
// nothing more, so what a network does with them is the network's to say.
type Preset struct {
	// Nodes are the per-node identities. This is the keyring proper.
	Nodes []Entry
	// Network holds the decisions the file recorded, if any. It is empty for a
	// ring that declares only identities.
	Network Network
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
	Index int `json:"index"`
	// Label is the name this identity carries. It is omitted for the numbered
	// identities a generated ring holds, whose label follows from the index; an
	// imported one ("faucet") records its own, because nothing else could
	// recover it.
	Label        string `json:"label,omitempty"`
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
	if err := f.validate(path); err != nil {
		return Preset{}, err
	}
	nodes, err := f.entries(path)
	if err != nil {
		return Preset{}, err
	}
	return Preset{
		Nodes: nodes,
		Network: Network{
			Validators: f.Validators,
			BLSKeys:    f.BLSPublicKeys,
			ExtraData:  f.ExtraData,
			Members:    splitCSV(f.SystemContractMembers),
			Alloc:      f.Alloc,
		},
		Password: f.Password,
	}, nil
}

// validate rejects a file that cannot describe a usable ring.
func (f presetFile) validate(path string) error {
	// A ring may hold identities and declare no validator set (the network
	// decides), or declare a set whose keys it does not hold (a network you did
	// not create). A file that does neither says nothing at all.
	if len(f.Nodes) == 0 && len(f.Validators) == 0 {
		return fmt.Errorf("keyring: %s holds no identities and declares no validators", path)
	}
	// BLS keys are optional as a set — the poa family has none — but if any are
	// present they are read positionally against the validators, so a partial
	// list would silently attach one validator's key to another.
	if len(f.BLSPublicKeys) != 0 && len(f.BLSPublicKeys) != len(f.Validators) {
		return fmt.Errorf("keyring: %s has %d validators but %d BLS keys",
			path, len(f.Validators), len(f.BLSPublicKeys))
	}
	return nil
}

// entries decodes the file's identities.
func (f presetFile) entries(path string) ([]Entry, error) {
	out := make([]Entry, 0, len(f.Nodes))
	for _, n := range f.Nodes {
		key, err := ParsePrivateKey(n.Nodekey)
		if err != nil {
			return nil, fmt.Errorf("keyring: %s node %d: %w", path, n.Index, err)
		}
		label := Label(n.Label)
		if label == "" {
			// A numbered identity's label follows from its index, so reading a
			// ring and generating one name entries the same way.
			label = nodeLabel(n.Index)
		}
		e := Entry{
			Label:   label,
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
		out = append(out, e)
	}
	return out, nil
}

// nodeLabel is the label a numbered identity carries: node1, node2, ...
func nodeLabel(index int) Label { return Label(fmt.Sprintf("node%d", index)) }

// Node returns the entry with the given 1-based index and whether it was found.
func (p Preset) Node(index int) (Entry, bool) {
	for _, n := range p.Nodes {
		if n.Index == index {
			return n, true
		}
	}
	return Entry{}, false
}

// NetworkFor answers who validates in a network of n validators.
//
// When the ring's file recorded a validator set, the first n of it are taken.
// When it recorded none — a ring that is identities and nothing else — the
// first n identities are used. Either way the caller asks the same question and
// does not have to know which kind of ring it was handed.
//
// n<=0 or n beyond what is available means "all of them".
//
// ExtraData survives only when the whole recorded set is used. It encodes the
// validator set, so a narrowed one would describe validators the network never
// starts; the genesis builder recomputes it from the set it is given.
//
// The governance council is not narrowed: it is independent of how many
// validators are active.
func (p Preset) NetworkFor(n int) Network {
	if len(p.Network.Validators) > 0 {
		out := p.Network
		if n > 0 && n < len(out.Validators) {
			// Copied, not resliced. A truncated slice shares its array, so a
			// caller appending to the result would write over the preset's own
			// validator list.
			out.Validators = truncate(out.Validators, n)
			out.BLSKeys = truncate(out.BLSKeys, n)
			out.ExtraData = ""
		}
		return out
	}

	// A ring with no declared set: the network's validators are its first n
	// identities, in ring order.
	out := p.Network
	limit := len(p.Nodes)
	if n > 0 && n < limit {
		limit = n
	}
	for _, e := range p.Nodes[:limit] {
		out.Validators = append(out.Validators, e.Address)
		if e.BLS != nil {
			out.BLSKeys = append(out.BLSKeys, e.BLS.PublicKey)
		}
	}
	if len(out.Members) == 0 {
		// A governance council with no members cannot pass anything, so a
		// network that needs one and was told nothing seats its validators —
		// which is what every existing preset records anyway.
		out.Members = append([]string(nil), out.Validators...)
	}
	return out
}

// truncate returns the first n of s as a new slice, tolerating a shorter s so a
// set with no BLS keys narrows without a bounds check at every call site.
//
// It copies rather than reslices: the result outlives this call, and a shared
// array is how an append on the result silently rewrites the original.
func truncate(s []string, n int) []string {
	if n >= len(s) {
		return append([]string(nil), s...)
	}
	return append([]string(nil), s[:n]...)
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
