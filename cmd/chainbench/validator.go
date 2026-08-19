package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/keygen"
	"github.com/0xmhha/chainbench/internal/keymat"
)

// newValidatorCmd is the validator-identity group. A validator is an account
// plus consensus-specific material (BLS keys for wbft; governance/stake
// registration for poa), so it gets its own command rather than living under
// `account`. Subcommands live in the validator_*.go files and are composed here.
func newValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Inspect and manage validator identities (chain-aware consensus roles)",
	}
	cmd.AddCommand(newValidatorNewCmd(), newValidatorImportCmd(), newValidatorRosterCmd(), newValidatorSetCmd())
	return cmd
}

// runValidator resolves a key, attaches the chain's consensus material, and
// prints the validator identity. For a wbft-family chain it derives the BLS key
// and proof-of-possession from the key in process; a poa chain has no genesis
// validator material (validators are registered at bootstrap), so it only
// reports the account with a note.
func runValidator(cmd *cobra.Command, chain string, source keymat.Source, sf *storeFlags, pf *passwordFlags, showPrivate, jsonOut bool) error {
	if chain == "" {
		return fmt.Errorf("--chain is required")
	}
	p, err := registry.Get(chain)
	if err != nil {
		return err
	}
	family := p.Manifest().ConsensusFamily

	a, err := source.Resolve(cmd.Context())
	if err != nil {
		return err
	}
	path, err := saveKey(sf, pf, a)
	if err != nil {
		return err
	}

	v := validatorOut{Chain: chain, Family: family, Address: a.Address().Hex(), Stored: path}
	if showPrivate {
		v.PrivateKey = "0x" + hex.EncodeToString(a.PrivateKeyBytes())
	}
	switch family {
	case "wbft":
		id, err := keygen.DeriveIdentity(hex.EncodeToString(a.PrivateKeyBytes()))
		if err != nil {
			return err
		}
		v.BLSPublicKey = id.BLSPubKey
		v.BLSPoP = id.BLSPoP
	case "poa":
		v.Note = "poa: this validator is registered at the governance/etcd bootstrap, not in genesis; no BLS material."
	default:
		v.Note = fmt.Sprintf("unknown consensus family %q", family)
	}
	return printValidator(cmd.OutOrStdout(), v, jsonOut)
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
