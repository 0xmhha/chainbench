package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/state"
)

func newCleanCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Stop a launched network and remove its data directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir is required")
			}
			// Guard: only remove a directory that looks like a chainbench data
			// dir (has a nodeset.json or genesis.json), so a mistaken path does
			// not delete something unrelated.
			if !isChainbenchDataDir(dataDir) {
				return fmt.Errorf("%q does not look like a chainbench data dir (no nodeset.json/genesis.json); refusing to remove", dataDir)
			}
			out := cmd.OutOrStdout()
			// Stop any running nodes first (best-effort).
			if ns, err := state.LoadNodeSet(dataDir); err == nil {
				stopped, errs := setup.StopNodeSet(cmd.Context(), driver.NewLocalDriver(), ns)
				for _, e := range errs {
					fmt.Fprintln(cmd.ErrOrStderr(), e)
				}
				fmt.Fprintf(out, "stopped %d node(s)\n", stopped)
			}
			if err := os.RemoveAll(dataDir); err != nil {
				return fmt.Errorf("remove %s: %w", dataDir, err)
			}
			fmt.Fprintf(out, "removed %s\n", dataDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root to stop and remove")
	return cmd
}

// isChainbenchDataDir reports whether dir contains chainbench setup artifacts,
// used as a safety guard before removal.
func isChainbenchDataDir(dir string) bool {
	for _, f := range []string{"nodeset.json", "genesis.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	return false
}
