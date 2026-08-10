package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/validatorset"
)

// newValidatorCmd is the validator-identity surface: a validator is an account
// plus consensus-specific material (BLS keys for wbft; governance/stake
// registration for poa), so it gets its own subcommand rather than living under
// `account`. Today it inspects the validator set; generation lands here too.
func newValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Inspect and manage validator identities (chain-aware consensus roles)",
	}
	cmd.AddCommand(newValidatorRosterCmd())
	return cmd
}

// newValidatorRosterCmd lists a chain's validator set (and related consensus
// roles) from a key set. Chain-aware: validators carry BLS for wbft-family
// chains and the anzeon governance council; a poa chain (wemix) has no genesis
// validators (they are registered at bootstrap).
func newValidatorRosterCmd() *cobra.Command {
	var chain, keysDir string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "roster",
		Short: "List a chain's validator set and consensus roles from a key set",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if chain == "" {
				return fmt.Errorf("--chain is required")
			}
			r, err := validatorset.Load(chain, keysDir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(r)
			}
			fmt.Fprintf(out, "chain: %s  family: %s\n", r.Chain, r.Family)
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ROLE\tINDEX\tADDRESS\tDETAIL")
			for _, a := range r.Accounts {
				idx := ""
				if a.Index > 0 {
					idx = fmt.Sprintf("%d", a.Index)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Role, idx, a.Address, a.Detail)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if r.Note != "" {
				fmt.Fprintf(out, "\nnote: %s\n", r.Note)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "key set (preset) directory")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the roster as JSON")
	return cmd
}
