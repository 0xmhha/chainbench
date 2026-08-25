package chainsetup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// Default material for a generated key set.
const (
	defaultGeneratedPassword = "chainbench"
	// defaultGeneratedBalance pre-funds each generated account in the genesis
	// alloc. Generated identities are not in any shipped preset's alloc, so
	// without this their first transaction cannot pay for gas.
	defaultGeneratedBalance = "0x152D02C7E14AF6800000" // 100_000 ether
)

// KeySet is the node identity material one environment launches from: the
// directory the launcher installs identities out of, plus its decoded metadata.
type KeySet struct {
	// Dir holds metadata.json, the shared password file, and node<i>/ identity
	// directories (nodekey + keystore).
	Dir string
	// Preset is the decoded metadata: validator addresses, BLS keys, extra-data,
	// alloc, and the per-node devp2p identities.
	Preset keyring.Preset
}

// KeySource decides where a run's node identities come from — algorithm steps 2
// and 3: generate fresh material, or use an existing set. Implementations are
// idempotent, so re-running a command against the same directory reuses what is
// already there instead of producing a second, conflicting identity set.
type KeySource interface {
	// Dir is where the key set lives once materialized. It is known before
	// Ensure runs so the launcher and genesis source can be wired at
	// construction time, while materialization stays bound to a context.
	Dir() string
	// Ensure materializes (or loads) the key set for at least n nodes.
	Ensure(ctx context.Context, n int) (KeySet, error)
	// Describe names the source for artifacts and error messages.
	Describe() string
}

// PresetKeySource uses an existing on-disk key set. This is the reproducible
// default: the same directory yields the same validator set, the same genesis
// extra-data, and therefore the same chain across runs.
type PresetKeySource struct {
	// Path is the preset directory (metadata.json + node<i>/ + password).
	Path string
}

func (s PresetKeySource) Dir() string      { return s.Path }
func (s PresetKeySource) Describe() string { return "preset:" + s.Path }

// Ensure loads the preset and checks it covers n nodes.
func (s PresetKeySource) Ensure(_ context.Context, n int) (KeySet, error) {
	p, err := store.LoadPreset(s.Path)
	if err != nil {
		return KeySet{}, fmt.Errorf("engine: key source: %w", err)
	}
	if len(p.Nodes) < n {
		return KeySet{}, fmt.Errorf("engine: key source: preset %s has %d node identities, need %d",
			s.Path, len(p.Nodes), n)
	}
	return KeySet{Dir: s.Path, Preset: p}, nil
}

// GeneratedKeySource generates a fresh random key set into Path on first use.
//
// Every identity — address, devp2p public key, BLS public key and
// proof-of-possession — is derived in process (keyring.Derive), so generating a
// key set needs no chain binary. It used to require the go-wbft bootnode tool,
// which is what made the committed preset the only practical way to start a
// network.
//
// The genesis extra-data is not stored with the set: the wbft family derives it
// from the validator set at genesis time, so a generated set cannot carry one
// that contradicts its own validators. PresetKeySource remains the path proven
// against a live chain.
type GeneratedKeySource struct {
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

func (s GeneratedKeySource) Dir() string      { return s.Path }
func (s GeneratedKeySource) Describe() string { return "generated:" + s.Path }

// Ensure generates the key set on first call and loads it on later ones, so a
// re-run keeps the identities the first run created (and therefore its genesis).
func (s GeneratedKeySource) Ensure(ctx context.Context, n int) (KeySet, error) {
	if _, err := os.Stat(filepath.Join(s.Path, "metadata.json")); err == nil {
		return PresetKeySource{Path: s.Path}.Ensure(ctx, n)
	}
	if err := ctx.Err(); err != nil {
		return KeySet{}, err
	}
	opts := store.GenerateOpts{
		Nodes: n,
		// A zero here has always meant "all of them" on this seam, so it is
		// passed as absent rather than as a declared zero, which would now mean
		// a ring that declares no validators at all.
		Validators: validatorCount(s.Validators),
		Out:        s.Path,
		Derive:     keyring.WithBLS,
		Password:   orDefault(s.Password, defaultGeneratedPassword),
		Balance:    orDefault(s.Balance, defaultGeneratedBalance),
	}
	if _, err := store.Generate(opts, nil); err != nil {
		return KeySet{}, fmt.Errorf("engine: key source: generate: %w", err)
	}
	return PresetKeySource{Path: s.Path}.Ensure(ctx, n)
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

// RegisterIdentities records a key set's node identities in the session's
// keyring under the labels "node1".."nodeN".
//
// Registration is not bookkeeping: each address is re-derived from the private
// key the node will actually run with and checked against the one the metadata
// declares. A key set whose declared identity has drifted from its key material
// produces a chain whose genesis registers one address while the node signs
// with another — a failure that otherwise surfaces much later as an unexplained
// consensus stall.
func RegisterIdentities(ctx context.Context, ring *store.Ring, ks KeySet, n int) error {
	if ring == nil {
		return nil
	}
	for i := 1; i <= n; i++ {
		nk, ok := ks.Preset.Node(i)
		if !ok {
			return fmt.Errorf("engine: key set %s has no identity for node%d", ks.Dir, i)
		}
		label := keyring.Label(fmt.Sprintf("node%d", i))
		// Ask for exactly what the set claims to hold: a poa identity has no BLS
		// material, and deriving some would invent a key it never had.
		d := keyring.AccountOnly
		if nk.BLS != nil {
			d = keyring.WithBLS
		}
		src := keyring.PrivateKeySource{Hex: nk.Nodekey.Hex()}
		if _, err := ring.AddExpecting(ctx, label, src, d, nk.Address); err != nil {
			return fmt.Errorf("engine: register identity %s: %w", label, err)
		}
	}
	return nil
}
