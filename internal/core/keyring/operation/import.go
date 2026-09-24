package operation

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/preset"
)

// Importing key material into a set, and resolving a reference to one key.
//
// A key can arrive as hex, as a mnemonic, as a keystore file, or by naming an
// entry of another set, and each needs a different question answered before it
// becomes an entry. Keeping them together is what stops a fifth source being
// added in whichever function the author happened to open.

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
	// The one read in this package that asks for the key, which is the whole
	// point of Export existing separately from Show.
	_, _, set, err := openSetWithKeys(ctx, in.Ring, d, true)
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
	e, err := store.ImportAt(ctx, files, dir, keyring.Label(in.Label), key, how)
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
	// With keys: importing a ring COPIES the identities, so it needs what makes
	// them identities. It is the third caller that asks for secrets, and like the
	// other two it is a call whose whole purpose is to move them.
	srcSet, err := preset.LoadKeyPresetWithKeysAt(ctx, srcFiles, srcDir)
	if err != nil {
		return SetOut{}, fmt.Errorf("keyring: import-ring: read source %s: %w", in.FromRing, err)
	}
	dstFiles, dstDir, origin, err := in.Ring.open(d)
	if err != nil {
		return SetOut{}, err
	}
	set, err := store.ImportRing(ctx, dstFiles, dstDir, srcSet, in.Password)
	if err != nil {
		return SetOut{Dir: displaySet(in.Ring, dstDir), Origin: origin}, err
	}
	return setOut(displaySet(in.Ring, dstDir), origin, set), nil
}

// source turns the ways of naming a key into one keyring.Source. Where a file
// sits is a property of its path, so a remote import is not a different kind
// of import; a mnemonic is a different origin, so it is its own input.
func (in ImportIn) source(d Deps, serverSet string) (keyring.Source, error) {
	return KeyRef{
		PrivateKey: in.PrivateKey, From: in.From, Mnemonic: in.Mnemonic,
		Passphrase: in.Passphrase, HDCoinType: in.HDCoinType, HDAccount: in.HDAccount,
		HDChange: in.HDChange, HDIndex: in.HDIndex,
		Password: staticIfSet(in.Password), ServerSet: serverSet, Docker: in.Docker,
	}.source(d)
}

// staticIfSet turns an inline password into the seam a KeyRef takes, or nil
// when none was given.
func staticIfSet(pw string) func() (string, error) {
	if pw == "" {
		return nil
	}
	return func() (string, error) { return pw, nil }
}

// KeyRef names where a single key comes from, as an operator described it: an
// inline private key, a mnemonic with its BIP-44 path, or a file named with the
// single path syntax (local, srv://<server>/path, [user@]host:path,
// ssh://user@host:port/path).
//
// It is public because two surfaces needed it and, until 2026-09-05, each had
// its own copy of this resolution — the CLI built keyring.Source values from
// its flags while this module built them from an import request, and the two
// were free to disagree about what "exactly one origin" or "a bare mnemonic"
// meant. One reading of a key reference now serves every surface.
type KeyRef struct {
	// PrivateKey is a key the caller already holds (0x-hex).
	PrivateKey string
	// Mnemonic derives the key from a BIP-39 phrase; Passphrase is the optional
	// "25th word" and the HD fields select the BIP-44 path (zero values are
	// m/44'/60'/0'/0/0).
	Mnemonic   string
	Passphrase string
	HDCoinType uint32
	HDAccount  uint32
	HDChange   uint32
	HDIndex    uint32
	// From names a key file with the single path syntax.
	From string
	// Password decrypts a keystore named by From. It is a seam rather than a
	// string so that a surface can prompt, or read a file, only if the key is
	// actually reached — asking for a password that is never used is a poor way
	// to treat an operator.
	Password func() (string, error)
	// ServerSet says which servers exist, for a srv:// path.
	ServerSet string
	// Docker treats those servers as local containers, translating dials
	// through the localmap beside the server set.
	Docker bool
}

// ResolveKey reads a key reference and returns the key it names.
func ResolveKey(ctx context.Context, d Deps, ref KeyRef) (derive.PrivateKey, error) {
	src, err := ref.source(d)
	if err != nil {
		return derive.PrivateKey{}, err
	}
	return src.Resolve(ctx)
}

// source turns the reference into the keyring source that reads it, refusing a
// reference that names no origin, more than one, or qualifies an origin it did
// not name.
func (ref KeyRef) source(d Deps) (keyring.Source, error) {
	given := 0
	for _, set := range []bool{ref.PrivateKey != "", ref.From != "", ref.Mnemonic != ""} {
		if set {
			given++
		}
	}
	// An option that only qualifies an absent origin is a typo about to be
	// ignored; refusing beats silently importing something else than asked.
	if ref.Mnemonic == "" && (ref.Passphrase != "" || ref.HDCoinType != 0 || ref.HDAccount != 0 || ref.HDChange != 0 || ref.HDIndex != 0) {
		return nil, fmt.Errorf("keyring: --passphrase and the --hd-* options qualify --mnemonic, which was not given")
	}
	if ref.From == "" && ref.Password != nil {
		return nil, fmt.Errorf("keyring: --password decrypts a keystore named by --from, which was not given")
	}
	switch {
	case given > 1:
		return nil, fmt.Errorf("keyring: provide exactly one of a private key, a mnemonic, or a path")
	case ref.PrivateKey != "":
		return keyring.PrivateKeySource{Hex: ref.PrivateKey}, nil
	case ref.Mnemonic != "":
		path := keyring.HDPath{CoinType: ref.HDCoinType, Account: ref.HDAccount, Change: ref.HDChange, Index: ref.HDIndex}
		if path.CoinType == 0 {
			path.CoinType = keyring.DefaultCoinType
		}
		return keyring.MnemonicSource{Mnemonic: ref.Mnemonic, Passphrase: ref.Passphrase, Path: path}, nil
	case ref.From == "":
		return nil, fmt.Errorf("keyring: this needs a private key, a mnemonic, or a path")
	}

	o, err := d.opener(ref.ServerSet, ref.Docker)
	if err != nil {
		return nil, err
	}
	tgt, err := o.OpenPath(ref.From)
	if err != nil {
		return nil, err
	}
	var pw keyring.PasswordSource
	if ref.Password != nil {
		pw = passwordFunc(ref.Password)
	}
	return keyring.FileSource{Files: tgt.Files, Path: tgt.DataRoot, Password: pw}, nil
}

// passwordFunc adapts the reference's seam to the keyring's password source.
