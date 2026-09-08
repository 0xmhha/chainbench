// Package testcmd lists the DSL test cases in a directory, so an operator can
// discover what is there before running one with `chainbench run`. It is the
// CLI mirror of the MCP test_list tool; both read internal/app.ListSpecs, so
// the two surfaces cannot drift (WA2).
package testcmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// printJSON writes the catalog as indented JSON.
func printJSON(w io.Writer, specs []app.SpecInfo) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(specs)
}

// New returns the `test` command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Discover the DSL test cases in a directory",
	}
	cmd.AddCommand(newListCmd())
	return cmd
}

func newListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list <dir>",
		Short: "List the runnable test cases under a directory (recursively)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, argv []string) error {
			specs, err := app.ListSpecs(argv[0])
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if jsonOut {
				return printJSON(w, specs)
			}
			if len(specs) == 0 {
				fmt.Fprintln(w, "no test cases found")
				return nil
			}
			for _, s := range specs {
				chain := s.Chain
				if chain == "" {
					chain = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Path, s.ID, chain, s.Description)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print the catalog as JSON")
	return cmd
}
