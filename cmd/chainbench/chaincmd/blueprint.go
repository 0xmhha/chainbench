package chaincmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newBlueprintCmd writes out the network a key set would compose, as a document
// an operator can read and edit. Rendering and file writing only — the
// generation goes through app.BlueprintFromPreset.
func newBlueprintCmd() *cobra.Command {
	var keysDir, chain, manifest, binary, peering, out string
	var producers, endpoints int
	cmd := &cobra.Command{
		Use:   "blueprint",
		Short: "Write the network a key set would compose, as an editable declaration",
		Long: "Turn a key set into a network declaration.\n\n" +
			"A composition used to go from a key set straight to a network, with nothing\n" +
			"inspectable in between. This writes the middle out: a document that says what\n" +
			"will be launched, which you can read, diff, edit and commit, and then compose\n" +
			"with `chain up --blueprint`.\n\n" +
			"The keys are referenced by path, never copied into the document.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if keysDir == "" {
				return fmt.Errorf("--from-preset is required: name the key set to describe")
			}
			res, err := app.BlueprintFromPreset(cmd.Context(), deps(cmd), app.BlueprintFromPresetIn{
				KeysDir: keysDir, Chain: chain, Manifest: manifest,
				Producers: producers, Endpoints: endpoints, Binary: binary, Peering: peering,
			})
			if err != nil {
				return err
			}
			if out == "" {
				_, err := cmd.OutOrStdout().Write(res.YAML)
				return err
			}
			// Refuse to replace a document someone may have edited. Overwriting
			// is what makes a generator unsafe to re-run, and the whole point of
			// writing the middle out is that it can be edited by hand.
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("%s already exists — remove it or write elsewhere", out)
			}
			if err := os.WriteFile(out, res.YAML, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "wrote %s: %d node(s) from %s\n", out, res.Nodes, keysDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&keysDir, "from-preset", "", "key set to describe (required)")
	cmd.Flags().StringVar(&chain, "chain", "", "chain id to record (stablenet|wbft|wemix)")
	cmd.Flags().StringVar(&manifest, "manifest", "", "external chain manifest to record instead of --chain")
	cmd.Flags().IntVar(&producers, "bp", 0, "block producers (default: every identity the set declares as a validator)")
	cmd.Flags().IntVar(&endpoints, "en", 0, "RPC endpoints")
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path to record")
	cmd.Flags().StringVar(&peering, "peering", "", "peer graph to record: mesh | proxied")
	cmd.Flags().StringVar(&out, "out", "", "write to this file instead of stdout (refuses to overwrite)")
	return cmd
}
