package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// defaultRingBalance pre-funds each generated account in the genesis alloc.
// Generated identities are in no shipped preset's alloc, so without a balance
// their first transaction cannot pay for gas.
const defaultRingBalance = "0x200000000000000000000000000000000000000000000000000000000000000"

// makeFlags are shared by `keyring new` and `keyring add`, which differ only in
// whether the ring has to exist.
type makeFlags struct {
	count      int
	validators int
	password   string
	balance    string
	derivation derivationFlag
}

// validatorsFlagHelp reads correctly for both verbs: `new` defaults to all of
// them, `add` defaults to none, because adding an identity and promoting one to
// validator are different decisions.
const validatorsFlagHelp = "how many join the validator set (new: default all; add: default none)"

func (f *makeFlags) bind(cmd *cobra.Command) {
	cmd.Flags().IntVar(&f.count, "count", 0, "how many identities to create")
	cmd.Flags().IntVar(&f.validators, "validators", 0, validatorsFlagHelp)
	cmd.Flags().StringVar(&f.password, "password", "1", "password for the generated keystores")
	cmd.Flags().StringVar(&f.balance, "balance", defaultRingBalance, "genesis balance per identity (0x-hex wei)")
	f.derivation.bind(cmd)
	_ = cmd.MarkFlagRequired("count")
}

func (f *makeFlags) opts(dir string) keyring.GenerateOpts {
	return keyring.GenerateOpts{
		Nodes:      f.count,
		Validators: f.validators,
		Out:        dir,
		Password:   f.password,
		Balance:    f.balance,
		Derive:     f.derivation.derivation(),
	}
}

// newKeyringNewCmd creates a ring.
func newKeyringNewCmd() *cobra.Command {
	var (
		ring ringFlags
		mk   makeFlags
	)
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a ring of identities",
		Long: "Creates a ring: per-identity keys, their derived address and (with\n" +
			"--with-bls) BLS material, an encrypted keystore each, and the index the\n" +
			"harness reads.\n\n" +
			"Nothing is executed — no chain binary needs to exist — so this is how a\n" +
			"network is started from scratch rather than from a committed fixture.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			dir, source := ring.resolve(env())
			fmt.Fprintf(out, "keyring: %s (%s)\n", dir, source)

			set, err := keyring.Generate(mk.opts(dir), func(line string) { fmt.Fprintln(out, line) })
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "created %d identities (%d validators) in %s\n",
				len(set.Nodes), len(set.Validators), dir)
			return nil
		},
	}
	ring.bind(cmd)
	mk.bind(cmd)
	return cmd
}

// newKeyringAddCmd extends an existing ring.
func newKeyringAddCmd() *cobra.Command {
	var (
		ring ringFlags
		mk   makeFlags
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add identities to an existing ring",
		Long: "Adds identities, keeping the ones already there. Existing identities are\n" +
			"referenced the moment they exist — in a genesis, in a running datadir, in\n" +
			"a test's declaration — so they are never regenerated.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			dir, source := ring.resolve(env())
			fmt.Fprintf(out, "keyring: %s (%s)\n", dir, source)

			before, err := keyring.LoadPreset(dir)
			if err != nil {
				return err
			}
			set, err := keyring.Extend(mk.opts(dir), func(line string) { fmt.Fprintln(out, line) })
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "added %d identities (%d total) in %s\n",
				len(set.Nodes)-len(before.Nodes), len(set.Nodes), dir)
			return nil
		},
	}
	ring.bind(cmd)
	mk.bind(cmd)
	return cmd
}
