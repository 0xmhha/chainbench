package reportcmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

func NewLog() *cobra.Command {
	var (
		dataDir string
		pattern string
		useRe   bool
		node    int
		level   string
		limit   int
	)
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Search a workspace's per-node logs (from --workspace-dir/logs)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dataDir == "" {
				return fmt.Errorf("--data-dir is required")
			}
			matches, err := collector.Search(dataDir, collector.SearchOpts{
				Pattern: pattern,
				Regexp:  useRe,
				Node:    node,
				Level:   level,
				Limit:   limit,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(matches) == 0 {
				fmt.Fprintln(out, "no matching log lines")
				return nil
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			for _, m := range matches {
				fmt.Fprintf(w, "node%d:%d\t%s\n", m.Node, m.Line, m.Text)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			fmt.Fprintf(out, "%d line(s)\n", len(matches))
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "workspace-dir", "", "workspace directory (searches <workspace-dir>/logs)")
	cmd.Flags().StringVar(&pattern, "pattern", "", "substring (or regexp with --regexp) to match")
	cmd.Flags().BoolVar(&useRe, "regexp", false, "treat --pattern as a regular expression")
	cmd.Flags().IntVar(&node, "node", 0, "restrict to a 1-based node index (0 = all)")
	cmd.Flags().StringVar(&level, "level", "", "minimum severity (TRACE|DEBUG|INFO|WARN|ERROR|CRIT)")
	cmd.Flags().IntVar(&limit, "limit", 0, "cap the number of lines (0 = no cap)")
	return cmd
}
