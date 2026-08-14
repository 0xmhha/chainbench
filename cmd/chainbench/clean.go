package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/engine"
)

func newCleanCmd() *cobra.Command {
	var (
		dataDir      string
		artifactRoot string
		olderThan    string
		keepLast     int
	)
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove a launched network's data dir, or GC old session artifacts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if artifactRoot != "" {
				return cleanSessions(cmd, artifactRoot, olderThan, keepLast)
			}
			if dataDir == "" {
				return fmt.Errorf("--data-dir (or --artifact-root for session GC) is required")
			}
			// Guard: only remove a directory that looks like a chainbench data
			// dir (has a nodeset.json or genesis.json), so a mistaken path does
			// not delete something unrelated.
			if !isChainbenchDataDir(dataDir) {
				return fmt.Errorf("%q does not look like a chainbench data dir (no nodeset.json/genesis.json); refusing to remove", dataDir)
			}
			out := cmd.OutOrStdout()
			// Stop any running nodes first (best-effort).
			if ns, err := session.LoadLocalNodeSet(dataDir); err == nil {
				stopped, errs := engine.StopNodeSet(cmd.Context(), driver.NewLocalDriver(), ns)
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
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", "", "session artifact root to garbage-collect")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "GC sessions older than this age (e.g. 7d, 12h)")
	cmd.Flags().IntVar(&keepLast, "keep-last", 0, "GC keeps the newest N sessions")
	return cmd
}

// cleanSessions garbage-collects completed session directories under root. Only
// sessions with a session.json are considered, so an in-progress run (which has
// not written its verdict yet) is preserved (F16-O4). --keep-last protects the
// newest N; --older-than removes sessions whose verdict is older than the age.
func cleanSessions(cmd *cobra.Command, root, olderThan string, keepLast int) error {
	if olderThan == "" && keepLast <= 0 {
		return fmt.Errorf("clean: with --artifact-root, provide --older-than and/or --keep-last")
	}
	ids, err := session.List(root)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}

	// --keep-last protects the newest N (List is oldest-first).
	protected := map[string]bool{}
	if keepLast > 0 {
		start := len(ids) - keepLast
		if start < 0 {
			start = 0
		}
		for _, id := range ids[start:] {
			protected[id] = true
		}
	}

	var cutoff time.Time
	haveCutoff := olderThan != ""
	if haveCutoff {
		age, perr := parseAge(olderThan)
		if perr != nil {
			return fmt.Errorf("clean: bad --older-than %q: %w", olderThan, perr)
		}
		cutoff = time.Now().Add(-age)
	}

	out := cmd.OutOrStdout()
	removed := 0
	for _, id := range ids {
		if protected[id] {
			continue
		}
		if haveCutoff {
			fi, statErr := os.Stat(session.SessionFilePath(root, id))
			if statErr != nil || fi.ModTime().After(cutoff) {
				continue // unreadable or newer than the cutoff: keep
			}
		}
		if err := os.RemoveAll(session.SessionDir(root, id)); err != nil {
			return fmt.Errorf("clean: remove %s: %w", id, err)
		}
		fmt.Fprintf(out, "removed session %s\n", id)
		removed++
	}
	fmt.Fprintf(out, "removed %d session(s)\n", removed)
	return nil
}

// parseAge parses a duration, additionally accepting day (Nd) and week (Nw)
// suffixes that time.ParseDuration does not.
func parseAge(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasSuffix(s, "d"):
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	case strings.HasSuffix(s, "w"):
		n, err := strconv.Atoi(strings.TrimSuffix(s, "w"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
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
