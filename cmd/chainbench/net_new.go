package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/netcompose"
)

// newNetNewCmd initializes a composition workspace: the target chain and where
// its data plane lives (local, or a remote SSH host).
func newNetNewCmd() *cobra.Command {
	var dataDir, chain, binary, keysDir string
	var tf targetFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Initialize a composition workspace for a chain (and its target)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := openWorkspace(dataDir)
			if err != nil {
				return err
			}
			detail, err := ws.New(netcompose.NewOpts{Chain: chain, Binary: binary, KeysDir: keysDir, Target: tf.spec()})
			if err != nil {
				return err
			}
			if err := ws.Save(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), detail)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "local workspace directory (keep it short: node IPC sockets have a 104-char limit)")
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (may also be set at start)")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "key set the network composes from (inspect/manage it with `account`)")
	tf.bind(cmd)
	return cmd
}
