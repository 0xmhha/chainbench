package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// defaultRingBalance pre-funds each generated identity in the genesis alloc.
// Generated identities are in no existing genesis, so without a balance their
// first transaction cannot pay for gas.
const defaultRingBalance = "0x200000000000000000000000000000000000000000000000000000000000000"

// validatorsFlagHelp reads correctly for both verbs: `new` defaults to all of
// them, `add` defaults to none, because adding an identity and promoting one to
// validator are different decisions. `--validators 0` on `new` declares none at
// all, which is a ring of identities and nothing else.
const validatorsFlagHelp = "how many join the validator set (new: default all, 0 declares none; add: default none)"

// makeFlags are shared by `keyring new` and `keyring add`, which differ only in
// whether the ring has to exist.
type makeFlags struct {
	ring       ringFlags
	bls        blsFlag
	count      int
	validators int
	password   string
	balance    string
}

func (f *makeFlags) bind(cmd *cobra.Command) {
	f.ring.bind(cmd)
	f.bls.bind(cmd)
	cmd.Flags().IntVar(&f.count, "count", 0, "how many identities to create")
	cmd.Flags().IntVar(&f.validators, "validators", 0, validatorsFlagHelp)
	cmd.Flags().StringVar(&f.password, "password", "1", "password for the generated keystores")
	cmd.Flags().StringVar(&f.balance, "balance", defaultRingBalance, "genesis balance per identity (0x-hex wei)")
	_ = cmd.MarkFlagRequired("count")
}

// in builds the use-case input. Whether --validators was typed is carried as a
// pointer, because the flag value alone cannot distinguish "none" from "unset"
// and the two mean opposite things.
func (f *makeFlags) in(cmd *cobra.Command) app.RingCreateIn {
	var validators *int
	if cmd.Flags().Changed("validators") {
		v := f.validators
		validators = &v
	}
	return app.RingCreateIn{
		Ring: f.ring.ref(), Count: f.count, Validators: validators,
		WithBLS: f.bls.on, Password: f.password, Balance: f.balance,
	}
}

// newKeyringNewCmd creates a ring.
func newKeyringNewCmd() *cobra.Command {
	var mk makeFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a ring of identities",
		Long: "Creates a ring: per-identity keys, their derived address and (with\n" +
			"--with-bls) BLS material, an encrypted keystore each, and the index the\n" +
			"harness reads.\n\n" +
			"Nothing is executed — no chain binary needs to exist — so this is how a\n" +
			"network is started from scratch rather than from a committed fixture.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRingCreate(cmd, app.KeyringNew, mk.in(cmd))
		},
	}
	mk.bind(cmd)
	return cmd
}

// newKeyringAddCmd extends an existing ring.
func newKeyringAddCmd() *cobra.Command {
	var mk makeFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add identities to an existing ring",
		Long: "Adds identities, keeping the ones already there. Existing identities are\n" +
			"referenced the moment they exist — in a genesis, in a running datadir, in\n" +
			"a test's declaration — so they are never regenerated.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRingCreate(cmd, app.KeyringAdd, mk.in(cmd))
		},
	}
	mk.bind(cmd)
	return cmd
}

// ringCreateFunc is the shape both creating verbs share.
type ringCreateFunc func(context.Context, app.Deps, app.RingCreateIn) (app.RingOut, error)

// runRingCreate calls the use case and renders it. The two verbs differ in
// which function they call, not in what they print.
func runRingCreate(cmd *cobra.Command, use ringCreateFunc, in app.RingCreateIn) error {
	out := cmd.OutOrStdout()
	r, err := use(cmd.Context(), app.Deps{}, in)
	announce(out, r)
	if err != nil {
		return err
	}
	for _, e := range r.Entries {
		fmt.Fprintf(out, "%-8s %s%s\n", e.Label, e.Address, blsSuffix(e))
	}
	if r.Validators == 0 {
		fmt.Fprintf(out, "\n%d identities in %s; the network declares which validate\n",
			len(r.Entries), r.Dir)
		return nil
	}
	fmt.Fprintf(out, "\n%d identities (%d validators) in %s\n",
		len(r.Entries), r.Validators, r.Dir)
	return nil
}

// blsSuffix abbreviates an identity's BLS key for a creation line, or says
// nothing when it has none.
func blsSuffix(e app.EntryOut) string {
	if e.BLSPubKey == "" {
		return ""
	}
	return "  bls=" + shortHex(e.BLSPubKey)
}
