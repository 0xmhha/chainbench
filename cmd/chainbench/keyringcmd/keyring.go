package keyringcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	keyringmod "github.com/0xmhha/chainbench/internal/keyring"
)

// New builds the keyring group: create, inspect, and move key material.
//
// It replaces three groups that split the same material by what it was going to
// be used for — `keys` for raw keypairs, `account` for on-chain identity,
// `validator` for consensus identity. They are the same key: on the wbft family
// a node's address and its BLS material derive from one secret.
//
// Every subcommand is flags plus rendering. What a keyring does lives in
// internal/app, so the MCP tools drive the same use cases and the two surfaces
// cannot answer differently.
func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "keyring",
		Short: "Create, inspect, and move key material",
		Long: "A keyring is a directory of node identities. Every command names one with\n" +
			"--keyring-dir (or " + keyringmod.RingEnv + "), and reports which one it used, so the\n" +
			"path a key came from is never a guess.\n\n" +
			"Identities are derived in process: no chain binary has to be built or on\n" +
			"PATH to make a ring.",
	}
	c.AddCommand(
		newKeyringNewCmd(),
		newKeyringAddCmd(),
		newKeyringListCmd(),
		newKeyringShowCmd(),
		newKeyringImportCmd(),
		newKeyringExportCmd(),
	)
	return c
}

// ringFlags name the ring a command works on. The ring may live on a server
// (srv://<server>/path in --keyring-dir), so the server set and the docker
// translation are part of naming it — on every verb, not just import.
type ringFlags struct {
	dir       string
	serverSet string
	docker    bool
}

func (f *ringFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.dir, "keyring-dir", "",
		"ring directory — identities are created in and read from here; a plain path is this machine, "+
			"srv://<server>/path places the ring on that server (default "+
			keyringmod.DefaultRingDir+", or "+keyringmod.RingEnv+")")
	cmd.Flags().StringVar(&f.serverSet, "server-set", "",
		"server-set file for srv:// paths: which servers exist and how to reach them")
	cmd.Flags().BoolVar(&f.docker, "docker", false,
		"the server is a local docker container: translate this tool's dials via the localmap next to the server set (addresses only — docker itself is not touched)")
}

func (f *ringFlags) ref() keyringmod.RingRef {
	return keyringmod.RingRef{Dir: f.dir, ServerSet: f.serverSet, Docker: f.docker}
}

// deps is the Deps every keyring verb runs with: operational side notes —
// today, the dial translations --docker applies — print as they happen, so a
// remote ring is never reached silently. They go to stderr, like every other
// group's: stdout belongs to the answer, and a --json consumer must never
// have to strip a report line off the front of it.
func deps(cmd *cobra.Command) keyringmod.Deps {
	errOut := cmd.ErrOrStderr()
	return keyringmod.Deps{Report: func(format string, args ...any) {
		fmt.Fprintf(errOut, format+"\n", args...)
	}}
}

// announce prints which ring was used and why, before anything else, so the
// path is never a guess — including when the command then fails.
//
// The use case reports a surface-neutral source ("explicit"); here it is named
// in this surface's own vocabulary, because an operator asking "why that ring?"
// wants the flag they typed.
func announce(out io.Writer, r keyringmod.RingOut) {
	fmt.Fprintf(out, "keyring: %s (%s)\n", r.Dir, ringSourceName(r.Source))
}

// ringSourceName renders a use-case source as a CLI reason.
func ringSourceName(source string) string {
	if source == "explicit" {
		return "--keyring-dir"
	}
	return source
}

// renderEntries writes the human listing.
func renderEntries(out io.Writer, r keyringmod.RingOut) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LABEL\tADDRESS\tROLE\tBLS")
	for _, e := range r.Entries {
		role, bls := "-", "-"
		if e.Validator {
			role = "validator"
		}
		if e.BLSPubKey != "" {
			bls = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Label, e.Address, role, bls)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(out, "\n%d identities, %d validators\n", len(r.Entries), r.Validators)
	return nil
}

// renderEntry writes one identity. The private key is printed only when the use
// case supplied one, which only export does.
func renderEntry(out io.Writer, e keyringmod.EntryOut) {
	fmt.Fprintf(out, "label:      %s\n", e.Label)
	fmt.Fprintf(out, "address:    %s\n", e.Address)
	if e.PublicKey != "" {
		fmt.Fprintf(out, "publicKey:  %s\n", e.PublicKey)
	}
	if e.BLSPubKey != "" {
		fmt.Fprintf(out, "blsPubKey:  %s\n", e.BLSPubKey)
		fmt.Fprintf(out, "blsPoP:     %s\n", e.BLSPoP)
	}
	if e.PrivateKey != "" {
		fmt.Fprintf(out, "privateKey: %s\n", e.PrivateKey)
	}
}

// emitJSON writes v as indented JSON.
func emitJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonFlag is the shared --json switch.
type jsonFlag struct{ on bool }

func (f *jsonFlag) bind(cmd *cobra.Command, what string) {
	cmd.Flags().BoolVar(&f.on, "json", false, "emit "+what+" as JSON")
}

// blsFlag is the --with-bls opt-in.
//
// BLS material is only read by the wbft family, so it is asked for rather than
// assumed; a ring for wemix carrying BLS keys would carry keys nobody reads. An
// identity made without it has no BLS material at all, which is a different
// thing from having an empty one.
type blsFlag struct{ on bool }

func (f *blsFlag) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.on, "with-bls", false,
		"also derive BLS material (required for the wbft family: stablenet, wbft)")
}

// labelFlag names one identity in a ring.
type labelFlag struct{ name string }

func (f *labelFlag) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.name, "name", "", "identity label (e.g. node1)")
}

// require refuses a missing --name with the way out, instead of cobra's bare
// "required flag not set": the operator who forgot it usually wanted either
// one identity by name or the whole ring, and the error should offer both.
func (f *labelFlag) require(verb string) error {
	if f.name != "" {
		return nil
	}
	return fmt.Errorf("%s works on one identity — pass --name (e.g. node1); to see them all: keyring list", verb)
}

// shortHex abbreviates a 0x-hex value for a progress line.
func shortHex(s string) string {
	const shown = 14
	if len(s) <= shown {
		return s
	}
	return strings.TrimSpace(s[:shown]) + "…"
}
