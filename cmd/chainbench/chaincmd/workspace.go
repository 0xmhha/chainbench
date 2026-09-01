package chaincmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// defaultWorkspaceDir is what a composition gets when --workspace-dir is
// omitted (chainsetup.DefaultWorkspaceDir). The chosen path is printed BEFORE
// anything uses it: every later step needs it, so it must never be a guess.
//
// The clock is the process's: this is an entry-point default, not logic a
// test needs to pin.
func defaultWorkspaceDir(cmd *cobra.Command) (string, error) {
	dir, err := chainsetup.DefaultWorkspaceDir(time.Now)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "workspace: %s (--workspace-dir omitted)\n", dir)
	return dir, nil
}
