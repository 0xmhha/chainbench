package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// DeclaredKeys writes down a key set the caller already holds.
//
// It is the third source alongside PresetKeys and GeneratedKeys, and it is what
// makes a preset optional rather than merely described as optional. A network
// declaration that carries its own nodekeys derives them into a keyring.Preset
// (blueprint.PresetFrom) and hands it here; the composition then reads keys the
// one way it already does, so nothing downstream learns that a second origin
// exists.
//
// The set is materialised because the rest of the system reads keys from disk:
// a node needs a keystore to unlock, a genesis source reads the ring directory,
// and provision ships identities from it. Holding the ring only in memory would
// mean teaching each of those a second way to be given keys, which is the
// divergence this whole track exists to end.
type DeclaredKeys struct {
	// Path is the directory the set is written to.
	Path string
	// Set is the already-derived ring. Its entries are written verbatim: this
	// source derives nothing, so what the document declared is what lands.
	Set keyring.Preset
	// Password encrypts the keystores written for the set; empty uses the
	// default a generated ring uses.
	Password string
	// Files is where the ring is written; nil is this machine's filesystem.
	Files filestore.Store
}

// Dir is the directory the set lives in.
func (s DeclaredKeys) Dir() string { return s.Path }

// Describe names the source for artifacts and error messages.
func (s DeclaredKeys) Describe() string {
	return fmt.Sprintf("declared:%s (%d identities)", s.Path, len(s.Set.Nodes))
}

// Ensure writes the declared set, or loads what is already there.
//
// An existing directory is loaded rather than overwritten, the way every source
// on this boundary behaves: re-running a command must reuse the identities a
// genesis and a datadir already refer to, and the keys behind them cannot be
// recovered once replaced.
func (s DeclaredKeys) Ensure(ctx context.Context, n int) (keyring.Preset, error) {
	if _, err := os.Stat(filepath.Join(s.Path, PresetFile)); err == nil {
		return PresetKeys{Path: s.Path}.Ensure(ctx, n)
	}
	if err := ctx.Err(); err != nil {
		return keyring.Preset{}, err
	}
	if len(s.Set.Nodes) < n {
		return keyring.Preset{}, fmt.Errorf("keyring: key source: the declaration carries %d identities, the network needs %d", len(s.Set.Nodes), n)
	}

	// ImportRing is the one writer for "a ring I already hold". Laying the
	// files out here instead would be a second answer to what a ring directory
	// contains, and the first attempt at exactly that missed the shared
	// password file — every node then died at launch with "failed to read
	// password file", after a composition that reported success at every step.
	//
	// It also verifies each entry re-derives the identity the index claims,
	// which a declaration wants more than a generated ring does: these keys
	// were typed by a person.
	set := s.Set
	set.Password = orDefault(s.Password, defaultGeneratedPassword)
	if _, err := ImportRing(ctx, s.Files, s.Path, set, set.Password); err != nil {
		return keyring.Preset{}, fmt.Errorf("keyring: key source: %w", err)
	}
	// Read back rather than return what was written. A set that cannot be
	// loaded again is a set the composition cannot use, and finding that out
	// here names the file instead of failing three steps later.
	return PresetKeys{Path: s.Path}.Ensure(ctx, n)
}
