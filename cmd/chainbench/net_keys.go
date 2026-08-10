package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/netcompose"
)

// newNetKeysCmd resolves the preset key set (node identities + validator set)
// and records it on the workspace.
func newNetKeysCmd() *cobra.Command {
	var dataDir, keysDir string
	var validators int
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Resolve the preset key set (node identities + validator set)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := openWorkspace(dataDir)
			if err != nil {
				return err
			}
			res, err := ws.Keys(netcompose.KeysOpts{KeysDir: keysDir, Validators: validators})
			if err != nil {
				return err
			}
			if err := ws.Save(); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "keys: %d node identities, %d validator(s) from %s\n", res.Nodes, res.Validators, res.KeysDir)
			for i, a := range res.Addresses {
				fmt.Fprintf(out, "  validator%d %s\n", i+1, a)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "local workspace directory")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "preset key directory")
	cmd.Flags().IntVar(&validators, "validators", 0, "active validator count (0 = whole preset)")
	return cmd
}
