// Package filecmd mounts the file command group: copying files to and from a
// server's data plane. It is the operator's door to the transfer capability the
// composition uses internally — placing a replacement binary on a server, or
// pulling a node's log or a key back off one.
//
// It is separate from the resource group on purpose: resource is queries only,
// and these commands write (upload) and read files. File placement follows the
// group rule: one file for the group, one per verb.
package filecmd

import (
	"github.com/spf13/cobra"
)

// New builds the file command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Copy files to and from a server's data plane (upload/download)",
		Long: "Move files between this machine and a server's data plane. `upload` places\n" +
			"local files under a purpose folder (bin, genesis, configs, keystore, keyrings)\n" +
			"or an explicit path, refusing a name collision unless --force-upload. `download`\n" +
			"copies a server file back to a local path. Both elevate through sudo where the\n" +
			"server set permits it, so a root-owned key is reachable.",
	}
	cmd.AddCommand(newUploadCmd(), newDownloadCmd())
	return cmd
}
