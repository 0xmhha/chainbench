// KeySet storage: the on-disk layout of a keyring — the index file, the
// per-entry directories, and the keystore/raw backends — read and written
// through the provision file boundary so a ring lives the same way on this
// machine or on a server. The key model (what an entry IS) stays in the
// keyring package; this package only persists it.
package preset

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// KeyIndexFile is the file a keyring's index lives in, inside the ring directory.
const KeyIndexFile = "metadata.json"

// KeyNode is one node as the file records it.
type KeyNode struct {
	Index int `json:"index"`
	// Label is the name this identity carries. It is omitted for the numbered
	// identities a generated ring holds, whose label follows from the index; an
	// imported one ("faucet") records its own, because nothing else could
	// recover it.
	Label string `json:"label,omitempty"`
	// Nodekey is what this format used to carry and no longer writes: the private
	// key lives in node<N>/nodekey, and duplicating it in the shared index meant
	// one remote read of the index disclosed every key in the ring.
	//
	// It is still PARSED, but only by legacyIndexKeys, and only when a caller has
	// asked for keys on a ring whose per-entry files are missing. The identity
	// read never looks at it, which is the whole point.
	Nodekey      string `json:"nodekey,omitempty"`
	PublicKey    string `json:"publicKey"`
	Address      string `json:"address"`
	BLSPublicKey string `json:"blsPublicKey,omitempty"`
	BLSPoP       string `json:"blsPoP,omitempty"`
}

// LoadKeyPreset reads <dir>/metadata.json and returns the decoded set.
//
// A node's public fields are read rather than re-derived: the file is the
// record of what a running network was given, and silently correcting it would
// hide a set whose identities and keys have come apart. Use [Entry.Verify] to
// check them on purpose.
func LoadKeyPreset(dir string) (Key, error) {
	return LoadKeyPresetAt(context.Background(), nil, dir)
}

// LoadKeyPresetWithKeysAt is LoadKeyPreset through files (nil = local): the ring's index is one
// file, so a ring on a server reads back with a single remote read.
//
// It returns IDENTITIES. The index used to carry the nodekeys as well, which made
// that single remote read a disclosure of every key in the ring — listing a ring
// cost the same as exporting one, and nothing said so. The keys now live only in
// node<N>/nodekey, and a caller that needs them asks for
// [LoadKeyPresetWithKeysAt] and pays N reads for it.
// LoadKeyPresetWithKeysAt is LoadKeyPresetAt plus each identity's private key, read
// from its own file (node<N>/nodekey).
//
// It is deliberately the longer call. Identities answer nearly every question a
// ring is asked — which addresses validate, does this ring match that genesis,
// what is node3's devp2p key — and none of them need a secret. The two callers
// that do need one say so by asking for it: verification re-derives an identity
// from its key, and export discloses it on purpose.
//
// The cost is N reads instead of one, which on a remote ring is N round trips.
// That is the price of the secret not travelling for the other questions, and it
// is only paid by the two that need it.
func LoadKeyPresetWithKeysAt(ctx context.Context, files filestore.Store, dir string) (Key, error) {
	set, err := LoadKeyPresetAt(ctx, files, dir)
	if err != nil {
		return Key{}, err
	}
	if files == nil {
		files = filestore.Local{}
	}
	legacy, lerr := legacyIndexKeys(ctx, files, dir)
	if lerr != nil {
		return Key{}, lerr
	}
	for i := range set.Nodes {
		key, kerr := nodeKeyAt(ctx, files, dir, set.Nodes[i].Index)
		if kerr == nil {
			set.Nodes[i].Nodekey = key
			continue
		}
		// A ring written before the index stopped carrying keys may have no
		// per-entry key file. Fall back to the index's copy for THIS call only —
		// the identity read never looks there, so an old ring keeps working
		// without making every read of it a disclosure again.
		if old, ok := legacy[set.Nodes[i].Index]; ok {
			set.Nodes[i].Nodekey = old
			continue
		}
		return Key{}, kerr
	}
	return set, nil
}

// legacyIndexKeys reads the private keys an older index carried, if any. A ring
// written by a current chainbench has none and this returns an empty map.
func legacyIndexKeys(ctx context.Context, files filestore.Store, dir string) (map[int]derive.PrivateKey, error) {
	b, err := files.Read(ctx, filepath.Join(dir, KeyIndexFile))
	if err != nil {
		return nil, fmt.Errorf("keyring: read keys: %w", err)
	}
	var f KeyFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("keyring: parse %s: %w", filepath.Join(dir, KeyIndexFile), err)
	}
	out := map[int]derive.PrivateKey{}
	for _, n := range f.Nodes {
		if n.Nodekey == "" {
			continue
		}
		key, perr := derive.ParsePrivateKey(n.Nodekey)
		if perr != nil {
			return nil, fmt.Errorf("keyring: %s node %d: %w", KeyIndexFile, n.Index, perr)
		}
		out[n.Index] = key
	}
	return out, nil
}

// LoadKeyPresetWithKeys is LoadKeyPresetWithKeysAt on the local filesystem.
func LoadKeyPresetWithKeys(dir string) (Key, error) {
	return LoadKeyPresetWithKeysAt(context.Background(), nil, dir)
}

// nodeKeyAt reads one identity's private key from node<N>/nodekey.
//
// It is the only way to obtain a key from a ring on disk. The index used to
// carry a copy, which made every read of it a disclosure of the whole ring; now
// a caller that wants one key reads one key.
func nodeKeyAt(ctx context.Context, files filestore.Store, dir string, index int) (derive.PrivateKey, error) {
	if files == nil {
		files = filestore.Local{}
	}
	path := filepath.Join(dir, string(NodeLabel(index)), "nodekey")
	b, err := files.Read(ctx, path)
	if err != nil {
		return derive.PrivateKey{}, fmt.Errorf("keyring: read %s: %w", path, err)
	}
	key, err := derive.ParsePrivateKey(strings.TrimSpace(string(b)))
	if err != nil {
		return derive.PrivateKey{}, fmt.Errorf("keyring: %s: %w", path, err)
	}
	return key, nil
}

func LoadKeyPresetAt(ctx context.Context, files filestore.Store, dir string) (Key, error) {
	if files == nil {
		files = filestore.Local{}
	}
	path := filepath.Join(dir, KeyIndexFile)
	b, err := files.Read(ctx, path)
	if err != nil {
		return Key{}, fmt.Errorf("keyring: read keys: %w", err)
	}
	var f KeyFile
	if err := json.Unmarshal(b, &f); err != nil {
		return Key{}, fmt.Errorf("keyring: parse %s: %w", path, err)
	}
	if err := f.validate(path); err != nil {
		return Key{}, err
	}
	nodes, err := f.entries(path)
	if err != nil {
		return Key{}, err
	}
	return Key{
		Nodes: nodes,
		Network: keyring.Network{
			Validators: f.Validators,
			BLSKeys:    f.BLSPublicKeys,
			ExtraData:  f.ExtraData,
			Members:    splitCSV(f.SystemContractMembers),
			Alloc:      f.Alloc,
		},
		Password: f.Password,
	}, nil
}

func (f KeyFile) validate(path string) error {
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
func (f KeyFile) entries(path string) ([]keyring.Entry, error) {
	out := make([]keyring.Entry, 0, len(f.Nodes))
	for _, n := range f.Nodes {
		label := keyring.Label(n.Label)
		if label == "" {
			// A numbered identity's label follows from its index, so reading a
			// ring and generating one name entries the same way.
			label = NodeLabel(n.Index)
		}
		e := keyring.Entry{
			Label: label,
			Index: n.Index,
			Identity: derive.Identity{
				PublicKey: n.PublicKey,
				Address:   n.Address,
			},
		}
		if n.BLSPublicKey != "" {
			e.BLS = &derive.BLS{PublicKey: n.BLSPublicKey, PoP: n.BLSPoP}
		}
		out = append(out, e)
	}
	return out, nil
}

// NodeLabel is the label a numbered identity carries: node1, node2, ...
func NodeLabel(index int) keyring.Label { return keyring.Label(fmt.Sprintf("node%d", index)) }

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

// KeyFile is the on-disk shape. It is unexported because the file format and
// the domain type are allowed to drift: the file keeps fields for
// compatibility that Preset no longer needs to expose.
type KeyFile struct {
	Description           string          `json:"description,omitempty"`
	Warning               string          `json:"warning,omitempty"`
	Password              string          `json:"password"`
	Validators            []string        `json:"validators"`
	BLSPublicKeys         []string        `json:"blsPublicKeys"`
	ExtraData             string          `json:"extraData,omitempty"`
	SystemContractMembers string          `json:"systemContractMembers,omitempty"`
	SystemContractBLSKeys string          `json:"systemContractBlsKeys,omitempty"`
	Alloc                 json.RawMessage `json:"alloc,omitempty"`
	Nodes                 []KeyNode       `json:"nodes"`
}

// LoadKeyPresetWithAccounts is LoadKeyPreset plus each identity's sealing account,
// read from its own keystore.
//
// It is the longer call for the same reason LoadKeyPresetWithKeys is: the index
// answers nearly every question a ring is asked in one read, and this one costs
// a directory read per node. The two callers that need it are the ones that
// have to agree — the genesis that funds and stakes the account, and the launch
// that unlocks it. Before this they did not read it at all: both assumed the
// account was the address the nodekey derives, and a ring where it is not
// produced a node that starts and dies on "no key for given address or file".
func LoadKeyPresetWithAccounts(dir string) (Key, error) {
	return loadKeyPresetWithAccountsAt(context.Background(), nil, dir)
}

// loadKeyPresetWithAccountsAt is LoadKeyPresetWithAccounts through files (nil = local).
func loadKeyPresetWithAccountsAt(ctx context.Context, files filestore.Store, dir string) (Key, error) {
	set, err := LoadKeyPresetAt(ctx, files, dir)
	if err != nil {
		return Key{}, err
	}
	for i := range set.Nodes {
		acct, aerr := keystoreAccount(dir, set.Nodes[i].Index)
		if aerr != nil {
			// A ring entry with no keystore is ordinary: only a node that seals
			// needs one. Its account is the address its nodekey derives, which
			// is what SealingAccount answers when this is empty.
			continue
		}
		if !strings.EqualFold(acct, set.Nodes[i].Address) {
			set.Nodes[i].Account = acct
		}
	}
	return set, nil
}

// keystoreAccount is the address one entry's keystore holds.
//
// Read from THIS machine. A key set is the operator's, and the keystore file is
// named with its account and a timestamp, so it is found by listing a directory
// — which a remote store cannot do. Every other reader of a keystore in this
// repository already works that way and re-roots the path when it ships the file
// to a target.
//
// A directory holding more than one takes the first by name. A node seals with
// one account, and a ring that gave it two has not said which; guessing the same
// way in one place is better than guessing differently in two.
func keystoreAccount(dir string, index int) (string, error) {
	ksDir := filepath.Join(dir, fmt.Sprintf("node%d", index), "keystore")
	ents, err := os.ReadDir(ksDir)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		b, rerr := os.ReadFile(filepath.Join(ksDir, name))
		if rerr != nil {
			continue
		}
		var doc struct {
			Address string `json:"address"`
		}
		if json.Unmarshal(b, &doc) != nil || doc.Address == "" {
			continue
		}
		return "0x" + strings.TrimPrefix(doc.Address, "0x"), nil
	}
	return "", fmt.Errorf("keyring: node%d has no keystore under %s", index, ksDir)
}
