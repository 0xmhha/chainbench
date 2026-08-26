package store

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/accounts/keystore"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// File and directory permissions for a generated ring.
// Keystore scrypt parameters. Light (geth's LightScryptN/P): a generated ring
// is a local test fixture, and lighter parameters keep multi-node generation
// fast while still producing a standard, node-readable v3 keystore.
const (
	keystoreScryptN = 1 << 12
	keystoreScryptP = 6
)

// GenerateOpts configures ring generation.
type GenerateOpts struct {
	// Files is where the ring is written and read; nil is this machine's
	// filesystem. A remote store places the ring on a server through the same
	// boundary provision uses, so "where the ring lives" is a caller's choice
	// rather than a second code path.
	Files filestore.Store
	// Nodes is how many identities to create.
	Nodes int
	// Validators is how many identities join the validator set.
	//
	// It is a pointer so that "the caller said nothing" and "the caller said
	// none" are different values rather than both being zero. They mean opposite
	// things, and each verb resolves an unset count its own way — [Generate]
	// takes all of them, [Extend] takes none — so no caller can pre-resolve it
	// without getting one of the two wrong.
	Validators *int
	// Out is the ring directory.
	Out string
	// Password encrypts the keystores.
	Password string
	// Balance pre-funds each generated account in the genesis alloc (0x-hex
	// wei).
	Balance string
	// Derive selects how much of each identity to compute. BLS material is only
	// used by the wbft family, and asking for it where it is not used produces
	// keys nobody reads.
	Derive Derivation
	// Rand supplies the entropy. Nil uses crypto/rand; a test passes its own so
	// generation is reproducible.
	Rand io.Reader
}

// Generate creates an N-node ring in opts.Out and returns it. progress, when
// non-nil, receives a line per node.
//
// Nothing is executed: every identity is derived in process, so a ring can be
// generated with no chain binary built or on PATH. That is what lets a network
// be declared from scratch rather than starting from a committed fixture.
func Generate(opts GenerateOpts, progress func(string)) (Preset, error) {
	return GenerateAt(context.Background(), opts, progress)
}

// GenerateAt is Generate against opts.Files (nil = local), with the context a
// remote store's I/O needs.
func GenerateAt(ctx context.Context, opts GenerateOpts, progress func(string)) (Preset, error) {
	// Creating over a ring that already exists would replace identities a
	// genesis, a datadir, or a test is already referring to, and the keys behind
	// them cannot be recovered. Adding to a ring is a different verb.
	if exists, err := opts.files().Exists(ctx, filepath.Join(opts.Out, PresetFile)); err == nil && exists {
		return Preset{}, fmt.Errorf("keyring: %s already holds a ring; add to it instead of creating over it", opts.Out)
	}
	// A ring generated without saying otherwise is a network's validator set,
	// which is what every existing preset is.
	want := opts.Nodes
	if opts.Validators != nil {
		want = *opts.Validators
	}
	if want < 0 || want > opts.Nodes {
		want = opts.Nodes
	}
	opts.Validators = &want
	return generate(ctx, Preset{}, opts, progress)
}

// files is the ring's store, defaulting to this machine.
func (o GenerateOpts) files() filestore.Store {
	if o.Files != nil {
		return o.Files
	}
	return filestore.Local{}
}

// Extend adds opts.Nodes entries to the ring already in opts.Out, keeping the
// ones that are there.
//
// Extending rather than regenerating matters because identities are referenced
// elsewhere the moment they exist — in a genesis, in a running datadir, in a
// test's declaration. Regenerating would silently replace them.
//
// opts.Validators is how many of the *new* entries join the validator set, and
// it defaults to none. Changing who validates changes what the chain is, so it
// is asked for rather than inferred from a count.
func Extend(opts GenerateOpts, progress func(string)) (Preset, error) {
	return ExtendAt(context.Background(), opts, progress)
}

// ExtendAt is Extend against opts.Files (nil = local).
func ExtendAt(ctx context.Context, opts GenerateOpts, progress func(string)) (Preset, error) {
	// Extending without saying otherwise promotes nobody: adding an identity and
	// changing who validates are different decisions.
	want := 0
	if opts.Validators != nil {
		want = *opts.Validators
	}
	if want > opts.Nodes {
		return Preset{}, fmt.Errorf("keyring: extend: %d new validators from %d new nodes",
			want, opts.Nodes)
	}
	opts.Validators = &want
	existing, err := LoadPresetAt(ctx, opts.files(), opts.Out)
	if err != nil {
		return Preset{}, fmt.Errorf("keyring: extend: %w", err)
	}
	return generate(ctx, existing, opts, progress)
}

// Import adds a key the caller already holds to the ring in dir, under label.
//
// It writes the entry into the ring's index, not into a directory beside it: an
// identity that the index does not list is one that `list` and `show` cannot
// see and a network cannot use, which is worse than not importing it at all.
func Import(dir string, label Label, key PrivateKey, d Derivation) (Entry, error) {
	return ImportAt(context.Background(), nil, dir, label, key, d)
}

// ImportAt is Import against files (nil = local).
func ImportAt(ctx context.Context, files filestore.Store, dir string, label Label, key PrivateKey, d Derivation) (Entry, error) {
	if files == nil {
		files = filestore.Local{}
	}
	if label == "" {
		return Entry{}, fmt.Errorf("keyring: import needs a label")
	}
	set, err := LoadPresetAt(ctx, files, dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Entry{}, err
	}
	for _, e := range set.Nodes {
		if e.Label == label {
			return Entry{}, fmt.Errorf("keyring: %s already holds %q", dir, label)
		}
	}

	id, err := derive.Derive(key, d)
	if err != nil {
		return Entry{}, err
	}
	e := Entry{Label: label, Index: len(set.Nodes) + 1, Nodekey: key, Identity: id}
	if err := writeEntryDir(ctx, files, filepath.Join(dir, string(label)), key, id, set.Password); err != nil {
		return Entry{}, err
	}
	set.Nodes = append(set.Nodes, e)
	if err := writePreset(ctx, GenerateOpts{Out: dir, Files: files}, set); err != nil {
		return Entry{}, err
	}
	return e, nil
}

func generate(ctx context.Context, existing Preset, opts GenerateOpts, progress func(string)) (Preset, error) {
	if opts.Nodes < 1 {
		return Preset{}, fmt.Errorf("keyring: nodes must be >= 1")
	}
	if opts.Rand == nil {
		opts.Rand = rand.Reader
	}
	if err := prepareRingDir(ctx, opts); err != nil {
		return Preset{}, err
	}
	set, err := appendEntries(ctx, existing, opts, progress)
	if err != nil {
		return Preset{}, err
	}
	if err := writePreset(ctx, opts, set); err != nil {
		return Preset{}, err
	}
	return set, nil
}

// prepareRingDir creates the ring directory and the shared password file, which
// is what a node unlocks with at launch (--password); the keystores use the
// same password.
func prepareRingDir(ctx context.Context, opts GenerateOpts) error {
	// The store creates parent directories on write, so the ring directory is
	// born with its first file.
	return opts.files().Write(ctx, filepath.Join(opts.Out, "password"), []byte(opts.Password), secretPerm)
}

// appendEntries creates opts.Nodes identities after the ones already in
// existing, and folds each into the ring's network decisions.
func appendEntries(ctx context.Context, existing Preset, opts GenerateOpts, progress func(string)) (Preset, error) {
	alloc, err := decodeAlloc(existing.Network.Alloc)
	if err != nil {
		return Preset{}, err
	}
	set := existing
	set.Password = opts.Password

	first := len(existing.Nodes) + 1
	promoted := 0
	for i := first; i < first+opts.Nodes; i++ {
		e, err := generateEntry(ctx, i, opts)
		if err != nil {
			return Preset{}, fmt.Errorf("keyring: node %d: %w", i, err)
		}
		set.Nodes = append(set.Nodes, e)
		alloc[strings.TrimPrefix(e.Address, "0x")] = map[string]any{"balance": opts.Balance}
		if promoted < *opts.Validators {
			set.Network = promote(set.Network, e)
			promoted++
		}
		if progress != nil {
			progress(describeEntry(e))
		}
	}

	set.Network.Members = append([]string(nil), set.Network.Validators...)
	// Extra-data is derived from the validator set, so it cannot survive a
	// change to that set.
	set.Network.ExtraData = ""

	raw, err := json.Marshal(alloc)
	if err != nil {
		return Preset{}, err
	}
	set.Network.Alloc = raw
	return set, nil
}

// promote adds one identity to the network's validator set, carrying its BLS
// key when it has one so the two lists stay index-aligned.
func promote(net Network, e Entry) Network {
	net.Validators = append(net.Validators, e.Address)
	if e.BLS != nil {
		net.BLSKeys = append(net.BLSKeys, e.BLS.PublicKey)
	}
	return net
}

// writePreset renders the index file.
//
// No extra-data is written: it encodes the validator set, so a stored copy goes
// stale as soon as a network runs a subset of these validators — and a genesis
// whose extra-data disagrees with its validator set is accepted, then fails in
// consensus. The family derives it at genesis time.
//
// The alloc is written even for a ring that declares no validator set, and it
// is the one network decision that survives here. Generated identities are in
// no existing genesis, so without a balance their first transaction cannot pay
// for gas, and nothing else can supply one until the network blueprint can
// declare accounts of its own.
func writePreset(ctx context.Context, opts GenerateOpts, set Preset) error {
	f := presetFile{
		Description: fmt.Sprintf("Generated ring: %d nodes (%d validators). chainbench keyring.",
			len(set.Nodes), len(set.Network.Validators)),
		Warning:               "TEST FIXTURE ONLY — do not import to mainnet/testnet.",
		Password:              set.Password,
		Validators:            set.Network.Validators,
		BLSPublicKeys:         set.Network.BLSKeys,
		SystemContractMembers: strings.Join(set.Network.Members, ","),
		SystemContractBLSKeys: strings.Join(set.Network.BLSKeys, ","),
		Alloc:                 set.Network.Alloc,
	}
	for _, e := range set.Nodes {
		n := presetNode{
			Index:     e.Index,
			Nodekey:   e.Nodekey.Hex(),
			PublicKey: e.PublicKey,
			Address:   e.Address,
		}
		if e.Label != nodeLabel(e.Index) {
			n.Label = string(e.Label)
		}
		if e.BLS != nil {
			n.BLSPublicKey, n.BLSPoP = e.BLS.PublicKey, e.BLS.PoP
		}
		f.Nodes = append(f.Nodes, n)
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return opts.files().Write(ctx, filepath.Join(opts.Out, PresetFile), b, publicPerm)
}

// generateEntry creates node i's material and writes its directory: the
// nodekey, an encrypted v3 keystore, and the public fields as plain files for
// an operator reading the directory by hand.
func generateEntry(ctx context.Context, i int, opts GenerateOpts) (Entry, error) {
	nodeDir := filepath.Join(opts.Out, fmt.Sprintf("node%d", i))
	key, err := derive.NewPrivateKey(opts.Rand)
	if err != nil {
		return Entry{}, err
	}
	id, err := derive.Derive(key, opts.Derive)
	if err != nil {
		return Entry{}, err
	}
	if err := writeEntryDir(ctx, opts.files(), nodeDir, key, id, opts.Password); err != nil {
		return Entry{}, err
	}
	return Entry{Label: nodeLabel(i), Index: i, Nodekey: key, Identity: id}, nil
}

// writeEntryDir lays out one identity's directory: the key, an encrypted
// keystore, and the derived public fields as plain files for an operator
// reading the directory by hand.
func writeEntryDir(ctx context.Context, files filestore.Store, dir string, key PrivateKey, id Identity, password string) error {
	if err := files.Write(ctx, filepath.Join(dir, "nodekey"), []byte(key.Hex()), secretPerm); err != nil {
		return err
	}
	if err := writeKeystore(ctx, files, dir, key, id.Address, password); err != nil {
		return err
	}
	public := map[string]string{"address": id.Address, "pubkey": id.PublicKey}
	if id.BLS != nil {
		public["bls_pubkey"] = id.BLS.PublicKey
	}
	for name, val := range public {
		if err := files.Write(ctx, filepath.Join(dir, name), []byte(val), publicPerm); err != nil {
			return err
		}
	}
	return nil
}

// decodeAlloc reads back an existing alloc so extending a ring keeps the
// balances already granted.
func decodeAlloc(raw json.RawMessage) (map[string]map[string]any, error) {
	out := map[string]map[string]any{}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("keyring: existing alloc: %w", err)
	}
	return out, nil
}

// describeEntry is the progress line for one generated entry.
func describeEntry(e Entry) string {
	if e.BLS == nil {
		return fmt.Sprintf("node %d  %s", e.Index, e.Address)
	}
	return fmt.Sprintf("node %d  %s  bls=%s…", e.Index, e.Address, shortHex(e.BLS.PublicKey))
}

// writeKeystore encrypts the account key into a standard v3 keystore where the
// node reads it (<datadir>/keystore), which is what the node's own `account
// import` used to produce — without shelling out to it.
func writeKeystore(ctx context.Context, files filestore.Store, nodeDir string, key PrivateKey, address, password string) error {
	keyjson, err := keystore.Encrypt(key.Bytes(), password, keystoreScryptN, keystoreScryptP)
	if err != nil {
		return fmt.Errorf("keystore encrypt: %w", err)
	}
	dir := filepath.Join(nodeDir, "keystore")
	return files.Write(ctx, filepath.Join(dir, keystoreFilename(address)), keyjson, secretPerm)
}

// keystoreFilename is the geth-convention keystore name
// (UTC--<timestamp>--<address>), which a node's keystore reader accepts.
func keystoreFilename(address string) string {
	ts := time.Now().UTC().Format("2006-01-02T15-04-05.000000000Z")
	return "UTC--" + ts + "--" + strings.TrimPrefix(strings.ToLower(address), "0x")
}

// shortHex abbreviates a 0x-hex value for a progress line.
func shortHex(s string) string {
	const shown = 14
	if len(s) <= shown {
		return s
	}
	return s[:shown]
}

// ImportRing clones a whole ring — every identity, its labels, and the
// network declaration (validators, BLS set, alloc) — into dir, re-encrypting
// keystores with password (empty keeps the source ring's).
//
// Every entry is verified before anything is written: the key must re-derive
// the address, devp2p public key and BLS material the source index records.
// That is the transfer's integrity check, and it is stronger than a checksum —
// a checksum proves the bytes arrived, this proves the keys still ARE the
// identities the index claims.
//
// The destination must not already hold a ring, for the same reason Generate
// refuses one: silently replacing referenced identities is unrecoverable.
func ImportRing(ctx context.Context, files filestore.Store, dir string, src Preset, password string) (Preset, error) {
	if files == nil {
		files = filestore.Local{}
	}
	if len(src.Nodes) == 0 {
		return Preset{}, fmt.Errorf("keyring: import-ring: the source ring holds no identities")
	}
	if exists, err := files.Exists(ctx, filepath.Join(dir, PresetFile)); err == nil && exists {
		return Preset{}, fmt.Errorf("keyring: %s already holds a ring; add to it instead of creating over it", dir)
	}
	for _, e := range src.Nodes {
		if err := e.Verify(); err != nil {
			return Preset{}, fmt.Errorf("keyring: import-ring: source entry %q failed verification: %w", e.Label, err)
		}
	}
	if password == "" {
		password = src.Password
	}
	if err := files.Write(ctx, filepath.Join(dir, "password"), []byte(password), secretPerm); err != nil {
		return Preset{}, err
	}
	for _, e := range src.Nodes {
		entryDir := filepath.Join(dir, string(e.Label))
		if err := writeEntryDir(ctx, files, entryDir, e.Nodekey, e.Identity, password); err != nil {
			return Preset{}, fmt.Errorf("keyring: import-ring: %q: %w", e.Label, err)
		}
	}
	set := src
	set.Password = password
	if err := writePreset(ctx, GenerateOpts{Out: dir, Files: files}, set); err != nil {
		return Preset{}, err
	}
	return set, nil
}
