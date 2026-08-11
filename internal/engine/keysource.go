package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/keyreg"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/keygen"
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
	Preset keys.Preset
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
	p, err := keys.LoadPreset(s.Path)
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
// BLS public keys and proofs-of-possession cannot be derived in Go here — they
// come from the external bootnode binary (design §3.5) — so Bootnode is
// required and a missing binary is a clear error rather than a key set that is
// silently short of the material a wbft-family genesis needs.
//
// Note the resulting genesis extra-data: keygen writes a zero placeholder, so a
// chain whose consensus reads the validator set out of extra-data will not
// accept a generated set until that RLP is computed. Use this source for
// endpoint-heavy or non-wbft topologies, or supply the extra-data by other
// means; PresetKeySource remains the path proven against a live chain.
type GeneratedKeySource struct {
	// Path is the directory the generated set is written to.
	Path string
	// Bootnode is the external bootnode binary used to derive BLS material.
	Bootnode string
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
	if s.Bootnode == "" {
		return KeySet{}, fmt.Errorf("engine: key source: generating a key set needs a bootnode binary (BLS derivation)")
	}
	if err := ctx.Err(); err != nil {
		return KeySet{}, err
	}
	opts := keygen.PresetOpts{
		Nodes:      n,
		Validators: s.Validators,
		Bootnode:   s.Bootnode,
		Out:        s.Path,
		Password:   orDefault(s.Password, defaultGeneratedPassword),
		Balance:    orDefault(s.Balance, defaultGeneratedBalance),
	}
	if _, err := keygen.GeneratePreset(opts, nil); err != nil {
		return KeySet{}, fmt.Errorf("engine: key source: generate: %w", err)
	}
	return PresetKeySource{Path: s.Path}.Ensure(ctx, n)
}

// orDefault returns v, or def when v is empty.
func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// RegisterIdentities records a key set's node identities in the session key
// registry under the names "node1".."nodeN".
//
// Registration is not bookkeeping: it re-derives each address from the private
// key the node will actually run with and fails when that disagrees with the
// address the metadata declares. A key set whose declared identity has drifted
// from its key material produces a chain whose genesis registers one address
// while the node signs with another — a failure that otherwise surfaces much
// later as an unexplained consensus stall.
func RegisterIdentities(ctx context.Context, reg keyreg.Registry, ks KeySet, n int) error {
	if reg == nil {
		return nil
	}
	for i := 1; i <= n; i++ {
		nk, ok := ks.Preset.Node(i)
		if !ok {
			return fmt.Errorf("engine: key set %s has no identity for node%d", ks.Dir, i)
		}
		name := fmt.Sprintf("node%d", i)
		if _, err := reg.Ensure(ctx, name, keyreg.Literal, nk.Nodekey, keyreg.EnsureOpts{
			ExpectAddress: nk.Address,
		}); err != nil {
			return fmt.Errorf("engine: register identity %s: %w", name, err)
		}
	}
	return nil
}
