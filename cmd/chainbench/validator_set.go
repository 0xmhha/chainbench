package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// newValidatorSetCmd generates a network's validator set — the preset key
// bundle (per-node keys, BLS, keystores, metadata) that defines the chain's
// validators. It was `keys generate`; it lives under `validator` because a
// preset is defined by its validator set, not by raw keys.
func newValidatorSetCmd() *cobra.Command {
	var (
		opts       keyring.GenerateOpts
		validators int
		basePort   int // superseded; see below
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Generate a validator set / preset key bundle (nodekeys, BLS, keystores, metadata)",
		Long: "Generates the preset key set the harness consumes (keyring.LoadPreset): per-node\n" +
			"nodekeys, their derived address + BLS public key/PoP (derived in process),\n" +
			"an encrypted keystore per node (via the accounts SDK — no node binary),\n" +
			"and a metadata.json. Use it to build validator sets larger than the committed\n" +
			"5-node preset (e.g. the n=6 quorum cases).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			// `validator set` builds a wbft validator set, which is defined by its
			// BLS keys; `keyring new` is where BLS became opt-in.
			opts.Derive = keyring.WithBLS
			// This command's zero has always meant "all of them"; pass it as
			// absent rather than as a declared zero, which now means a ring
			// that declares no validators at all.
			if validators > 0 {
				opts.Validators = &validators
			}
			meta, err := keyring.Generate(opts, func(line string) { fmt.Fprintln(out, line) })
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "wrote %d-node preset (%d validators) to %s\n", len(meta.Nodes), len(meta.Network.Validators), opts.Out)
			return nil
		},
	}
	cmd.Flags().IntVar(&opts.Nodes, "nodes", 0, "total nodes to generate")
	cmd.Flags().IntVar(&validators, "validators", 0, "how many nodes are validators (default: all)")
	cmd.Flags().StringVar(&opts.Out, "out", "", "output preset directory")
	cmd.Flags().StringVar(&opts.Password, "password", "1", "keystore password")
	// The generated metadata used to carry an enode per node, built from this
	// port and 127.0.0.1. Nothing read it: enodes are assembled at compose time
	// from the public key and the node's actual host and port, which is the only
	// place both are known. The flag is accepted so scripts keep running.
	cmd.Flags().IntVar(&basePort, "base-p2p", 30301, "deprecated: ignored, enodes are built at compose time")
	_ = cmd.Flags().MarkDeprecated("base-p2p", "no longer used — enodes are built when a network is composed")
	cmd.Flags().StringVar(&opts.Balance, "balance", "0x200000000000000000000000000000000000000000000000000000000000000", "genesis balance per node (0x-hex wei)")
	_ = cmd.MarkFlagRequired("nodes")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}
