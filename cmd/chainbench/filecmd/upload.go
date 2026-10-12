package filecmd

import (
	"fmt"
	"os"
	"strconv"

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
		digest, logins, mode               string
		docker, force                      bool
	)
	cmd := &cobra.Command{
		Use:   "upload [flags] <local-file>...",
		Short: "Upload local files to a server (refuses a name collision unless --force-upload)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			perm, err := parseMode(mode)
			if err != nil {
				return err
			}
			out, err := app.Upload(cmd.Context(), surface.Deps(cmd).WithLogins(logins), app.UploadIn{
				Target: app.TransferServer{
					ServerSet: serverSet, Server: server, Docker: docker,
					WorkspaceConfigPath: workspaceConfig,
				},
				Purpose: purpose, RemotePath: remotePath,
				LocalPaths: args, Force: force, SHA256: digest, Mode: perm,
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
	cmd.Flags().StringVar(&mode, "mode", "", "octal mode of the written file, e.g. 0755 (default: the local file's mode)")
	cmd.Flags().StringVar(&digest, "sha256", "", "hex sha256 the one local file must have; checked before writing and again on the target")
	cmd.Flags().StringVar(&logins, "server-logins", "", "server-set file whose ssh blocks supply this upload's logins, apart from --server-set")
	cmd.Flags().BoolVar(&force, "force-upload", false, "overwrite a destination that already exists (default: refuse a name collision)")
	return cmd
}

// parseMode reads an octal file mode, or zero when none was given.
func parseMode(s string) (os.FileMode, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(s, 8, 32)
	if err != nil || n == 0 || n > 0o777 {
		return 0, fmt.Errorf("--mode %q: want an octal permission such as 0755", s)
	}
	return os.FileMode(n), nil
}
