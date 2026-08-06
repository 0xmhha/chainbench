package keyreg

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/driver"
)

// File and directory names/permissions for persisted keys.
const (
	filePrivate = "private"
	fileAddress = "address"
	fileBLS     = "bls"
	filePoP     = "pop"

	keyDirPerm  fs.FileMode = 0o700
	keyFilePerm fs.FileMode = 0o600
	metaPerm    fs.FileMode = 0o644
)

// Deps injects the collaborators the registry needs, so it stays decoupled from
// any chain: how to generate a random key, how to derive an address for an
// existing key, and (optionally) how to fetch a remote key.
type Deps struct {
	// Generate returns a fresh random keypair. Defaults to accounts.GenerateKey.
	Generate func() (priv []byte, addr string, err error)
	// DeriveAddress derives the address of an existing private key. Required for
	// the LocalFile and RemoteDownload sources.
	DeriveAddress func(priv []byte) (string, error)
	// FetchRemote reads a remote key file for RemoteDownload. Nil disables it.
	FetchRemote func(ctx context.Context, ref string) ([]byte, error)
}

// reg is the concrete Registry: an in-memory name->Key map persisted under a
// session keys directory.
type reg struct {
	keysDir string
	deps    Deps

	mu   sync.Mutex
	keys map[string]Key
}

// New returns a Registry that persists keys under keysDir. Generate defaults to
// accounts.GenerateKey when not supplied.
func New(keysDir string, deps Deps) Registry {
	if deps.Generate == nil {
		deps.Generate = accounts.GenerateKey
	}
	return &reg{keysDir: keysDir, deps: deps, keys: make(map[string]Key)}
}

// Ensure returns the named key, materializing it per src if absent (idempotent).
func (r *reg) Ensure(ctx context.Context, name string, src Source, ref string, opts EnsureOpts) (Key, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if k, ok := r.keys[name]; ok {
		return k, nil
	}

	priv, addr, err := r.materialize(ctx, src, ref)
	if err != nil {
		return Key{}, fmt.Errorf("keyreg: ensure %q: %w", name, err)
	}
	k := Key{Name: name, Address: addr, Private: priv}

	if opts.NeedBLS {
		if opts.BLS == nil {
			return Key{}, fmt.Errorf("keyreg: %q needs BLS but no deriver was provided", name)
		}
		k.BLS, k.PoP, err = opts.BLS.Derive(ctx, priv)
		if err != nil {
			return Key{}, fmt.Errorf("keyreg: %q derive BLS: %w", name, err)
		}
	}

	if err := r.persist(k); err != nil {
		return Key{}, err
	}
	r.keys[name] = k
	return k, nil
}

// materialize obtains the private key and address for a source.
func (r *reg) materialize(ctx context.Context, src Source, ref string) (priv []byte, addr string, err error) {
	switch src {
	case Random:
		return r.deps.Generate()
	case LocalFile:
		raw, err := os.ReadFile(ref)
		if err != nil {
			return nil, "", err
		}
		return r.fromExisting(raw)
	case RemoteDownload:
		if r.deps.FetchRemote == nil {
			return nil, "", fmt.Errorf("RemoteDownload requires a remote fetcher")
		}
		raw, err := r.deps.FetchRemote(ctx, ref)
		if err != nil {
			return nil, "", err
		}
		return r.fromExisting(raw)
	default:
		return nil, "", fmt.Errorf("unknown source %d", src)
	}
}

// fromExisting parses a hex private key and derives its address.
func (r *reg) fromExisting(raw []byte) (priv []byte, addr string, err error) {
	priv, err = decodeHexKey(raw)
	if err != nil {
		return nil, "", err
	}
	if r.deps.DeriveAddress == nil {
		return nil, "", fmt.Errorf("existing key needs a DeriveAddress function")
	}
	addr, err = r.deps.DeriveAddress(priv)
	if err != nil {
		return nil, "", err
	}
	return priv, addr, nil
}

// Get returns an already-registered key from memory.
func (r *reg) Get(name string) (Key, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[name]
	return k, ok
}

// UploadTo ships each named key's private material to remotePath/<name>/private.
func (r *reg) UploadTo(ctx context.Context, fp driver.FileProvisioner, names []string, remotePath string) error {
	for _, name := range names {
		k, ok := r.Get(name)
		if !ok {
			return fmt.Errorf("keyreg: upload: unknown key %q", name)
		}
		dst := filepath.Join(remotePath, name, filePrivate)
		if err := fp.ProvisionFile(ctx, dst, []byte(hex.EncodeToString(k.Private)), keyFilePerm); err != nil {
			return fmt.Errorf("keyreg: upload %q: %w", name, err)
		}
	}
	return nil
}

// keyFile is one file to persist for a key.
type keyFile struct {
	name    string
	content []byte
	perm    fs.FileMode
}

// persist writes the key under keysDir/<name>/, private material with 0600.
func (r *reg) persist(k Key) error {
	dir := filepath.Join(r.keysDir, k.Name)
	if err := os.MkdirAll(dir, keyDirPerm); err != nil {
		return fmt.Errorf("keyreg: create %s: %w", dir, err)
	}
	files := []keyFile{
		{filePrivate, []byte(hex.EncodeToString(k.Private)), keyFilePerm},
		{fileAddress, []byte(k.Address), metaPerm},
	}
	if len(k.BLS) > 0 {
		files = append(files,
			keyFile{fileBLS, []byte(hex.EncodeToString(k.BLS)), keyFilePerm},
			keyFile{filePoP, []byte(hex.EncodeToString(k.PoP)), keyFilePerm},
		)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), f.content, f.perm); err != nil {
			return fmt.Errorf("keyreg: write %s/%s: %w", k.Name, f.name, err)
		}
	}
	return nil
}

// decodeHexKey parses a hex-encoded private key, tolerating a 0x prefix and
// surrounding whitespace.
func decodeHexKey(raw []byte) ([]byte, error) {
	s := strings.TrimSpace(string(raw))
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("empty private key")
	}
	return b, nil
}
