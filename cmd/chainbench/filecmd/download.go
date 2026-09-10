package filecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// newDownloadCmd copies a server file to a local path, by purpose+name or an
// explicit remote path. Flag binding + app.Download + output.
func newDownloadCmd() *cobra.Command {
	var (
		serverSet, server, workspaceConfig string
		purpose, name, remotePath, out     string
		docker                             bool
	)
	cmd := &cobra.Command{
		Use:   "download [flags]",
		Short: "Download a server file to a local path (elevates through sudo where permitted)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.Download(cmd.Context(), surface.Deps(cmd), app.DownloadIn{
				Target: app.TransferServer{
					ServerSet: serverSet, Server: server, Docker: docker,
					WorkspaceConfigPath: workspaceConfig,
				},
				Purpose: purpose, Name: name, RemotePath: remotePath, Out: out,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "downloaded to %s\n", res.Out)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverSet, "server-set", "", "server-set file: which servers exist and how to reach them")
	cmd.Flags().StringVar(&server, "server", "", "server to download from, by name from the server set")
	cmd.Flags().StringVar(&workspaceConfig, "workspace-config", "", "environment file owning the target dataRoot and its purpose directories (required)")
	cmd.Flags().StringVar(&purpose, "purpose", "", "source folder: bin | genesis | configs | keystore | keyrings (with --name)")
	cmd.Flags().StringVar(&name, "name", "", "file name within the purpose folder")
	cmd.Flags().StringVar(&remotePath, "remote", "", "explicit absolute source path on the target (alternative to --purpose)")
	cmd.Flags().StringVar(&out, "out", "", "local path to write (required; written 0600)")
	cmd.Flags().BoolVar(&docker, "docker", false, "servers are local docker containers: translate dials via the localmap next to the server set")
	return cmd
}
