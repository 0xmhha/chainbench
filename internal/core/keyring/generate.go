package keyring

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/accounts/keystore"
)

// File and directory permissions for a generated ring.
const (
	dirPerm    os.FileMode = 0o755
	secretPerm os.FileMode = 0o600
	publicPerm os.FileMode = 0o644
)

// Keystore scrypt parameters. Light (geth's LightScryptN/P): a generated ring
// is a local test fixture, and lighter parameters keep multi-node generation
// fast while still producing a standard, node-readable v3 keystore.
const (
	keystoreScryptN = 1 << 12
	keystoreScryptP = 6
)

// NoValidators asks [Generate] to declare no validator set: the ring holds
// identities, and a network says which of them validate.
//
// It is a sentinel rather than 0 because 0 is what an unset field looks like,
// and "the caller said nothing" has to keep meaning "all of them" for the rings
// that already exist.
const NoValidators = -1

// GenerateOpts configures ring generation.
type GenerateOpts struct {
	// Nodes is how many identities to create.
	Nodes int
	// Validators is how many of them join the validator set. Unset means all of
	// them for a new ring and none for an extended one; [NoValidators] declares
	// none explicitly.
	Validators int
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
	switch {
	case opts.Validators == NoValidators:
		opts.Validators = 0
	case opts.Validators < 1 || opts.Validators > opts.Nodes:
		// A ring generated without saying otherwise is a network's validator
		// set, which is what every existing preset is.
		opts.Validators = opts.Nodes
	}
	return generate(Preset{}, opts, progress)
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
	if opts.Validators > opts.Nodes {
		return Preset{}, fmt.Errorf("keyring: extend: %d new validators from %d new nodes",
			opts.Validators, opts.Nodes)
	}
	existing, err := LoadPreset(opts.Out)
	if err != nil {
		return Preset{}, fmt.Errorf("keyring: extend: %w", err)
	}
	return generate(existing, opts, progress)
}

func generate(existing Preset, opts GenerateOpts, progress func(string)) (Preset, error) {
	if opts.Nodes < 1 {
		return Preset{}, fmt.Errorf("keyring: nodes must be >= 1")
	}
	if opts.Rand == nil {
		opts.Rand = rand.Reader
	}
	if err := os.MkdirAll(opts.Out, dirPerm); err != nil {
		return Preset{}, err
	}
	// The shared password file is what a node unlocks with at launch
	// (--password); the keystores below use the same password.
	if err := os.WriteFile(filepath.Join(opts.Out, "password"), []byte(opts.Password), secretPerm); err != nil {
		return Preset{}, err
	}

	set := existing
	set.Password = opts.Password
	alloc, err := decodeAlloc(existing.Network.Alloc)
	if err != nil {
		return Preset{}, err
	}

	first := len(existing.Nodes) + 1
	for i := first; i < first+opts.Nodes; i++ {
		e, err := generateEntry(i, opts)
		if err != nil {
			return Preset{}, fmt.Errorf("keyring: node %d: %w", i, err)
		}
		set.Nodes = append(set.Nodes, e)
		alloc[strings.TrimPrefix(e.Address, "0x")] = map[string]any{"balance": opts.Balance}
		if len(set.Network.Validators)-len(existing.Network.Validators) < opts.Validators {
			set.Network.Validators = append(set.Network.Validators, e.Address)
			if e.BLS != nil {
				set.Network.BLSKeys = append(set.Network.BLSKeys, e.BLS.PublicKey)
			}
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

	if err := writePreset(opts, set); err != nil {
		return Preset{}, err
	}
	return set, nil
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
// for gas, and nothing else can supply one until the blueprint declares
// accounts (worklist N10).
func writePreset(opts GenerateOpts, set Preset) error {
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
		if e.BLS != nil {
			n.BLSPublicKey, n.BLSPoP = e.BLS.PublicKey, e.BLS.PoP
		}
		f.Nodes = append(f.Nodes, n)
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(opts.Out, PresetFile), b, publicPerm)
}

// generateEntry creates node i's material and writes its directory: the
// nodekey, an encrypted v3 keystore, and the public fields as plain files for
// an operator reading the directory by hand.
func generateEntry(i int, opts GenerateOpts) (Entry, error) {
	nodeDir := filepath.Join(opts.Out, fmt.Sprintf("node%d", i))
	if err := os.MkdirAll(nodeDir, dirPerm); err != nil {
		return Entry{}, err
	}

	key, err := NewPrivateKey(opts.Rand)
	if err != nil {
		return Entry{}, err
	}
	id, err := Derive(key, opts.Derive)
	if err != nil {
		return Entry{}, err
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "nodekey"), []byte(key.Hex()), secretPerm); err != nil {
		return Entry{}, err
	}
	if err := writeKeystore(nodeDir, key, id.Address, opts.Password); err != nil {
		return Entry{}, err
	}
	public := map[string]string{"address": id.Address, "pubkey": id.PublicKey}
	if id.BLS != nil {
		public["bls_pubkey"] = id.BLS.PublicKey
	}
	for name, val := range public {
		if err := os.WriteFile(filepath.Join(nodeDir, name), []byte(val), publicPerm); err != nil {
			return Entry{}, err
		}
	}
	return Entry{Label: nodeLabel(i), Index: i, Nodekey: key, Identity: id}, nil
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
func writeKeystore(nodeDir string, key PrivateKey, address, password string) error {
	keyjson, err := keystore.Encrypt(key.Bytes(), password, keystoreScryptN, keystoreScryptP)
	if err != nil {
		return fmt.Errorf("keystore encrypt: %w", err)
	}
	dir := filepath.Join(nodeDir, "keystore")
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, keystoreFilename(address)), keyjson, secretPerm)
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
