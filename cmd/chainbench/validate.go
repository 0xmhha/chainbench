package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/testspec"
)

// newValidateCmd parses DSL specs offline and reports which are well-formed,
// without launching anything — fast feedback while authoring or porting specs.
func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [spec.json ...]",
		Short: "Parse and validate DSL test specs without running them",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateSpecs(cmd.OutOrStdout(), args)
		},
	}
	return cmd
}

// validateSpecs parses each spec file and prints a per-file result table. It
// returns exit code 1 when any spec is unreadable or invalid.
func validateSpecs(out io.Writer, paths []string) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SPEC\tID\tRESULT")
	invalid := 0
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(w, "%s\t-\tERROR: %v\n", p, err)
			invalid++
			continue
		}
		s, err := testspec.Parse(raw)
		if err != nil {
			fmt.Fprintf(w, "%s\t-\tINVALID: %v\n", p, err)
			invalid++
			continue
		}
		fmt.Fprintf(w, "%s\t%s\tOK\n", p, s.ID)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if invalid > 0 {
		return &exitError{code: 1, err: fmt.Errorf("validate: %d invalid spec(s)", invalid)}
	}
	return nil
}
