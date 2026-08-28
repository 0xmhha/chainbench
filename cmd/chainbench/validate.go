package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/0xmhha/chainbench/internal/testhelper"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/testspec"
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

// validateResult is one spec's validation outcome (the --json shape). OK is true
// when the spec parses and all names resolve; Result carries the human detail.
type validateResult struct {
	Spec   string `json:"spec"`
	ID     string `json:"id,omitempty"`
	OK     bool   `json:"ok"`
	Result string `json:"result"`
}

// validateSpecs parses each spec file and reports a per-file result (table, or
// JSON with --json). When chain is set, a valid spec's result also reflects
// applicability and required capabilities against that chain. It returns exit
// code 1 when any spec is unreadable or invalid (applicability/capability
// outcomes are informational and remain OK).
func validateSpecs(out io.Writer, paths []string, chain string, jsonOut bool) error {
	var caps []string
	if chain != "" {
		plugin, err := registry.Get(chain)
		if err != nil {
			return fmt.Errorf("validate: --chain: %w", err)
		}
		caps = plugin.Manifest().Capabilities
	}

	// Resolve step/assertion names against the built-in registry so typo'd names
	// are caught offline rather than at run time.
	reg := testhelper.Registry()

	results := make([]validateResult, 0, len(paths))
	invalid := 0
	for _, p := range paths {
		r := validateResult{Spec: p}
		raws, err := testspec.ReadFiles([]string{p})
		if err != nil {
			r.Result = "ERROR: " + err.Error()
		} else if testspec.IsEnv(raws[0]) {
			// An env is a declaration, not a run: it validates on its own
			// terms and is exercised through the cases that name it.
			if env, perr := testspec.ParseEnv(raws[0]); perr != nil {
				r.Result = "INVALID: " + perr.Error()
			} else {
				r.ID, r.OK, r.Result = env.ID, true, "env declaration for chain "+env.Chain
			}
		} else if s, perr := testspec.Parse(raws[0]); perr != nil {
			r.Result = "INVALID: " + perr.Error()
		} else if unresolved := testspec.Unresolved(s, reg); len(unresolved) > 0 {
			r.ID = s.ID
			r.Result = "UNRESOLVED: " + strings.Join(unresolved, ", ")
		} else {
			r.ID, r.OK, r.Result = s.ID, true, specResult(s, chain, caps)
		}
		if !r.OK {
			invalid++
		}
		results = append(results, r)
	}

	if err := renderValidate(out, results, jsonOut); err != nil {
		return err
	}
	if invalid > 0 {
		return &exitError{code: 1, err: fmt.Errorf("validate: %d invalid spec(s)", invalid)}
	}
	return nil
}

// renderValidate writes the results as a table or, with jsonOut, as a JSON array.
func renderValidate(out io.Writer, results []validateResult, jsonOut bool) error {
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

// specResult describes a parsed spec's status against an optional target chain:
// plain OK without a chain, else OK / SKIP (not applicable) / SKIP (needs caps).
func specResult(s testspec.Spec, chain string, caps []string) string {
	if chain == "" {
		return "OK"
	}
	if !chainApplies(s.ApplicableChains, chain) {
		return "SKIP (chain not applicable)"
	}
	if missing := missingCaps(s.Requires, caps); len(missing) > 0 {
		return "SKIP (needs caps: " + strings.Join(missing, ",") + ")"
	}
	return "OK"
}

// chainApplies reports whether a spec's applicableChains (comma/space separated,
// empty = all) includes chain.
func chainApplies(applicableChains, chain string) bool {
	list := strings.FieldsFunc(applicableChains, func(r rune) bool { return r == ',' || r == ' ' })
	if len(list) == 0 {
		return true
	}
	for _, c := range list {
		if c == chain {
			return true
		}
	}
	return false
}

// missingCaps returns the required capabilities not present in have.
func missingCaps(required, have []string) []string {
	set := make(map[string]bool, len(have))
	for _, c := range have {
		set[c] = true
	}
	var missing []string
	for _, r := range required {
		if !set[r] {
			missing = append(missing, r)
		}
	}
	return missing
}
