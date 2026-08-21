package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

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
			entry, err := keyring.Import(dir, keyring.Label(name), key, derivation.derivation())
			if err != nil {
				return err
			}
			v := viewOf(entry)
			v.Stored = entryDir(dir, entry)
			if jsonOut {
				return encodeJSON(out, v)
			}
			fmt.Fprintf(out, "imported %s -> %s\n", v.Label, v.Stored)
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
