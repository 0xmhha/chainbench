package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/accountset"
)

// newAccountRosterCmd lists the accounts a chain needs, by role, from a key set.
// The roles are chain-aware (validators + governance council for wbft-family
// chains; node identities only for poa/wemix, whose validators are set at
// bootstrap).
func newAccountRosterCmd() *cobra.Command {
	var chain, keysDir string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "roster",
		Short: "List a chain's accounts by role (validators, governance, nodes) from a key set",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if chain == "" {
				return fmt.Errorf("--chain is required")
			}
			r, err := accountset.Load(chain, keysDir)
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
