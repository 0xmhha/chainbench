package keyringcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring/operation"
)

// newKeyringImportCmd brings a key that already exists into a key set.
//
// The key may be here or on another host; --from says where with one path
// syntax, so importing from a server is not a different command. Prefer
// srv://<server>/path, which keeps the host address in the gitignored server set
// instead of the command line.
func newKeyringImportCmd() *cobra.Command {
	var (
		ring  ringFlags
		label labelFlag
		bls   blsFlag
		jsonF jsonFlag
		in    operation.ImportIn
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an existing key into a key set",
		Long: "Imports a key that already exists — held as hex, or read from a file here\n" +
			"or on another host — and writes it into the ring's index under --name.\n\n" +
			"  --from /srv/keys/node1              this machine\n" +
			"  --from srv://bp1/srv/keys/node1     the server set entry \"bp1\"\n" +
			"  --from ubuntu@host:/srv/keys/node1  a host named directly\n\n" +
			"With --from-ring the unit is the whole ring, not one key: every identity\n" +
			"keeps its label, the validator declaration travels with the keys, and each\n" +
			"entry is verified against the source index before anything is written.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			in.Ring, in.WithBLS = ring.ref(), bls.on
			in.Docker = ring.docker
			if in.FromRing != "" {
				if label.name != "" || in.From != "" || in.PrivateKey != "" || in.Mnemonic != "" {
					return fmt.Errorf("--from-ring copies a whole key set, labels and all — it cannot be combined with --name or a single-key origin")
				}
				r, err := operation.ImportSet(cmd.Context(), deps(cmd), in)
				if err != nil {
					return err
				}
				if jsonF.on {
					return emitJSON(out, r)
				}
				announce(out, r)
				return renderEntries(out, r)
			}
			if err := label.require("import"); err != nil {
				return err
			}
			in.Label = label.name
			e, err := operation.Import(cmd.Context(), deps(cmd), in)
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
	ring.bind(cmd)
	label.bind(cmd)
	bls.bind(cmd)
	jsonF.bind(cmd, "the result")
	cmd.Flags().StringVar(&in.From, "from", "",
		"key file path: /local/path | srv://<server>/path | [user@]host:path | ssh://user@host:port/path")
	cmd.Flags().StringVar(&in.PrivateKey, "private-key", "", "import a key the caller already holds (0x-hex)")
	cmd.Flags().StringVar(&in.Mnemonic, "mnemonic", "", "derive the key from a BIP-39 mnemonic")
	cmd.Flags().StringVar(&in.Passphrase, "passphrase", "", "optional BIP-39 passphrase (with --mnemonic)")
	cmd.Flags().Uint32Var(&in.HDCoinType, "hd-coin-type", 0, "BIP-44 coin type for --mnemonic (default 60, Ethereum)")
	cmd.Flags().Uint32Var(&in.HDAccount, "hd-account", 0, "BIP-44 account index for --mnemonic")
	cmd.Flags().Uint32Var(&in.HDChange, "hd-change", 0, "BIP-44 change level for --mnemonic (0 external, 1 internal)")
	cmd.Flags().Uint32Var(&in.HDIndex, "hd-index", 0, "BIP-44 address index for --mnemonic")
	cmd.Flags().StringVar(&in.Password, "password", "", "password for a keystore named by --from (with --from-ring: re-encrypt with this password; default keeps the source's)")
	cmd.Flags().StringVar(&in.ExpectAddress, "expect-address", "",
		"refuse the import unless the key derives exactly this address")
	cmd.Flags().StringVar(&in.FromRing, "from-ring", "",
		"clone a whole key set instead of one key: same path syntax as --keyring-dir; labels, validator declaration, and alloc are copied, and every entry is verified against the source index")
	return cmd
}
