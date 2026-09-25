package lifecyclecmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

func NewClean() *cobra.Command {
	var (
		dataDir      string
		artifactRoot string
		olderThan    string
		keepLast     int
		stale        bool
		serverSet    string
		wcPath       string
		docker       bool
		keepUnder    []string
		apply        bool
	)
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove a launched network's data dir, or GC old session artifacts or stale compositions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if stale {
				return cleanStaleCompositions(cmd, app.StaleCompositionsIn{
					ServerSet: serverSet, Docker: docker, WorkspaceConfigPath: wcPath,
					KeepUnder: keepUnder, Apply: apply,
				})
			}
			if artifactRoot != "" {
				return cleanSessions(cmd, artifactRoot, olderThan, keepLast)
			}
			if dataDir == "" {
				return fmt.Errorf("--workspace-dir (or --artifact-root for session GC) is required")
			}
			res, err := app.NetworkRemove(cmd.Context(), app.Deps{}, app.NetworkRemoveIn{DataDir: dataDir})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "stopped %d node(s)\n", res.Stopped)
			fmt.Fprintf(out, "removed %s\n", res.Removed)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace to stop and remove")
	cmd.Flags().StringVar(&artifactRoot, "artifact-root", "", "session artifact root to garbage-collect")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "GC sessions older than this age (e.g. 7d, 12h)")
	cmd.Flags().IntVar(&keepLast, "keep-last", 0, "GC keeps the newest N sessions")
	cmd.Flags().BoolVar(&stale, "stale-compositions", false,
		"list compositions the servers hold that no known workspace refers to (with --server-set and --workspace-config)")
	cmd.Flags().StringVar(&serverSet, "server-set", "", "server-set file whose servers are searched (--stale-compositions)")
	cmd.Flags().StringVar(&wcPath, "workspace-config", "", "workspace-config giving the data root and its directories (--stale-compositions)")
	cmd.Flags().BoolVar(&docker, "docker", false, "reach the servers through the localmap next to the server set (--stale-compositions)")
	cmd.Flags().StringArrayVar(&keepUnder, "keep-under", nil,
		"directory searched for workspaces whose compositions are kept; repeatable. ~/.chainbench is always searched (--stale-compositions)")
	cmd.Flags().BoolVar(&apply, "apply", false, "remove what --stale-compositions lists; without it nothing is changed")
	return cmd
}

// cleanStaleCompositions lists the compositions nothing refers to, and removes
// them only when --apply is given.
func cleanStaleCompositions(cmd *cobra.Command, in app.StaleCompositionsIn) error {
	res, err := app.StaleCompositions(cmd.Context(), app.Deps{}, in)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, c := range res.Running {
		fmt.Fprintf(out, "kept     %s %s (a process runs from it)\n", c.Server, c.ID)
	}
	verb := "stale   "
	if in.Apply {
		verb = "removed "
	}
	for _, c := range res.Stale {
		fmt.Fprintf(out, "%s %s %s (%d dir(s))\n", verb, c.Server, c.ID, len(c.Dirs))
	}
	fmt.Fprintf(out, "%d stale composition(s), %d kept running, %d known workspace(s)\n", len(res.Stale), len(res.Running), res.Known)
	if !in.Apply && len(res.Stale) > 0 {
		fmt.Fprintln(out, "nothing was removed — rerun with --apply to remove them")
	}
	return nil
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
//
// An age at or below zero is refused. It is not a small age: GCSessions reads a
// non-positive OlderThan as "no age policy at all", so "--older-than -7d
// --keep-last 5" silently dropped the age the operator gave and kept the last
// five whatever their date — deleting more than was asked for, on a command that
// removes directories. The refusal belongs here, where the value is still what
// the operator typed.
func parseAge(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	d, err := parseAgeUnits(s)
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("an age must be positive (got %s)", d)
	}
	return d, nil
}

func parseAgeUnits(s string) (time.Duration, error) {
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
