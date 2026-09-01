package chaincmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// newNetNewCmd initializes a composition workspace: the target chain and where
// its data plane lives (local, or a remote SSH host). Flag binding + chainsetup.NetNew
// + output — the logic lives in the app layer, shared with the MCP tool.
func newNetNewCmd() *cobra.Command {
	var dataDir, chain, binary, keysDir, manifestPath, templatePath string
	var docker bool
	var serverSet string
	var tf targetFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Initialize a composition workspace for a chain (and its target)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				var err error
				if dataDir, err = defaultWorkspaceDir(cmd); err != nil {
					return err
				}
			}
			target, err := tf.spec()
			if err != nil {
				return err
			}
			out, err := chainsetup.NetNew(cmd.Context(), deps(cmd), chainsetup.NetNewIn{
				DataDir: dataDir, Chain: chain, Binary: binary, KeysDir: keysDir, Target: target,
				ManifestPath: manifestPath, TemplatePath: templatePath, Docker: docker,
				ServerSet: serverSet,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out.Detail)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory — where the composition is set up (default: ~/.chainbench/<timestamp>/chainsetup; keep it short: node IPC sockets have a 104-char limit)")
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix); ignored with --manifest")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external chain manifest JSON (project-supplied chain, on a built-in family)")
	cmd.Flags().StringVar(&templatePath, "genesis-template", "", "path to the genesis template for --manifest")
	cmd.Flags().StringVar(&binary, "binary", "", "node binary path (may also be set at start)")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "key set the network composes from (inspect/manage it with `account`)")
	cmd.Flags().BoolVar(&docker, "docker", false,
		"servers are local docker containers: translate this tool's dials via the localmap next to the server set (addresses only — docker itself is not touched)")
	cmd.Flags().StringVar(&serverSet, "server-set", "",
		"server-set file the servers come from (recorded now; allocate may override): which servers exist and how to reach them")
	tf.bind(cmd)
	return cmd
}
