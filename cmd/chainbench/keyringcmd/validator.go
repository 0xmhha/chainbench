package keyringcmd

import (
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/validatorset"
)

// NewValidator builds the validator-identity group. A validator is an account
// plus consensus-specific material (BLS keys for wbft; governance/stake
// registration for poa), so it gets its own command rather than living under
// `account`. Subcommands live in the validator_*.go files and are composed here.
func NewValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Inspect and manage validator identities; key material lives under `keyring`",
	}
	cmd.AddCommand(newValidatorNewCmd(), newValidatorImportCmd(), newValidatorRosterCmd(), newValidatorSetCmd())
	return cmd
}

// runValidator resolves a key, attaches the chain's consensus material, and
// prints the validator identity. For a wbft-family chain it derives the BLS key
// and proof-of-possession from the key in process; a poa chain has no genesis
// validator material (validators are registered at bootstrap), so it only
// reports the account with a note.
func runValidator(cmd *cobra.Command, chain string, source keyring.Source, sf *storeFlags, pf *PasswordFlags, showPrivate, jsonOut bool) error {
	if chain == "" {
		return fmt.Errorf("--chain is required")
	}
	p, err := registry.Get(chain)
	if err != nil {
		return err
	}
	family := p.Manifest().ConsensusFamily

	key, err := source.Resolve(cmd.Context())
	if err != nil {
		return err
	}
	path, err := saveKey(sf, pf, key)
	if err != nil {
		return err
	}
	// A wbft validator's BLS material comes from the same key as its address,
	// so ask for it up front and let the family decide whether it is used.
	id, err := derive.Derive(key, derivationFor(family))
	if err != nil {
		return err
	}

	v := validatorOut{Chain: chain, Family: family, Address: id.Address, Stored: path}
	if showPrivate {
		v.PrivateKey = "0x" + key.Hex()
	}
	switch family {
	case "wbft":
		v.BLSPublicKey = id.BLS.PublicKey
		v.BLSPoP = id.BLS.PoP
	case "poa":
		v.Note = "poa: this validator is registered at the governance/etcd bootstrap, not in genesis; no BLS material."
	default:
		v.Note = fmt.Sprintf("unknown consensus family %q", family)
	}
	return printValidator(cmd.OutOrStdout(), v, jsonOut)
}

// derivationFor asks for BLS material only where a family uses it, so a poa
// validator does not pay for a computation whose result it would discard.
func derivationFor(family string) derive.Derivation {
	if family == "wbft" {
		return derive.WithBLS
	}
	return derive.AccountOnly
}

// validatorOut is a validator identity for display.
type validatorOut struct {
	Chain        string `json:"chain"`
	Family       string `json:"family"`
	Address      string `json:"address"`
	PrivateKey   string `json:"privateKey,omitempty"`
	BLSPublicKey string `json:"blsPublicKey,omitempty"`
	BLSPoP       string `json:"blsPoP,omitempty"`
	Stored       string `json:"stored,omitempty"`
	Note         string `json:"note,omitempty"`
}

func printValidator(out io.Writer, v validatorOut, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	fmt.Fprintf(out, "chain:   %s  family: %s\n", v.Chain, v.Family)
	if v.PrivateKey != "" {
		fmt.Fprintf(out, "privateKey: %s\n", v.PrivateKey)
	}
	fmt.Fprintf(out, "address: %s\n", v.Address)
	if v.BLSPublicKey != "" {
		fmt.Fprintf(out, "blsPublicKey: %s\nblsPoP:       %s\n", v.BLSPublicKey, v.BLSPoP)
	}
	if v.Stored != "" {
		fmt.Fprintf(out, "stored:  %s\n", v.Stored)
	}
	if v.Note != "" {
		fmt.Fprintf(out, "note: %s\n", v.Note)
	}
	return nil
}

// newValidatorNewCmd generates a new validator identity for a chain: a fresh
// account plus the chain's consensus material (BLS/PoP for wbft, derived in process;
// none for poa, whose validators register at bootstrap). Optionally stores the
// key.
func newValidatorNewCmd() *cobra.Command {
	var chain string
	var jsonOut bool
	var sf storeFlags
	var pf PasswordFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a new validator identity for a chain (chain-aware consensus material)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidator(cmd, chain, keyring.RandomSource{}, &sf, &pf, true, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the validator identity as JSON")
	sf.bind(cmd)
	pf.Bind(cmd)
	return cmd
}

// newValidatorImportCmd imports an existing key as a validator identity for a
// chain — from a private key, mnemonic, or file — and attaches the chain's
// consensus material (BLS/PoP for wbft, derived in process).
func newValidatorImportCmd() *cobra.Command {
	var chain string
	var jsonOut bool
	var src SourceFlags
	var sf storeFlags
	var pf PasswordFlags
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a key as a validator identity for a chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source, err := src.Source(pf.Source())
			if err != nil {
				return err
			}
			return runValidator(cmd, chain, source, &sf, &pf, false, jsonOut)
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "chain id (stablenet|wbft|wemix)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the validator identity as JSON")
	src.Bind(cmd)
	sf.bind(cmd)
	pf.Bind(cmd)
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

// newValidatorSetCmd generates a network's validator set — the preset key
// bundle (per-node keys, BLS, keystores, metadata) that defines the chain's
// validators. It was `keys generate`; it lives under `validator` because a
// preset is defined by its validator set, not by raw keys.
func newValidatorSetCmd() *cobra.Command {
	var (
		opts       store.GenerateOpts
		validators int
		basePort   int // superseded; see below
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Generate a validator set / preset key bundle (nodekeys, BLS, keystores, metadata)",
		Long: "Generates the preset key set the harness consumes (store.LoadPreset): per-node\n" +
			"nodekeys, their derived address + BLS public key/PoP (derived in process),\n" +
			"an encrypted keystore per node (via the accounts SDK — no node binary),\n" +
			"and a metadata.json. Use it to build validator sets larger than the committed\n" +
			"5-node preset (e.g. the n=6 quorum cases).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			// `validator set` builds a wbft validator set, which is defined by its
			// BLS keys; `keyring new` is where BLS became opt-in.
			opts.Derive = derive.WithBLS
			// This command's zero has always meant "all of them"; pass it as
			// absent rather than as a declared zero, which now means a ring
			// that declares no validators at all.
			if validators > 0 {
				opts.Validators = &validators
			}
			meta, err := store.Generate(opts, func(line string) { fmt.Fprintln(out, line) })
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "wrote %d-node preset (%d validators) to %s\n", len(meta.Nodes), len(meta.Network.Validators), opts.Out)
			return nil
		},
	}
	cmd.Flags().IntVar(&opts.Nodes, "nodes", 0, "total nodes to generate")
	cmd.Flags().IntVar(&validators, "validators", 0, "how many nodes are validators (default: all)")
	cmd.Flags().StringVar(&opts.Out, "out", "", "output preset directory")
	cmd.Flags().StringVar(&opts.Password, "password", "1", "keystore password")
	// The generated metadata used to carry an enode per node, built from this
	// port and 127.0.0.1. Nothing read it: enodes are assembled at compose time
	// from the public key and the node's actual host and port, which is the only
	// place both are known. The flag is accepted so scripts keep running.
	cmd.Flags().IntVar(&basePort, "base-p2p", 30301, "deprecated: ignored, enodes are built at compose time")
	_ = cmd.Flags().MarkDeprecated("base-p2p", "no longer used — enodes are built when a network is composed")
	cmd.Flags().StringVar(&opts.Balance, "balance", "0x200000000000000000000000000000000000000000000000000000000000000", "genesis balance per node (0x-hex wei)")
	_ = cmd.MarkFlagRequired("nodes")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}
