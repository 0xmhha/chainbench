package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newKeyringImportCmd brings a key that already exists into a ring.
//
// The key may be here or on another host; --from says where with one path
// syntax, so importing from a server is not a different command. Prefer
// srv://<server>/path, which keeps the host address in the gitignored inventory
// instead of the command line.
func newKeyringImportCmd() *cobra.Command {
	var (
		ring  ringFlags
		label labelFlag
		bls   blsFlag
		jsonF jsonFlag
		in    app.RingImportIn
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an existing key into a ring",
		Long: "Imports a key that already exists — held as hex, or read from a file here\n" +
			"or on another host — and writes it into the ring's index under --name.\n\n" +
			"  --from /srv/keys/node1              this machine\n" +
			"  --from srv://bp1/srv/keys/node1     the inventory entry \"bp1\"\n" +
			"  --from ubuntu@host:/srv/keys/node1  a host named directly\n",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			in.Ring, in.Label, in.WithBLS = ring.ref(), label.name, bls.on
			e, err := app.KeyringImport(cmd.Context(), app.Deps{}, in)
			if err != nil {
				return err
			}
			if jsonF.on {
				return emitJSON(out, e)
			}
			fmt.Fprintf(out, "imported %s\n", e.Label)
			renderEntry(out, e)
			return nil
		},
	}
	ring.bindWithInventory(cmd)
	label.bind(cmd)
	bls.bind(cmd)
	jsonF.bind(cmd, "the result")
	cmd.Flags().StringVar(&in.From, "from", "",
		"key file path: /local/path | srv://<server>/path | [user@]host:path | ssh://user@host:port/path")
	cmd.Flags().StringVar(&in.PrivateKey, "private-key", "", "import a key the caller already holds (0x-hex)")
	cmd.Flags().StringVar(&in.Password, "password", "", "password for a keystore named by --from")
	return cmd
}
