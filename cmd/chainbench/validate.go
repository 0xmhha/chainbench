package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/testengine"
)

// newValidateCmd parses DSL specs offline and reports which are well-formed,
// without launching anything — fast feedback while authoring or porting specs.
// With --chain it also reports whether each spec would run on that chain.
func newValidateCmd() *cobra.Command {
	var (
		chain   string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "validate [spec.json ...]",
		Short: "Parse and validate DSL test specs without running them",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateSpecs(cmd.OutOrStdout(), args, chain, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "also report whether each spec applies to this chain")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit results as JSON instead of a table")
	return cmd
}

// validateSpecs runs the shared offline validation (the same core the MCP
// surface and the run pre-flight use) and renders it. It returns exit code 1
// when any spec is unreadable or invalid; applicability/capability outcomes are
// informational and stay OK.
func validateSpecs(out io.Writer, paths []string, chain string, jsonOut bool) error {
	results, err := testengine.ValidateSpecs(paths, chain)
	if err != nil {
		return err
	}
	if err := renderValidate(out, results, jsonOut); err != nil {
		return err
	}
	invalid := 0
	for _, r := range results {
		if !r.OK {
			invalid++
		}
	}
	if invalid > 0 {
		return &exitError{code: 1, err: fmt.Errorf("validate: %d invalid spec(s)", invalid)}
	}
	return nil
}

// renderValidate writes the results as a table or, with jsonOut, as a JSON array.
func renderValidate(out io.Writer, results []testengine.ValidateResult, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SPEC\tID\tRESULT")
	for _, r := range results {
		id := r.ID
		if id == "" {
			id = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Spec, id, r.Result)
	}
	return w.Flush()
}
