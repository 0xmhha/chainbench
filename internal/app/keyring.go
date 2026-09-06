package app

import (
	"context"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/operation"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The keyring verbs live in the keyring module; app wraps them thinly so that
// every surface reaches the feature through this layer (architecture-v2 §2,
// revised 2026-09-05). Thinly is the contract: these adapt a dependency set and
// forward, and decide nothing.

type (
	// RingRef names the ring a verb works on.
	RingRef = operation.SetRef
	// RingOut reports which ring a verb acted on, and what it holds.
	RingOut = operation.SetOut
	// EntryOut is one identity's public material.
	EntryOut = operation.EntryOut
	// RingCreateIn shapes keyring new/add.
	RingCreateIn = operation.CreateIn
	// RingListIn shapes keyring list.
	RingListIn = operation.ListIn
	// RingEntryIn shapes keyring show/export.
	RingEntryIn = operation.EntryIn
	// RingImportIn shapes keyring import (single key or whole ring).
	RingImportIn = operation.ImportIn
	// KeyRef names where a single key comes from: an inline private key, a
	// mnemonic with its BIP-44 path, or a file at a local or remote path.
	KeyRef = operation.KeyRef
	// PrivateKey is a resolved key, with the addresses derivable from it.
	PrivateKey = derive.PrivateKey
	// SaveKeyIn says where a key is written and in what form.
	SaveKeyIn = operation.SaveKeyIn
)

// KeySetEnv mirrors the store's default for surface help text.
const KeySetEnv = operation.KeySetEnv

// DefaultHDCoinType is the BIP-44 coin type a mnemonic derives under when the
// caller names none (60, Ethereum). Surfaces show it in help text.
const DefaultHDCoinType = keyring.DefaultCoinType

// DefaultKeySetDir is where an unnamed key set lives, for surfaces that print
// it in help text.
func DefaultKeySetDir() (string, error) { return operation.DefaultKeySetDir() }

// keyringDeps adapts this layer's dependency set to the module's.
func (d Deps) keyringDeps() operation.Deps {
	return operation.Deps{
		Env: d.Env,
		Open: func(serverSet string, docker bool) operation.Opener {
			return resource.Opener{
				ServerSet: serverSet, Docker: docker,
				Env: d.Env, Report: d.Logf,
			}
		},
	}
}

// KeyringNew creates a ring.
func KeyringNew(ctx context.Context, d Deps, in RingCreateIn) (RingOut, error) {
	return operation.New(ctx, d.keyringDeps(), in)
}

// KeyringAdd extends a ring.
func KeyringAdd(ctx context.Context, d Deps, in RingCreateIn) (RingOut, error) {
	return operation.Add(ctx, d.keyringDeps(), in)
}

// KeyringList lists a ring's identities.
func KeyringList(ctx context.Context, d Deps, in RingListIn) (RingOut, error) {
	return operation.List(ctx, d.keyringDeps(), in)
}

// KeyringShow shows one identity's public material.
func KeyringShow(ctx context.Context, d Deps, in RingEntryIn) (EntryOut, error) {
	return operation.Show(ctx, d.keyringDeps(), in)
}

// KeyringExport reveals one identity's private key.
func KeyringExport(ctx context.Context, d Deps, in RingEntryIn) (EntryOut, error) {
	return operation.Export(ctx, d.keyringDeps(), in)
}

// KeyringImport writes an existing key into a ring's index.
func KeyringImport(ctx context.Context, d Deps, in RingImportIn) (EntryOut, error) {
	return operation.Import(ctx, d.keyringDeps(), in)
}

// KeyringImportRing clones a whole ring.
func KeyringImportRing(ctx context.Context, d Deps, in RingImportIn) (RingOut, error) {
	return operation.ImportSet(ctx, d.keyringDeps(), in)
}

// ResolveKey reads a key reference and returns the key it names.
//
// It is one entry point because a key reference has one meaning. Two surfaces
// each reading the flags their own way is how "exactly one origin" comes to be
// enforced in one of them and not the other.
func ResolveKey(ctx context.Context, d Deps, ref KeyRef) (PrivateKey, error) {
	return operation.ResolveKey(ctx, d.keyringDeps(), ref)
}

// SaveKey writes a resolved key where the caller asked, in the form it asked.
func SaveKey(ctx context.Context, d Deps, key PrivateKey, in SaveKeyIn) (string, error) {
	return operation.SaveKey(ctx, d.keyringDeps(), key, in)
}

// GenerateKey makes a fresh key.
func GenerateKey(ctx context.Context, d Deps) (PrivateKey, error) {
	return operation.GenerateKey(ctx, d.keyringDeps())
}
