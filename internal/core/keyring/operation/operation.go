// Package keyring is the key module's surface: the verbs a surface (CLI, or
// app on behalf of MCP) runs against a key set, composed from the model
// (core/keyring), the storage (core/keyring/store), and the netmap module for
// anything that lives on a server. CLI calls this package directly; app wraps
// it thinly for MCP.
package operation

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	model "github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/machine"
)

// Opener opens a path — a plain directory here, or the target syntax naming a
// server — into the handles an operation reads and writes through. It is the
// only thing this package needs from whatever owns machines and server sets.
type Opener interface {
	OpenPath(path string) (*machine.Access, error)
}

// OpenerFor builds the opener for one key set's location choices. The caller
// supplies it because the rules for reaching a server (which set file, whether
// the servers are local containers) belong to the module that owns machines.
type OpenerFor func(serverSet string, docker bool) Opener

// Deps is what the operations need from their caller: the environment, and
// the opener that reaches a key set wherever it lives. A nil Env reads the
// process environment — surfaces run in it, and the keyring location variable
// must work without ceremony; tests inject their own. Reporting belongs to
// the opener, which is the thing with something to report.
type Deps struct {
	Env func(string) string
	// Open builds the opener for a location. Nil means local paths only —
	// naming a server without it is an error, not a silent local write.
	Open OpenerFor
}

// opener returns the opener for these choices, or an error naming what is
// missing when the caller supplied none.
func (d Deps) opener(serverSet string, docker bool) (Opener, error) {
	if d.Open == nil {
		return nil, fmt.Errorf("keyring: this caller can only reach local key sets (no opener was supplied)")
	}
	return d.Open(serverSet, docker), nil
}

func (d Deps) env() func(string) string {
	if d.Env == nil {
		return os.Getenv
	}
	return d.Env
}

// DefaultKeySetDir and KeySetEnv are the store's — re-stated here only until the
// surfaces call the store directly (worklist V3.3).
const (
	DefaultKeySetDir = store.DefaultKeySetDir
	KeySetEnv        = store.KeySetEnv
)

// SetRef names the key set a use case works on.
type SetRef struct {
	// Dir is the key set directory; empty falls back to the environment and then
	// to DefaultKeySetDir.
	Dir string
	// ServerSet is the server-set file consulted for an srv:// source; empty uses
	// the default server-set file.
	ServerSet string
	// Docker treats the key set's server as a local docker container: the dial is
	// translated through the localmap next to the server set. The flag is the
	// power switch; a leftover mapping file alone activates nothing.
	Docker bool
}

// resolve returns the key set directory and where that choice came from; the
// answer is the store's (where a key set lives is storage knowledge).
func (r SetRef) resolve(env func(string) string) (dir, source string) {
	return store.Locate(r.Dir, env)
}

// open resolves the key set to a file store and a directory on it. A plain path
// is this machine; the target syntax (srv://<server>/path, user@host:/path,
// ssh://…) places the key set on a server through the same boundary provision uses.
// Before this, a remote-looking key set path was treated as a local directory
// NAME — a key set created "on the server" landed silently on the operator's
// machine, which is worse than a refusal.
func (r SetRef) open(d Deps) (files filestore.Store, dir, source string, err error) {
	dir, source = r.resolve(d.env())
	// The netmap module is the one dial-wiring point: server-set lookup,
	// --docker translation, and the translation report all live there, so
	// this consumer cannot diverge from any other.
	o, err := r.opener(d)
	if err != nil {
		return nil, dir, source, err
	}
	tgt, err := o.OpenPath(dir)
	if err != nil {
		return nil, dir, source, err
	}
	return tgt.Files, tgt.DataRoot, source, nil
}

// opener binds this key set's server-set and docker choices to the opener the
// caller injected.
func (r SetRef) opener(d Deps) (Opener, error) {
	return d.opener(r.ServerSet, r.Docker)
}

// SetOut reports which key set a use case acted on, and what it holds afterwards.
type SetOut struct {
	// Dir is the resolved key set directory.
	Dir string
	// Source is where that directory came from: explicit, the environment
	// variable's name, or "default".
	Source string
	// Entries are the key set's identities, public material only.
	Entries []EntryOut
	// Validators is how many identities the key set declares as validators. Zero
	// means the key set declares no validator set and a network decides.
	Validators int
}

// EntryOut is one identity as a surface reports it.
//
// The private key is absent unless a use case was asked for it explicitly, so
// listing or showing a key set cannot leak by construction.
type EntryOut struct {
	Label      string `json:"label"`
	Index      int    `json:"index,omitempty"`
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey,omitempty"`
	BLSPubKey  string `json:"blsPublicKey,omitempty"`
	BLSPoP     string `json:"blsPoP,omitempty"`
	Validator  bool   `json:"validator"`
	PrivateKey string `json:"privateKey,omitempty"`
}

// CreateIn creates a key set.
type CreateIn struct {
	Ring SetRef
	// Count is how many identities to create.
	Count int
	// Validators is how many identities join the validator set. Nil is "the
	// caller said nothing", which each verb reads its own way; a pointer to 0
	// declares none, a key set of identities and nothing else. The two cannot be
	// one value, because zero is also what an unset field looks like.
	Validators *int
	// WithBLS derives BLS material, which only the wbft family reads.
	WithBLS bool
	// Password encrypts the generated keystores.
	Password string
	// Balance pre-funds each identity in the genesis alloc (0x-hex wei).
	Balance string
}

// New creates a key set of fresh identities.
func New(ctx context.Context, d Deps, in CreateIn) (SetOut, error) {
	files, dir, source, err := in.Ring.open(d)
	if err != nil {
		return SetOut{Dir: in.Ring.Dir, Source: source}, err
	}
	opts := in.opts(dir)
	opts.Files = files
	set, err := store.GenerateAt(ctx, opts, nil)
	if err != nil {
		return SetOut{Dir: displaySet(in.Ring, dir), Source: source}, err
	}
	return setOut(displaySet(in.Ring, dir), source, set), nil
}

// Add adds identities to a key set that already exists.
func Add(ctx context.Context, d Deps, in CreateIn) (SetOut, error) {
	files, dir, source, err := in.Ring.open(d)
	if err != nil {
		return SetOut{Dir: in.Ring.Dir, Source: source}, err
	}
	opts := in.opts(dir)
	opts.Files = files
	set, err := store.ExtendAt(ctx, opts, nil)
	if err != nil {
		return SetOut{Dir: displaySet(in.Ring, dir), Source: source}, err
	}
	return setOut(displaySet(in.Ring, dir), source, set), nil
}

// displaySet is what a report calls the key set: the spelling the operator gave
// (srv://server1/path) rather than the bare on-target path it resolved to.
func displaySet(ref SetRef, resolved string) string {
	if ref.Dir != "" && ref.Dir != resolved {
		return ref.Dir
	}
	return resolved
}

// opts renders the generation options.
//
// The validator count is passed through untouched, including its absence. Each
// verb resolves an unset count its own way — creating a key set takes all of them,
// extending one takes none — so resolving it here would have to know which verb
// called and would get the other wrong. It did, once.
func (in CreateIn) opts(dir string) store.GenerateOpts {
	how := derive.AccountOnly
	if in.WithBLS {
		how = derive.WithBLS
	}
	return store.GenerateOpts{
		Nodes: in.Count, Validators: in.Validators, Out: dir,
		Password: in.Password, Balance: in.Balance, Derive: how,
	}
}

// ListIn reads a key set.
type ListIn struct {
	Ring SetRef
	// Verify re-derives every identity from its own key and fails on a
	// mismatch, which is how a key set whose records have drifted from its key
	// material is caught before a network runs on it.
	Verify bool
}

// List reports what a key set holds.
func List(ctx context.Context, d Deps, in ListIn) (SetOut, error) {
	dir, source, set, err := openSet(ctx, in.Ring, d)
	if err != nil {
		return SetOut{Dir: dir, Source: source}, err
	}
	if in.Verify {
		for _, e := range set.Nodes {
			if err := e.Verify(); err != nil {
				return SetOut{Dir: dir, Source: source}, err
			}
		}
	}
	return setOut(dir, source, set), nil
}

// EntryIn names one identity in a key set.
type EntryIn struct {
	Ring SetRef
	// Label is the identity's name, e.g. "node1" or "faucet".
	Label string
}

// Show reports one identity's public material.
func Show(ctx context.Context, d Deps, in EntryIn) (EntryOut, error) {
	_, _, set, err := openSet(ctx, in.Ring, d)
	if err != nil {
		return EntryOut{}, err
	}
	e, err := findEntry(set, in.Label)
	if err != nil {
		return EntryOut{}, err
	}
	return entryOut(e, validatorSet(set)), nil
}

// Export reports one identity including its private key.
//
// It is a separate use case from Show rather than a flag on it, so that
// disclosing a secret is a call a reader can find, and so a surface can offer
// one without offering the other.
func Export(ctx context.Context, d Deps, in EntryIn) (EntryOut, error) {
	out, err := Show(ctx, d, in)
	if err != nil {
		return EntryOut{}, err
	}
	_, _, set, err := openSet(ctx, in.Ring, d)
	if err != nil {
		return EntryOut{}, err
	}
	e, err := findEntry(set, in.Label)
	if err != nil {
		return EntryOut{}, err
	}
	out.PrivateKey = "0x" + e.Nodekey.Hex()
	return out, nil
}

// ImportIn brings a key that already exists into a key set.
type ImportIn struct {
	Ring SetRef
	// Label is the name to store the identity under.
	Label string
	// From names the key file with the single path syntax: a local path,
	// srv://<server>/path, [user@]host:path, or ssh://user@host:port/path.
	// Prefer srv://, which keeps the host address in the server set rather than
	// in a command line or an agent's transcript.
	From string
	// PrivateKey is a key the caller already holds (0x-hex), as an alternative
	// to From.
	PrivateKey string
	// Mnemonic derives the key from a BIP-39 mnemonic, as an alternative to
	// From and PrivateKey. Passphrase is the optional "25th word", and the HD
	// fields select the BIP-44 path (zero values are m/44'/60'/0'/0/0).
	Mnemonic   string
	Passphrase string
	HDCoinType uint32
	HDAccount  uint32
	HDChange   uint32
	HDIndex    uint32
	// Password decrypts a keystore named by From.
	Password string
	// WithBLS derives BLS material for the imported key.
	WithBLS bool
	// Docker treats the servers as local docker containers: the harness's own
	// dials are translated through the localmap file next to the server set.
	// The flag is the power switch — a leftover mapping file alone activates
	// nothing, and the flag without the file is an error.
	Docker bool
	// ExpectAddress, when set, is the address the imported key must derive;
	// a mismatch is refused before anything is written. It is how a caller
	// who knows what the key should be makes the transfer prove it.
	ExpectAddress string
	// FromRing imports a whole key set instead of one key: every identity with
	// its label, and the network declaration (validators, BLS set, alloc).
	// Each entry is verified against the source index before anything is
	// written. Mutually exclusive with the single-key origins and Label.
	FromRing string
}

// Import writes an existing key into a key set's index.
func Import(ctx context.Context, d Deps, in ImportIn) (EntryOut, error) {
	if in.FromRing != "" {
		return EntryOut{}, fmt.Errorf("keyring: a whole-ring import returns a key set — use ImportSet")
	}
	files, dir, _, err := in.Ring.open(d)
	if err != nil {
		return EntryOut{}, err
	}
	src, err := in.source(d, in.Ring.ServerSet)
	if err != nil {
		return EntryOut{}, err
	}
	key, err := src.Resolve(ctx)
	if err != nil {
		return EntryOut{}, err
	}
	how := derive.AccountOnly
	if in.WithBLS {
		how = derive.WithBLS
	}
	if in.ExpectAddress != "" {
		id, err := derive.Derive(key, derive.AccountOnly)
		if err != nil {
			return EntryOut{}, err
		}
		if !strings.EqualFold(id.Address, in.ExpectAddress) {
			return EntryOut{}, fmt.Errorf("keyring: the key derives %s, not the expected %s — refusing to import a different identity",
				id.Address, in.ExpectAddress)
		}
	}
	e, err := store.ImportAt(ctx, files, dir, model.Label(in.Label), key, how)
	if err != nil {
		return EntryOut{}, err
	}
	return entryOut(e, nil), nil
}

// ImportSet clones a whole key set named by in.FromRing (a local path or
// target syntax) into in.Ring: every identity with its label, and the network
// declaration. Each entry is verified against the source index before anything
// is written — the key must still derive the address, devp2p key and BLS
// material the index records — so a transfer that changed anything is refused
// whole rather than materialized broken.
func ImportSet(ctx context.Context, d Deps, in ImportIn) (SetOut, error) {
	if in.FromRing == "" {
		return SetOut{}, fmt.Errorf("keyring: import-ring needs --from-ring")
	}
	srcRef := SetRef{Dir: in.FromRing, ServerSet: in.Ring.ServerSet, Docker: in.Docker || in.Ring.Docker}
	srcFiles, srcDir, _, err := srcRef.open(d)
	if err != nil {
		return SetOut{}, err
	}
	srcSet, err := store.LoadPresetAt(ctx, srcFiles, srcDir)
	if err != nil {
		return SetOut{}, fmt.Errorf("keyring: import-ring: read source %s: %w", in.FromRing, err)
	}
	dstFiles, dstDir, source, err := in.Ring.open(d)
	if err != nil {
		return SetOut{}, err
	}
	set, err := store.ImportRing(ctx, dstFiles, dstDir, srcSet, in.Password)
	if err != nil {
		return SetOut{Dir: displaySet(in.Ring, dstDir), Source: source}, err
	}
	return setOut(displaySet(in.Ring, dstDir), source, set), nil
}

// source turns the ways of naming a key into one model.Source. Where a file
// sits is a property of its path, so a remote import is not a different kind
// of import; a mnemonic is a different origin, so it is its own input.
func (in ImportIn) source(d Deps, serverSet string) (model.Source, error) {
	given := 0
	for _, set := range []bool{in.PrivateKey != "", in.From != "", in.Mnemonic != ""} {
		if set {
			given++
		}
	}
	// An option that only qualifies an absent origin is a typo about to be
	// ignored; refusing beats silently importing something else than asked.
	if in.Mnemonic == "" && (in.Passphrase != "" || in.HDCoinType != 0 || in.HDAccount != 0 || in.HDChange != 0 || in.HDIndex != 0) {
		return nil, fmt.Errorf("keyring: --passphrase and the --hd-* options qualify --mnemonic, which was not given")
	}
	if in.From == "" && in.Password != "" {
		return nil, fmt.Errorf("keyring: --password decrypts a keystore named by --from, which was not given")
	}
	switch {
	case given > 1:
		return nil, fmt.Errorf("keyring: provide exactly one of a private key, a mnemonic, or a path")
	case in.PrivateKey != "":
		return model.PrivateKeySource{Hex: in.PrivateKey}, nil
	case in.Mnemonic != "":
		path := model.HDPath{CoinType: in.HDCoinType, Account: in.HDAccount, Change: in.HDChange, Index: in.HDIndex}
		if path.CoinType == 0 {
			path.CoinType = model.DefaultCoinType
		}
		return model.MnemonicSource{Mnemonic: in.Mnemonic, Passphrase: in.Passphrase, Path: path}, nil
	case in.From == "":
		return nil, fmt.Errorf("keyring: import needs a private key, a mnemonic, or a path")
	}

	o, err := d.opener(serverSet, in.Docker)
	if err != nil {
		return nil, err
	}
	tgt, err := o.OpenPath(in.From)
	if err != nil {
		return nil, err
	}
	var pw model.PasswordSource
	if in.Password != "" {
		pw = model.StaticPassword(in.Password)
	}
	return model.FileSource{Files: tgt.Files, Path: tgt.DataRoot, Password: pw}, nil
}

// openSet resolves and loads a key set, naming the source in the error so that a
// missing default key set is not a mystery.
func openSet(ctx context.Context, ref SetRef, d Deps) (dir, source string, set model.Preset, err error) {
	files, dir, source, err := ref.open(d)
	if err != nil {
		return displaySet(ref, dir), source, model.Preset{}, err
	}
	set, err = store.LoadPresetAt(ctx, files, dir)
	dir = displaySet(ref, dir)
	if err != nil {
		return dir, source, model.Preset{}, fmt.Errorf("keyring %s (%s): %w", dir, source, err)
	}
	return dir, source, set, nil
}

// findEntry looks up an identity by label, listing what the key set holds when the
// name is not one of them.
func findEntry(set model.Preset, label string) (model.Entry, error) {
	for _, e := range set.Nodes {
		if string(e.Label) == label {
			return e, nil
		}
	}
	have := make([]string, 0, len(set.Nodes))
	for _, e := range set.Nodes {
		have = append(have, string(e.Label))
	}
	return model.Entry{}, fmt.Errorf("no identity named %q (have: %v)", label, have)
}

// validatorSet indexes the key set's declared validators by lowercase address.
func validatorSet(set model.Preset) map[string]bool {
	out := make(map[string]bool, len(set.Network.Validators))
	for _, a := range set.Network.Validators {
		out[lower(a)] = true
	}
	return out
}

// setOut renders a whole key set.
func setOut(dir, source string, set model.Preset) SetOut {
	vals := validatorSet(set)
	out := SetOut{Dir: dir, Source: source, Validators: len(set.Network.Validators)}
	for _, e := range set.Nodes {
		out.Entries = append(out.Entries, entryOut(e, vals))
	}
	return out
}

// entryOut renders one identity without its secret.
func entryOut(e model.Entry, validators map[string]bool) EntryOut {
	out := EntryOut{
		Label:     string(e.Label),
		Index:     e.Index,
		Address:   e.Address,
		PublicKey: e.PublicKey,
		Validator: validators[lower(e.Address)],
	}
	if e.BLS != nil {
		out.BLSPubKey, out.BLSPoP = e.BLS.PublicKey, e.BLS.PoP
	}
	return out
}

// lower folds an address for comparison; a file and a derivation may disagree
// on case without disagreeing on the address.
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// env resolves the environment lookup, defaulting to the process environment.
