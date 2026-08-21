package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// env is the environment lookup the keyring commands use. It is a function so a
// test can supply its own without touching the process environment.
func env() func(string) string { return os.Getenv }

// newKeyringListCmd summarizes a ring.
func newKeyringListCmd() *cobra.Command {
	var (
		ring    ringFlags
		jsonOut bool
		verify  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List a ring's identities",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			set, _, err := ring.open(out)
			if err != nil {
				return err
			}
			views := make([]entryView, 0, len(set.Nodes))
			for _, e := range set.Nodes {
				v := viewOf(e)
				views = append(views, v)
			}
			if verify {
				if err := verifyAll(set); err != nil {
					return err
				}
			}
			if jsonOut {
				return encodeJSON(out, views)
			}
			return writeTable(out, set, views)
		},
	}
	ring.bind(cmd)
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the listing as JSON")
	cmd.Flags().BoolVar(&verify, "verify", false,
		"re-derive every identity from its own key and fail on a mismatch")
	return cmd
}

// verifyAll re-derives every identity. A ring whose recorded identity has
// drifted from its key material launches nodes signing as one address while the
// genesis registers another, so it is worth being able to ask.
func verifyAll(set keyring.Preset) error {
	for _, e := range set.Nodes {
		if err := e.Verify(); err != nil {
			return err
		}
	}
	return nil
}

// writeTable renders the human listing.
func writeTable(out io.Writer, set keyring.Preset, views []entryView) error {
	validators := map[string]bool{}
	for _, a := range set.Network.Validators {
		validators[strings.ToLower(a)] = true
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LABEL\tADDRESS\tROLE\tBLS")
	for _, v := range views {
		role := "-"
		if validators[strings.ToLower(v.Address)] {
			role = "validator"
		}
		bls := "-"
		if v.BLSPubKey != "" {
			bls = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.Label, v.Address, role, bls)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(out, "\n%d identities, %d validators\n", len(views), len(set.Network.Validators))
	return nil
}

// newKeyringShowCmd prints one identity in full.
func newKeyringShowCmd() *cobra.Command {
	var (
		ring    ringFlags
		name    string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show one identity in full",
		Long: "Prints an identity's public material. The private key is never printed —\n" +
			"use `keyring export` to ask for it on purpose.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			set, _, err := ring.open(out)
			if err != nil {
				return err
			}
			e, err := findEntry(set, name)
			if err != nil {
				return err
			}
			v := viewOf(e)
			if jsonOut {
				return encodeJSON(out, v)
			}
			fmt.Fprintf(out, "label:      %s\n", v.Label)
			fmt.Fprintf(out, "address:    %s\n", v.Address)
			fmt.Fprintf(out, "publicKey:  %s\n", v.PublicKey)
			if v.BLSPubKey != "" {
				fmt.Fprintf(out, "blsPubKey:  %s\n", v.BLSPubKey)
				fmt.Fprintf(out, "blsPoP:     %s\n", v.BLSPoP)
			}
			return nil
		},
	}
	ring.bind(cmd)
	cmd.Flags().StringVar(&name, "name", "", "identity label (e.g. node1)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the identity as JSON")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// newKeyringExportCmd prints one identity's private key.
func newKeyringExportCmd() *cobra.Command {
	var (
		ring    ringFlags
		name    string
		jsonOut bool
		confirm bool
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Print an identity's private key",
		Long: "Prints a secret to stdout, which puts it in shell history and scrollback.\n" +
			"--yes is required so that cannot happen by accident, and never as part of\n" +
			"a listing.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if !confirm {
				return fmt.Errorf("export prints a private key to stdout; pass --yes to confirm")
			}
			set, _, err := ring.open(out)
			if err != nil {
				return err
			}
			e, err := findEntry(set, name)
			if err != nil {
				return err
			}
			v := viewOf(e)
			v.Nodekey = "0x" + e.Nodekey.Hex()
			if jsonOut {
				return encodeJSON(out, v)
			}
			fmt.Fprintf(out, "label:      %s\n", v.Label)
			fmt.Fprintf(out, "address:    %s\n", v.Address)
			fmt.Fprintf(out, "privateKey: %s\n", v.Nodekey)
			return nil
		},
	}
	ring.bind(cmd)
	cmd.Flags().StringVar(&name, "name", "", "identity label (e.g. node1)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the identity as JSON")
	cmd.Flags().BoolVar(&confirm, "yes", false, "confirm printing a private key")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// findEntry looks up an identity by label, listing what a ring holds when the
// name is not one of them.
func findEntry(set keyring.Preset, name string) (keyring.Entry, error) {
	for _, e := range set.Nodes {
		if string(e.Label) == name {
			return e, nil
		}
	}
	labels := make([]string, 0, len(set.Nodes))
	for _, e := range set.Nodes {
		labels = append(labels, string(e.Label))
	}
	return keyring.Entry{}, fmt.Errorf("no identity named %q (have: %v)", name, labels)
}

// encodeJSON writes v as indented JSON.
func encodeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
