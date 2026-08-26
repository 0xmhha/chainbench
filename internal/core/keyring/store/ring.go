package store

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// Label names one entry in a ring: "node1", "bp1", "faucet".
//
// It is a named type because it crosses package boundaries and is a map key.

// File names inside an entry's directory.
const (
	fileKey     = "private"
	fileAddress = "address"
	fileBLS     = "bls"
	filePoP     = "pop"

	entryDirPerm fs.FileMode = 0o700
)

// entryFile is one file written for an entry.
type entryFile struct {
	name    string
	content string
	perm    fs.FileMode
}

// Ring is a set of labelled keys held for the life of a command and persisted
// under a directory.
//
// It is the same idea as the rest of this package rather than a separate one:
// getting a key by name, keeping it, and handing it to a node are what a
// keyring does. It was a separate package while derivation had to be injected —
// BLS material came from running an external binary, so the registry could not
// compute it — and that constraint is gone.
type Ring struct {
	dir string

	mu      sync.Mutex
	entries map[Label]Entry
}

// NewRing returns a ring persisting entries under dir.
func NewRing(dir string) *Ring {
	return &Ring{dir: dir, entries: make(map[Label]Entry)}
}

// Dir is where the ring's entries are written.
func (r *Ring) Dir() string { return r.dir }

// Add returns the entry for label, resolving it from src if it is not already
// held. It is idempotent: a second call with the same label returns the first
// result, so re-running a command reuses the identity it created rather than
// producing a second, conflicting one.
//
// d selects how much to derive: a poa validator asks for [AccountOnly] and gets
// an entry with no BLS material, rather than one with a zero key that reads
// like a real one.
func (r *Ring) Add(ctx context.Context, label Label, src Source, d Derivation) (Entry, error) {
	if e, ok := r.Get(label); ok {
		return e, nil
	}

	// Resolving may read a file on another host, and deriving is arithmetic on
	// a curve; neither is done under the lock. Holding a mutex across network
	// I/O would serialize every other caller behind one slow host.
	key, err := src.Resolve(ctx)
	if err != nil {
		return Entry{}, fmt.Errorf("keyring: add %q: %w", label, err)
	}
	id, err := derive.Derive(key, d)
	if err != nil {
		return Entry{}, fmt.Errorf("keyring: add %q: %w", label, err)
	}
	e := Entry{Label: label, Nodekey: key, Identity: id}

	// Two callers may have raced to here. The first to take the lock wins, and
	// the loser discards its key rather than replacing an identity another
	// caller has already been handed.
	r.mu.Lock()
	defer r.mu.Unlock()
	if held, ok := r.entries[label]; ok {
		return held, nil
	}
	if err := r.write(e); err != nil {
		return Entry{}, err
	}
	r.entries[label] = e
	return e, nil
}

// AddExpecting is Add with the address the caller believes this key belongs to.
// The address is derived from the key material and compared, so a declaration
// that has drifted from its key is caught here rather than as a chain that
// registers one address in its genesis while the node signs with another.
func (r *Ring) AddExpecting(ctx context.Context, label Label, src Source, d Derivation, want string) (Entry, error) {
	e, err := r.Add(ctx, label, src, d)
	if err != nil {
		return Entry{}, err
	}
	if want != "" && !strings.EqualFold(want, e.Address) {
		return Entry{}, fmt.Errorf("keyring: %q derives address %s but %s was declared", label, e.Address, want)
	}
	return e, nil
}

// Get returns a held entry.
func (r *Ring) Get(label Label) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[label]
	return e, ok
}

// Labels returns the held labels in sorted order, so output that lists a ring
// is stable between runs.
func (r *Ring) Labels() []Label {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Sorted(maps.Keys(r.entries))
}

// Install writes the named entries' key material under dir on files, which may
// be this machine or a host reached over SSH.
//
// It is how a node gets the key it launches with. Only the private material is
// shipped: everything else about the identity derives from it, and shipping a
// derived value is how a node and its genesis come to disagree.
func (r *Ring) Install(ctx context.Context, files filestore.Store, dir string, labels []Label) error {
	for _, label := range labels {
		e, ok := r.Get(label)
		if !ok {
			return fmt.Errorf("keyring: install: no entry named %q", label)
		}
		dst := filepath.Join(dir, string(label), fileKey)
		if err := files.Write(ctx, dst, []byte(e.Nodekey.Hex()), secretPerm); err != nil {
			return fmt.Errorf("keyring: install %q: %w", label, err)
		}
	}
	return nil
}

// write persists one entry under <dir>/<label>/. The key is owner-only; the
// derived public fields are written beside it so an operator reading the
// directory can see what the key is without decoding it.
func (r *Ring) write(e Entry) error {
	if r.dir == "" {
		return nil
	}
	dir := filepath.Join(r.dir, string(e.Label))
	if err := os.MkdirAll(dir, entryDirPerm); err != nil {
		return fmt.Errorf("keyring: create %s: %w", dir, err)
	}
	files := []entryFile{
		{fileKey, e.Nodekey.Hex(), secretPerm},
		{fileAddress, e.Address, publicPerm},
	}
	if e.BLS != nil {
		files = append(files,
			entryFile{fileBLS, e.BLS.PublicKey, publicPerm},
			entryFile{filePoP, e.BLS.PoP, publicPerm},
		)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte(f.content), f.perm); err != nil {
			return fmt.Errorf("keyring: write %s/%s: %w", e.Label, f.name, err)
		}
	}
	return nil
}
