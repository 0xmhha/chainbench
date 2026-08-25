package keyringcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// newKeyringListCmd summarizes a ring.
func newKeyringListCmd() *cobra.Command {
	var (
		ring   ringFlags
		jsonF  jsonFlag
		verify bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List a ring's identities",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			r, err := app.KeyringList(cmd.Context(), deps(cmd),
				app.RingListIn{Ring: ring.ref(), Verify: verify})
			if err != nil {
				return err
			}
			announce(out, r)
			if jsonF.on {
				return emitJSON(out, r.Entries)
			}
			return renderEntries(out, r)
		},
	}
	ring.bind(cmd)
	jsonF.bind(cmd, "the listing")
	cmd.Flags().BoolVar(&verify, "verify", false,
		"re-derive every identity from its own key and fail on a mismatch")
	return cmd
}

// newKeyringShowCmd prints one identity's public material.
func newKeyringShowCmd() *cobra.Command {
	var (
		ring  ringFlags
		label labelFlag
		jsonF jsonFlag
	)
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show one identity in full",
		Long: "Prints an identity's public material. The private key is never printed —\n" +
			"use `keyring export` to ask for it on purpose.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if err := label.require("show"); err != nil {
				return err
			}
			e, err := app.KeyringShow(cmd.Context(), deps(cmd),
				app.RingEntryIn{Ring: ring.ref(), Label: label.name})
			if err != nil {
				return err
			}
			if jsonF.on {
				return emitJSON(out, e)
			}
			renderEntry(out, e)
			return nil
		},
	}
	ring.bind(cmd)
	label.bind(cmd)
	jsonF.bind(cmd, "the identity")
	return cmd
}

// newKeyringExportCmd prints one identity's private key.
func newKeyringExportCmd() *cobra.Command {
	var (
		ring    ringFlags
		label   labelFlag
		jsonF   jsonFlag
		confirm bool
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Print an identity's private key",
		Long: "Prints a secret to stdout, which puts it in shell history and scrollback.\n" +
			"--yes is required so that cannot happen by accident, and never as part of\n" +
			"a listing.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return fmt.Errorf("export prints a private key to stdout; pass --yes to confirm")
			}
			out := cmd.OutOrStdout()
			if err := label.require("export"); err != nil {
				return err
			}
			e, err := app.KeyringExport(cmd.Context(), deps(cmd),
				app.RingEntryIn{Ring: ring.ref(), Label: label.name})
			if err != nil {
				return err
			}
			if jsonF.on {
				return emitJSON(out, e)
			}
			renderEntry(out, e)
			return nil
		},
	}
	ring.bind(cmd)
	label.bind(cmd)
	jsonF.bind(cmd, "the identity")
	cmd.Flags().BoolVar(&confirm, "yes", false, "confirm printing a private key")
	return cmd
}
