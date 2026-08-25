// Ring storage: the on-disk layout of a keyring — the index file, the
// per-entry directories, and the keystore/raw backends — read and written
// through the provision file seam so a ring lives the same way on this
// machine or on a server. The key model (what an entry IS) stays in the
// keyring package; this package only persists it.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/provision"
)

// PresetFile is the file a keyring's index lives in, inside the ring directory.
const PresetFile = "metadata.json"

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
	return LoadPresetAt(context.Background(), nil, dir)
}

// LoadPresetAt is LoadPreset through files (nil = local): the ring's index is
// one file, so a ring on a server reads back with a single remote read.
func LoadPresetAt(ctx context.Context, files provision.FileStore, dir string) (Preset, error) {
	if files == nil {
		files = provision.LocalFileStore{}
	}
	path := filepath.Join(dir, PresetFile)
	b, err := files.Read(ctx, path)
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
		key, err := keyring.ParsePrivateKey(n.Nodekey)
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
