package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newNetNewCmd initializes a composition workspace: the target chain and where
// its data plane lives (local, or a remote SSH host). Flag binding + app.NetNew
// + output — the logic lives in the app layer, shared with the MCP tool.
func newNetNewCmd() *cobra.Command {
	var dataDir, chain, binary, keysDir, manifestPath, templatePath string
	var tf targetFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Initialize a composition workspace for a chain (and its target)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir is required")
			}
			target, err := tf.spec()
			if err != nil {
				return err
			}
			out, err := app.NetNew(cmd.Context(), app.Deps{}, app.NetNewIn{
				DataDir: dataDir, Chain: chain, Binary: binary, KeysDir: keysDir, Target: target,
				ManifestPath: manifestPath, TemplatePath: templatePath,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out.Detail)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "local workspace directory (keep it short: node IPC sockets have a 104-char limit)")
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix); ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON (project-supplied chain, on a built-in family)")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (may also be set at start)")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "key set the network composes from (inspect/manage it with `account`)")
	tf.bind(cmd)
	return cmd
}
