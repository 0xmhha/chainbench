package app

import (
	"context"

	keyringmod "github.com/0xmhha/chainbench/internal/keyring"
)

// The keyring verbs live in the keyring module; app wraps them thinly so MCP
// reaches every feature through this layer (architecture-v2 §2). CLI calls
// the module directly and does not pass through here.

type (
	// RingRef names the ring a verb works on.
	RingRef = keyringmod.RingRef
	// RingOut reports which ring a verb acted on, and what it holds.
	RingOut = keyringmod.RingOut
	// EntryOut is one identity's public material.
	EntryOut = keyringmod.EntryOut
	// RingCreateIn shapes keyring new/add.
	RingCreateIn = keyringmod.RingCreateIn
	// RingListIn shapes keyring list.
	RingListIn = keyringmod.RingListIn
	// RingEntryIn shapes keyring show/export.
	RingEntryIn = keyringmod.RingEntryIn
	// RingImportIn shapes keyring import (single key or whole ring).
	RingImportIn = keyringmod.RingImportIn
)

// DefaultRingDir and RingEnv mirror the store's defaults for surface help text.
const (
	DefaultRingDir = keyringmod.DefaultRingDir
	RingEnv        = keyringmod.RingEnv
)

// keyringDeps adapts this layer's dependency set to the module's.
func (d Deps) keyringDeps() keyringmod.Deps {
	return keyringmod.Deps{Env: d.Env, Report: d.Logf}
}

// KeyringNew creates a ring.
func KeyringNew(ctx context.Context, d Deps, in RingCreateIn) (RingOut, error) {
	return keyringmod.KeyringNew(ctx, d.keyringDeps(), in)
}

// KeyringAdd extends a ring.
func KeyringAdd(ctx context.Context, d Deps, in RingCreateIn) (RingOut, error) {
	return keyringmod.KeyringAdd(ctx, d.keyringDeps(), in)
}

// KeyringList lists a ring's identities.
func KeyringList(ctx context.Context, d Deps, in RingListIn) (RingOut, error) {
	return keyringmod.KeyringList(ctx, d.keyringDeps(), in)
}

// KeyringShow shows one identity's public material.
func KeyringShow(ctx context.Context, d Deps, in RingEntryIn) (EntryOut, error) {
	return keyringmod.KeyringShow(ctx, d.keyringDeps(), in)
}

// KeyringExport reveals one identity's private key.
func KeyringExport(ctx context.Context, d Deps, in RingEntryIn) (EntryOut, error) {
	return keyringmod.KeyringExport(ctx, d.keyringDeps(), in)
}

// KeyringImport writes an existing key into a ring's index.
func KeyringImport(ctx context.Context, d Deps, in RingImportIn) (EntryOut, error) {
	return keyringmod.KeyringImport(ctx, d.keyringDeps(), in)
}

// KeyringImportRing clones a whole ring.
func KeyringImportRing(ctx context.Context, d Deps, in RingImportIn) (RingOut, error) {
	return keyringmod.KeyringImportRing(ctx, d.keyringDeps(), in)
}
