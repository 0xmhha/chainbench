package filecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
)

// newUploadCmd places local files on a server, under a purpose folder or an
// explicit path. Flag binding + app.Upload + output — the logic lives in the
// app layer.
func newUploadCmd() *cobra.Command {
	var (
		serverSet, server, workspaceConfig string
		purpose, remotePath                string
		docker, force                      bool
	)
	cmd := &cobra.Command{
		Use:   "upload [flags] <local-file>...",
		Short: "Upload local files to a server (refuses a name collision unless --force-upload)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := app.Upload(cmd.Context(), surface.Deps(cmd), app.UploadIn{
				Target: app.TransferServer{
					ServerSet: serverSet, Server: server, Docker: docker,
					WorkspaceConfigPath: workspaceConfig,
				},
				Purpose: purpose, RemotePath: remotePath,
				LocalPaths: args, Force: force,
			})
			for _, p := range out.Uploaded {
				fmt.Fprintf(cmd.OutOrStdout(), "uploaded %s\n", p)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&serverSet, "server-set", "", "server-set file: which servers exist and how to reach them")
	cmd.Flags().StringVar(&server, "server", "", "server to upload to, by name from the server set")
	cmd.Flags().StringVar(&workspaceConfig, "workspace-config", "", "environment file owning the target dataRoot and its purpose directories (required)")
	cmd.Flags().StringVar(&purpose, "purpose", "", "destination folder: bin | genesis | configs | keystore | keyrings (each file lands there by its name)")
	cmd.Flags().StringVar(&remotePath, "remote", "", "explicit absolute destination path on the target (for one local file; alternative to --purpose)")
	cmd.Flags().BoolVar(&docker, "docker", false, "servers are local docker containers: translate dials via the localmap next to the server set")
	cmd.Flags().BoolVar(&force, "force-upload", false, "overwrite a destination that already exists (default: refuse a name collision)")
	return cmd
}
