package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
)

// Default material for a generated key set.
const (
	defaultGeneratedPassword = "chainbench"
	// defaultGeneratedBalance pre-funds each generated account in the genesis
	// alloc. Generated identities are not in any shipped preset's alloc, so
	// without this their first transaction cannot pay for gas.
	defaultGeneratedBalance = "0x152D02C7E14AF6800000" // 100_000 ether
)

// KeySource decides where a network's node identities come from: an existing
// set on disk, or fresh material generated on first use. Implementations are
// idempotent, so re-running a command against the same directory reuses what
// is already there instead of producing a second, conflicting identity set.
//
// Both the composition steps (`net keys`) and the test engine's local run go
// through this boundary, which is why it lives with the store and not with
// either of them.
type KeySource interface {
	// Dir is where the key set lives once materialized. It is known before
	// Ensure runs so a launcher and a genesis source can be wired at
	// construction time, while materialization stays bound to a context.
	Dir() string
	// Ensure materializes (or loads) the key set for at least n nodes and
	// returns it decoded.
	Ensure(ctx context.Context, n int) (keyring.Preset, error)
	// Describe names the source for artifacts and error messages.
	Describe() string
}

// PresetKeys uses an existing on-disk key set. This is the reproducible
// default: the same directory yields the same validator set, the same genesis
// extra-data, and therefore the same chain across runs.
type PresetKeys struct {
	// Path is the preset directory (metadata.json + node<i>/ + password).
	Path string
}

// Dir is the preset directory.
func (s PresetKeys) Dir() string { return s.Path }

// Describe names the source.
func (s PresetKeys) Describe() string { return "preset:" + s.Path }

// Ensure loads the preset and checks it covers n nodes.
func (s PresetKeys) Ensure(_ context.Context, n int) (keyring.Preset, error) {
	p, err := LoadPreset(s.Path)
	if err != nil {
		return keyring.Preset{}, fmt.Errorf("keyring: key source: %w", err)
	}
	if len(p.Nodes) < n {
		return keyring.Preset{}, fmt.Errorf("keyring: key source: preset %s has %d node identities, need %d",
			s.Path, len(p.Nodes), n)
	}
	return p, nil
}

// GeneratedKeys generates a fresh random key set into Path on first use.
//
// Every identity — address, devp2p public key, BLS public key and
// proof-of-possession — is derived in process (derive.Derive), so generating a
// key set needs no chain binary.
//
// The genesis extra-data is not stored with the set: the wbft family derives it
// from the validator set at genesis time, so a generated set cannot carry one
// that contradicts its own validators. PresetKeys remains the path proven
// against a live chain.
type GeneratedKeys struct {
	// Path is the directory the generated set is written to.
	Path string
	// Password encrypts the generated keystores; empty uses a default.
	Password string
	// Balance pre-funds each generated account in the genesis alloc (0x-hex
	// wei); empty uses a default.
	Balance string
	// Validators is how many of the generated nodes join the validator set;
	// <=0 means all of them.
	Validators int
}

// Dir is the directory the set is generated into.
func (s GeneratedKeys) Dir() string { return s.Path }

// Describe names the source.
func (s GeneratedKeys) Describe() string { return "generated:" + s.Path }

// Ensure generates the key set on first call and loads it on later ones, so a
// re-run keeps the identities the first run created (and therefore its genesis).
func (s GeneratedKeys) Ensure(ctx context.Context, n int) (keyring.Preset, error) {
	if _, err := os.Stat(filepath.Join(s.Path, "metadata.json")); err == nil {
		return PresetKeys{Path: s.Path}.Ensure(ctx, n)
	}
	if err := ctx.Err(); err != nil {
		return keyring.Preset{}, err
	}
	opts := GenerateOpts{
		Nodes: n,
		// A zero here has always meant "all of them" on this boundary, so it is
		// passed as absent rather than as a declared zero, which would now mean
		// a ring that declares no validators at all.
		Validators: validatorCount(s.Validators),
		Out:        s.Path,
		Derive:     derive.WithBLS,
		Password:   orDefault(s.Password, defaultGeneratedPassword),
		Balance:    orDefault(s.Balance, defaultGeneratedBalance),
	}
	if _, err := Generate(opts, nil); err != nil {
		return keyring.Preset{}, fmt.Errorf("keyring: key source: generate: %w", err)
	}
	return PresetKeys{Path: s.Path}.Ensure(ctx, n)
}

// validatorCount renders a legacy zero-means-all count as an absent one.
func validatorCount(n int) *int {
	if n <= 0 {
		return nil
	}
	return &n
}

// orDefault returns v, or def when v is empty.
func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Register records a key set's first n node identities in this ring under the
// labels "node1".."nodeN". A nil ring registers nothing.
//
// Registration is not bookkeeping: each address is re-derived from the private
// key the node will actually run with and checked against the one the metadata
// declares. A key set whose declared identity has drifted from its key material
// produces a chain whose genesis registers one address while the node signs
// with another — a failure that otherwise surfaces much later as an unexplained
// consensus stall.
func (r *KeySet) Register(ctx context.Context, set keyring.Preset, n int) error {
	if r == nil {
		return nil
	}
	for i := 1; i <= n; i++ {
		nk, ok := set.Node(i)
		if !ok {
			return fmt.Errorf("keyring: key set has no identity for node%d", i)
		}
		label := nodeLabel(i)
		// Ask for exactly what the set claims to hold: a poa identity has no BLS
		// material, and deriving some would invent a key it never had.
		d := derive.AccountOnly
		if nk.BLS != nil {
			d = derive.WithBLS
		}
		src := keyring.PrivateKeySource{Hex: nk.Nodekey.Hex()}
		if _, err := r.AddExpecting(ctx, label, src, d, nk.Address); err != nil {
			return fmt.Errorf("keyring: register identity %s: %w", label, err)
		}
	}
	return nil
}
