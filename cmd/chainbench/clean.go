package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
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
			res, err := app.NetworkRemove(cmd.Context(), app.Deps{}, app.NetworkRemoveIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			printStopFailures(cmd, res.Failed)
			fmt.Fprintf(out, "stopped %d node(s)\n", res.Stopped)
			fmt.Fprintf(out, "removed %s\n", res.Removed)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data root to stop and remove")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", "", "session artifact root to garbage-collect")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "GC sessions older than this age (e.g. 7d, 12h)")
	cmd.Flags().IntVar(&keepLast, "keep-last", 0, "GC keeps the newest N sessions")
	return cmd
}

// cleanSessions garbage-collects completed session directories under root,
// parsing the age flag into the duration the use case takes.
func cleanSessions(cmd *cobra.Command, root, olderThan string, keepLast int) error {
	var age time.Duration
	if olderThan != "" {
		parsed, err := parseAge(olderThan)
		if err != nil {
			return fmt.Errorf("clean: bad --older-than %q: %w", olderThan, err)
		}
		age = parsed
	}
	res, err := app.GCSessions(cmd.Context(), app.Deps{}, app.GCSessionsIn{
		Root: root, OlderThan: age, KeepLast: keepLast,
	})
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, id := range res.Removed {
		fmt.Fprintf(out, "removed session %s\n", id)
	}
	fmt.Fprintf(out, "removed %d session(s)\n", len(res.Removed))
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
