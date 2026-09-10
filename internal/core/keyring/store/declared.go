package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// Pinned are the 1-based indexes whose key a person actually named, as
	// opposed to the ones filled in with fresh entropy because the declaration
	// left them out.
	//
	// The difference decides what an existing ring means. A pinned index that
	// disagrees with the ring on disk is a contradiction worth stopping for: the
	// operator asked for one identity and the composition would silently run on
	// another. An unpinned index disagrees every time by construction — it is a
	// new random key on each call — so comparing it would fail every re-run.
	// Empty means nothing was pinned, and the existing ring is reused as before.
	Pinned []int
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
		// Reuse, but not in silence. The ring on disk wins — it is what an
		// existing genesis and the datadirs already refer to — so a key the
		// operator pinned to something else is not quietly discarded; it is
		// reported before anything is composed on top of the wrong identity.
		existing, err := PresetKeys{Path: s.Path}.Ensure(ctx, n)
		if err != nil {
			return keyring.Preset{}, err
		}
		if err := s.checkPinnedAgainst(existing); err != nil {
			return keyring.Preset{}, err
		}
		return existing, nil
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

// checkPinnedAgainst compares the identities a person pinned with the ones the
// existing ring holds, and refuses on the first disagreement.
//
// Only pinned indexes are compared: an index the declaration left out carries
// fresh entropy on every call, so comparing it would turn every re-run into a
// failure. Addresses are public, so the message names both sides — that is what
// tells the operator whether to point at a different key set or accept the ring.
// Nothing is written here; the ring on disk is untouched either way.
func (s DeclaredKeys) checkPinnedAgainst(existing keyring.Preset) error {
	if len(s.Pinned) == 0 {
		return nil
	}
	declared := make(map[int]keyring.Entry, len(s.Set.Nodes))
	for _, e := range s.Set.Nodes {
		declared[e.Index] = e
	}
	have := make(map[int]keyring.Entry, len(existing.Nodes))
	for _, e := range existing.Nodes {
		have[e.Index] = e
	}
	for _, idx := range s.Pinned {
		want, ok := declared[idx]
		if !ok {
			continue
		}
		got, ok := have[idx]
		if !ok {
			return fmt.Errorf(
				"keyring: key source: node%d declares a key, but the key set already at %s has no node%d — point at a different key set, or remove the declared key to reuse this one",
				idx, s.Path, idx)
		}
		if !strings.EqualFold(want.Address, got.Address) {
			return fmt.Errorf(
				"keyring: key source: node%d declares the key for %s, but the key set already at %s holds %s — the existing set is kept and nothing was changed; point at a different key set, or remove the declared key to reuse this one",
				idx, want.Address, s.Path, got.Address)
		}
	}
	return nil
}
