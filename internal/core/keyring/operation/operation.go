package operation

import (
	"context"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The verbs a key set takes: new, add, list, show, save.
//
// A set is reached through an Opener the caller injects, so the same verbs work
// on a local directory and on a server's data plane without knowing which.
//
// Importing material, and resolving a reference to one key, are in import.go.

type Opener interface {
	OpenPath(path string) (*resource.Access, error)
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

// KeySetEnv is the store's — re-stated here only until the surfaces call the
// store directly (worklist V3.3).
const KeySetEnv = store.KeySetEnv

// DefaultKeySetDir is the store's answer for an unnamed key set. It is a
// function rather than a constant because the promised location is under the
// operator's home, which is knowledge and not a literal.
func DefaultKeySetDir() (string, error) { return store.DefaultKeySetDir() }

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
	// Listing reports identities and needs no secret; --verify re-derives each
	// identity from its key and cannot be done without one.
	dir, source, set, err := openSetWithKeys(ctx, in.Ring, d, in.Verify)
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

type passwordFunc func() (string, error)

func (f passwordFunc) Password() (string, error) { return f() }

// openSet resolves and loads a key set, naming the source in the error so that a
// missing default key set is not a mystery.
func openSet(ctx context.Context, ref SetRef, d Deps) (dir, source string, set keyring.Preset, err error) {
	return openSetWithKeys(ctx, ref, d, false)
}

// openSetWithKeys is openSet with the disclosure made a parameter. withKeys=false
// reads identities only, which is what every question about a ring needs bar
// two; verification (re-derives from the key) and export (discloses it on
// purpose) ask for true.
//
// It decides what TRAVELS, not merely what the caller holds. The ring index
// carries no private key any more, so the identity read moves none; asking for
// keys reads node<N>/nodekey per entry, which is N round trips on a remote ring
// and is the reason the two callers that need them are the only ones that ask.
func openSetWithKeys(ctx context.Context, ref SetRef, d Deps, withKeys bool) (dir, source string, set keyring.Preset, err error) {
	files, dir, source, err := ref.open(d)
	if err != nil {
		return displaySet(ref, dir), source, keyring.Preset{}, err
	}
	if withKeys {
		set, err = store.LoadPresetWithKeysAt(ctx, files, dir)
	} else {
		set, err = store.LoadPresetAt(ctx, files, dir)
	}
	dir = displaySet(ref, dir)
	if err != nil {
		return dir, source, keyring.Preset{}, fmt.Errorf("keyring %s (%s): %w", dir, source, err)
	}
	return dir, source, set, nil
}

// findEntry looks up an identity by label, listing what the key set holds when the
// name is not one of them.
func findEntry(set keyring.Preset, label string) (keyring.Entry, error) {
	for _, e := range set.Nodes {
		if string(e.Label) == label {
			return e, nil
		}
	}
	have := make([]string, 0, len(set.Nodes))
	for _, e := range set.Nodes {
		have = append(have, string(e.Label))
	}
	return keyring.Entry{}, fmt.Errorf("no identity named %q (have: %v)", label, have)
}

// validatorSet indexes the key set's declared validators by lowercase address.
func validatorSet(set keyring.Preset) map[string]bool {
	out := make(map[string]bool, len(set.Network.Validators))
	for _, a := range set.Network.Validators {
		out[lower(a)] = true
	}
	return out
}

// setOut renders a whole key set.
func setOut(dir, source string, set keyring.Preset) SetOut {
	vals := validatorSet(set)
	out := SetOut{Dir: dir, Source: source, Validators: len(set.Network.Validators)}
	for _, e := range set.Nodes {
		out.Entries = append(out.Entries, entryOut(e, vals))
	}
	return out
}

// entryOut renders one identity without its secret.
func entryOut(e keyring.Entry, validators map[string]bool) EntryOut {
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

// SaveKeyIn says where a resolved key should be written and in what form.
type SaveKeyIn struct {
	// Dir and Name locate the file; Format is "keystore" (encrypted, needs a
	// password) or "file" (0x-hex, 0600, for a throwaway local ring).
	Dir    string
	Name   string
	Format string
	// Password guards a keystore. It is a seam so that a surface can prompt, or
	// read a file, only when the key is actually written.
	Password func() (string, error)
}

// SaveKey writes a key and returns the file it wrote.
//
// The format names are part of the vocabulary an operator types, so reading
// them belongs here rather than in each surface: a spelling accepted by the CLI
// and rejected by a tool would be the same feature disagreeing with itself.
func SaveKey(_ context.Context, _ Deps, key derive.PrivateKey, in SaveKeyIn) (string, error) {
	var backend store.Backend
	switch in.Format {
	case "", "keystore":
		if in.Password == nil {
			return "", fmt.Errorf("keyring: keystore storage needs a password (--password / --password-file / --password-once)")
		}
		backend = store.KeystoreBackend{}
	case "file":
		backend = store.RawFileBackend{}
	default:
		return "", fmt.Errorf("keyring: unknown storage format %q (keystore or file)", in.Format)
	}
	var pw keyring.PasswordSource
	if in.Password != nil {
		pw = passwordFunc(in.Password)
	}
	return backend.Save(in.Dir, in.Name, key, pw)
}

// GenerateKey makes a fresh key.
//
// It is a use case rather than a surface's call to a random source because
// where entropy comes from is the keyring's business, and because a surface
// that reaches for it directly is one that could reach for a different one.
func GenerateKey(ctx context.Context, _ Deps) (derive.PrivateKey, error) {
	return keyring.RandomSource{}.Resolve(ctx)
}
