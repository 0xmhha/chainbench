package netcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// defaultWorkspaceDir is where a composition sets up when --workspace-dir is
// omitted: a fresh, timestamped directory under the operator's home, so two
// runs never collide and an operator can always find the last one.
//
//	~/.chainbench/<YYYYMMDD-HHMMSS>/chainsetup
//
// The path is deliberately short — node IPC sockets live under it and a unix
// socket path is capped at 104 characters. A <test-name> segment joins the
// scheme when the test-running surface exists to supply one (module-plan P5).
//
// The clock is the process's: this is an entry-point default, printed before
// use, not logic a test needs to pin.
func defaultWorkspaceDir(cmd *cobra.Command) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("net: --workspace-dir omitted and no home directory to default under: %w", err)
	}
	dir := filepath.Join(home, ".chainbench", time.Now().Format("20060102-150405"), "chainsetup")
	// Say where the workspace went BEFORE anything uses it: every later step
	// needs this path, so it must never be a guess.
	fmt.Fprintf(cmd.OutOrStdout(), "workspace: %s (--workspace-dir omitted)\n", dir)
	return dir, nil
}
