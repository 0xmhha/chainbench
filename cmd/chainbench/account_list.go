package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/accounts/types"
	"github.com/0xmhha/chainbench/internal/keymat"
)

// newAccountListCmd lists the accounts stored in a directory (as produced by
// `account new --out` / `keys new --out`): keystore JSON files and raw hex key
// files, by address. It needs no password — a keystore's address is plaintext.
func newAccountListCmd() *cobra.Command {
	var dir string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored accounts (addresses) in a directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dir == "" {
				return fmt.Errorf("--dir is required")
			}
			accts, err := listStoredAccounts(dir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(accts)
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tADDRESS")
			for _, a := range accts {
				fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.Type, a.Address)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "directory of stored keys (keystore .json / raw .key)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the list as JSON")
	return cmd
}

// storedAccount is one account found in a key directory.
type storedAccount struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Address string `json:"address"`
}

// listStoredAccounts scans dir for keystore (.json) and raw (.key) files and
// resolves each to an address — the keystore address is read from the JSON
// (plaintext, no password), a raw key is derived.
func listStoredAccounts(dir string) ([]storedAccount, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("account list: %w", err)
	}
	var out []storedAccount
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(dir, name)
		switch {
		case strings.HasSuffix(name, ".json"):
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var ks struct {
				Address string `json:"address"`
			}
			if json.Unmarshal(data, &ks) == nil && ks.Address != "" {
				out = append(out, storedAccount{Name: name, Type: "keystore", Address: normalizeAddress(ks.Address)})
			}
		case strings.HasSuffix(name, ".key"):
			a, err := keymat.FileSource{Path: path}.Resolve(context.Background())
			if err == nil {
				out = append(out, storedAccount{Name: name, Type: "raw", Address: a.Address().Hex()})
			}
		}
	}
	return out, nil
}

// normalizeAddress returns the checksummed 0x-hex form of a keystore address
// (which is stored without a 0x prefix), falling back to the raw value.
func normalizeAddress(a string) string {
	addr, err := types.HexToAddress("0x" + strings.TrimPrefix(a, "0x"))
	if err != nil {
		return "0x" + a
	}
	return addr.Hex()
}
