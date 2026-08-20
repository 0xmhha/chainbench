package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// Permissions for an imported identity: the key is owner-only, the derived
// public fields are readable so an operator can see what a directory holds
// without decoding anything.
const (
	keyDirMode     os.FileMode = 0o700
	keyFileMode    os.FileMode = 0o600
	publicFileMode os.FileMode = 0o644
)

// importedFile is one file written for an imported identity.
type importedFile struct {
	name    string
	content string
	perm    os.FileMode
}

// newKeyringImportCmd brings a key that already exists into a ring.
//
// The key may be here or on another host; --from says where with one path
// syntax, so importing from a server is not a different command. Prefer
// srv://<server>/path, which keeps the host address in the gitignored inventory
// instead of the command line.
func newKeyringImportCmd() *cobra.Command {
	var (
		ring       ringFlags
		src        sourceFlags
		pf         passwordFlags
		derivation derivationFlag
		name       string
		jsonOut    bool
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an existing key into a ring",
		Long: "Imports a key that already exists — held as hex, derived from a mnemonic,\n" +
			"or read from a file here or on another host — and writes it into the ring\n" +
			"under --name.\n\n" +
			"  --from /srv/keys/node1              this machine\n" +
			"  --from srv://bp1/srv/keys/node1     the inventory entry \"bp1\"\n" +
			"  --from ubuntu@host:/srv/keys/node1  a host named directly\n",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			dir, source := ring.resolve(env())
			fmt.Fprintf(out, "keyring: %s (%s)\n", dir, source)

			keySource, err := src.source(pf.source())
			if err != nil {
				return err
			}
			key, err := keySource.Resolve(cmd.Context())
			if err != nil {
				return err
			}
			id, err := keyring.Derive(key, derivation.derivation())
			if err != nil {
				return err
			}
			entry := keyring.Entry{Label: keyring.Label(name), Identity: id, Nodekey: key}

			stored, err := writeImported(dir, entry)
			if err != nil {
				return err
			}
			v := viewOf(entry)
			v.Stored = stored
			if jsonOut {
				return encodeJSON(out, v)
			}
			fmt.Fprintf(out, "imported %s -> %s\n", v.Label, stored)
			fmt.Fprintf(out, "address:    %s\n", v.Address)
			if v.BLSPubKey != "" {
				fmt.Fprintf(out, "blsPubKey:  %s\n", v.BLSPubKey)
			}
			return nil
		},
	}
	ring.bind(cmd)
	src.bind(cmd)
	pf.bind(cmd)
	derivation.bind(cmd)
	cmd.Flags().StringVar(&name, "name", "", "label to store the identity under")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the result as JSON")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// writeImported writes an imported identity into the ring under its label,
// refusing to overwrite one that is already there.
//
// Silently replacing an identity is the worst outcome available here: whatever
// referenced the old one — a genesis, a datadir, a test — keeps referring to an
// address that no longer has a key behind it.
func writeImported(ring string, e keyring.Entry) (string, error) {
	dir := entryDir(ring, e)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("%s already exists in %s; remove it or choose another --name", e.Label, ring)
	}
	if err := os.MkdirAll(dir, keyDirMode); err != nil {
		return "", err
	}
	files := []importedFile{
		{"nodekey", e.Nodekey.Hex(), keyFileMode},
		{"address", e.Address, publicFileMode},
		{"pubkey", e.PublicKey, publicFileMode},
	}
	if e.BLS != nil {
		files = append(files, importedFile{"bls_pubkey", e.BLS.PublicKey, publicFileMode})
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte(f.content), f.perm); err != nil {
			return "", err
		}
	}
	return dir, nil
}
