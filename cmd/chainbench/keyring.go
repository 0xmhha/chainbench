package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// DefaultKeyringDir is the ring a command uses when none is named, and
// KeyringEnv overrides it. A ring is just a directory, so keys/preset is not a
// special thing — it is one ring that happens to be committed.
const (
	DefaultKeyringDir = "keys/default"
	KeyringEnv        = "CHAINBENCH_KEYRING"
)

// newKeyringCmd is the keyring group: create, inspect, and move key material.
//
// It replaces three groups that split the same material by what it was going to
// be used for — `keys` for raw keypairs, `account` for on-chain identity,
// `validator` for consensus identity. They are the same key: on the wbft family
// a node's address and its BLS material derive from one secret. Splitting by
// role meant three ways to make a key and three shapes to read one back.
func newKeyringCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "keyring",
		Short: "Create, inspect, and move key material",
		Long: "A keyring is a directory of node identities. Every command names one with\n" +
			"--keyring (or " + KeyringEnv + "), and reports which one it used, so the\n" +
			"path a key came from is never a guess.\n\n" +
			"Identities are derived in process: no chain binary has to be built or on\n" +
			"PATH to make a ring.",
	}
	c.AddCommand(
		newKeyringNewCmd(),
		newKeyringAddCmd(),
		newKeyringListCmd(),
		newKeyringShowCmd(),
		newKeyringImportCmd(),
		newKeyringExportCmd(),
	)
	return c
}

// ringFlags names the ring a command works on.
type ringFlags struct {
	dir string
}

func (f *ringFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.dir, "keyring", "",
		"ring directory (default "+DefaultKeyringDir+", or "+KeyringEnv+")")
}

// resolve returns the ring directory and where that choice came from. The
// source is reported rather than assumed: a command that silently fell back to
// a default ring is how an operator ends up inspecting one ring and launching
// from another.
func (f *ringFlags) resolve(env func(string) string) (dir, source string) {
	if f.dir != "" {
		return f.dir, "--keyring"
	}
	if v := env(KeyringEnv); v != "" {
		return v, KeyringEnv
	}
	return DefaultKeyringDir, "default"
}

// open resolves the ring, loads it, and announces which one it used.
func (f *ringFlags) open(out io.Writer) (keyring.Preset, string, error) {
	dir, source := f.resolve(os.Getenv)
	set, err := keyring.LoadPreset(dir)
	if err != nil {
		// The source matters more than the path in an error: "keys/default is
		// missing" is confusing until you know nothing named it.
		return keyring.Preset{}, dir, fmt.Errorf("keyring %s (%s): %w", dir, source, err)
	}
	fmt.Fprintf(out, "keyring: %s (%s)\n", dir, source)
	return set, dir, nil
}

// derivationFlag is the --with-bls opt-in.
//
// BLS material is only read by the wbft family, so it is asked for rather than
// assumed; a ring for wemix that carried BLS keys would carry keys nobody uses.
// An entry generated without it has no BLS material at all, which is a
// different thing from having an empty one.
type derivationFlag struct {
	withBLS bool
}

func (f *derivationFlag) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.withBLS, "with-bls", false,
		"also derive BLS material (required for the wbft family: stablenet, wbft)")
}

func (f *derivationFlag) derivation() keyring.Derivation {
	if f.withBLS {
		return keyring.WithBLS
	}
	return keyring.AccountOnly
}

// entryView is one entry as the keyring commands report it. Secrets are absent
// unless a command sets them, so printing an entry cannot leak by default.
type entryView struct {
	Label     string `json:"label"`
	Index     int    `json:"index,omitempty"`
	Address   string `json:"address"`
	PublicKey string `json:"publicKey,omitempty"`
	BLSPubKey string `json:"blsPublicKey,omitempty"`
	BLSPoP    string `json:"blsPoP,omitempty"`
	Nodekey   string `json:"nodekey,omitempty"`
	Stored    string `json:"stored,omitempty"`
}

// viewOf renders an entry without its secret.
func viewOf(e keyring.Entry) entryView {
	v := entryView{
		Label:     string(e.Label),
		Index:     e.Index,
		Address:   e.Address,
		PublicKey: e.PublicKey,
	}
	if e.BLS != nil {
		v.BLSPubKey, v.BLSPoP = e.BLS.PublicKey, e.BLS.PoP
	}
	return v
}

// entryDir is where one entry's files live inside a ring.
func entryDir(ring string, e keyring.Entry) string {
	return filepath.Join(ring, string(e.Label))
}
