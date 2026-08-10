package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/keygen"
)

// newValidatorSetCmd generates a network's validator set — the preset key
// bundle (per-node keys, BLS, keystores, metadata) that defines the chain's
// validators. It was `keys generate`; it lives under `validator` because a
// preset is defined by its validator set, not by raw keys.
func newValidatorSetCmd() *cobra.Command {
	var opts keygen.PresetOpts
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Generate a validator set / preset key bundle (nodekeys, BLS, keystores, metadata)",
		Long: "Generates the preset key set the harness consumes (keys.LoadPreset): per-node\n" +
			"nodekeys, their derived address + BLS public key/PoP (via the go-wbft bootnode\n" +
			"tool), an encrypted keystore per node (via the node binary's `account import`),\n" +
			"and a metadata.json. Use it to build validator sets larger than the committed\n" +
			"5-node preset (e.g. the n=6 quorum cases).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			meta, err := keygen.GeneratePreset(opts, func(line string) { fmt.Fprintln(out, line) })
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "wrote %d-node preset (%d validators) to %s\n", len(meta.Nodes), len(meta.Validators), opts.Out)
			return nil
		},
	}
	cmd.Flags().IntVar(&opts.Nodes, "nodes", 0, "total nodes to generate")
	cmd.Flags().IntVar(&opts.Validators, "validators", 0, "how many nodes are validators (default: all)")
	cmd.Flags().StringVar(&opts.Bootnode, "bootnode", "bootnode", "path to the go-wbft bootnode tool (address+BLS derivation)")
	cmd.Flags().StringVar(&opts.Binary, "binary", "", "path to the node binary (gwemix) for keystore `account import`")
	cmd.Flags().StringVar(&opts.Out, "out", "", "output preset directory")
	cmd.Flags().StringVar(&opts.Password, "password", "1", "keystore password")
	cmd.Flags().IntVar(&opts.BasePort, "base-p2p", 30301, "base p2p port for enode URLs")
	cmd.Flags().StringVar(&opts.Balance, "balance", "0x200000000000000000000000000000000000000000000000000000000000000", "genesis balance per node (0x-hex wei)")
	_ = cmd.MarkFlagRequired("nodes")
	_ = cmd.MarkFlagRequired("binary")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}
