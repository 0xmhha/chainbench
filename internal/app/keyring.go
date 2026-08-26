package app

import (
	"context"
	"github.com/0xmhha/chainbench/internal/netmap"

	"github.com/0xmhha/chainbench/internal/core/keyring/operation"
)

// The keyring verbs live in the keyring module; app wraps them thinly so MCP
// reaches every feature through this layer (architecture-v2 §2). CLI calls
// the module directly and does not pass through here.

type (
	// RingRef names the ring a verb works on.
	RingRef = operation.RingRef
	// RingOut reports which ring a verb acted on, and what it holds.
	RingOut = operation.RingOut
	// EntryOut is one identity's public material.
	EntryOut = operation.EntryOut
	// RingCreateIn shapes keyring new/add.
	RingCreateIn = operation.RingCreateIn
	// RingListIn shapes keyring list.
	RingListIn = operation.RingListIn
	// RingEntryIn shapes keyring show/export.
	RingEntryIn = operation.RingEntryIn
	// RingImportIn shapes keyring import (single key or whole ring).
	RingImportIn = operation.RingImportIn
)

// DefaultRingDir and RingEnv mirror the store's defaults for surface help text.
const (
	DefaultRingDir = operation.DefaultRingDir
	RingEnv        = operation.RingEnv
)

// keyringDeps adapts this layer's dependency set to the module's.
func (d Deps) keyringDeps() operation.Deps {
	return operation.Deps{
		Env: d.Env,
		Open: func(serverSet string, docker bool) operation.Opener {
			return netmap.Opener{
				ServerSet: serverSet, Docker: docker,
				Env: d.Env, Report: d.Logf,
			}
		},
	}
}

// KeyringNew creates a ring.
func KeyringNew(ctx context.Context, d Deps, in RingCreateIn) (RingOut, error) {
	return operation.KeyringNew(ctx, d.keyringDeps(), in)
}

// KeyringAdd extends a ring.
func KeyringAdd(ctx context.Context, d Deps, in RingCreateIn) (RingOut, error) {
	return operation.KeyringAdd(ctx, d.keyringDeps(), in)
}

// KeyringList lists a ring's identities.
func KeyringList(ctx context.Context, d Deps, in RingListIn) (RingOut, error) {
	return operation.KeyringList(ctx, d.keyringDeps(), in)
}

// KeyringShow shows one identity's public material.
func KeyringShow(ctx context.Context, d Deps, in RingEntryIn) (EntryOut, error) {
	return operation.KeyringShow(ctx, d.keyringDeps(), in)
}

// KeyringExport reveals one identity's private key.
func KeyringExport(ctx context.Context, d Deps, in RingEntryIn) (EntryOut, error) {
	return operation.KeyringExport(ctx, d.keyringDeps(), in)
}

// KeyringImport writes an existing key into a ring's index.
func KeyringImport(ctx context.Context, d Deps, in RingImportIn) (EntryOut, error) {
	return operation.KeyringImport(ctx, d.keyringDeps(), in)
}

// KeyringImportRing clones a whole ring.
func KeyringImportRing(ctx context.Context, d Deps, in RingImportIn) (RingOut, error) {
	return operation.KeyringImportRing(ctx, d.keyringDeps(), in)
}
