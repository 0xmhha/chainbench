package keyringcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/0xmhha/chainbench/internal/resource"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/keyring/operation"
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
			"--keyring-dir (or " + operation.KeySetEnv + "), and reports which one it used, so the\n" +
			"path a key came from is never a guess.\n\n" +
			"Identities are derived in process: no chain binary has to be built or on\n" +
			"PATH to make a key set.",
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

// ringFlags name the key set a command works on. The key set may live on a server
// (srv://<server>/path in --keyring-dir), so the server set and the docker
// translation are part of naming it — on every verb, not just import.
type ringFlags struct {
	dir       string
	serverSet string
	docker    bool
}

func (f *ringFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.dir, "keyring-dir", "",
		"key set directory — identities are created in and read from here; a plain path is this machine, "+
			"srv://<server>/path places the key set on that server (default "+
			operation.DefaultKeySetDir+", or "+operation.KeySetEnv+")")
	cmd.Flags().StringVar(&f.serverSet, "server-set", "",
		"server-set file for srv:// paths: which servers exist and how to reach them")
	cmd.Flags().BoolVar(&f.docker, "docker", false,
		"the server is a local docker container: translate this tool's dials via the localmap next to the server set (addresses only — docker itself is not touched)")
}

func (f *ringFlags) ref() operation.SetRef {
	return operation.SetRef{Dir: f.dir, ServerSet: f.serverSet, Docker: f.docker}
}

// deps is the Deps every keyring verb runs with: operational side notes —
// today, the dial translations --docker applies — print as they happen, so a
// remote key set is never reached silently. They go to stderr, like every other
// group's: stdout belongs to the answer, and a --json consumer must never
// have to strip a report line off the front of it.
func deps(cmd *cobra.Command) operation.Deps {
	errOut := cmd.ErrOrStderr()
	report := func(format string, args ...any) {
		fmt.Fprintf(errOut, format+"\n", args...)
	}
	return operation.Deps{
		// The resource module owns machines and server sets; the operations
		// only need something that opens a path, so this is where the two
		// meet.
		Open: func(serverSet string, docker bool) operation.Opener {
			return resource.Opener{
				ServerSet: serverSet, Docker: docker,
				Env: os.Getenv, Report: report,
			}
		},
	}
}

// announce prints which key set was used and why, before anything else, so the
// path is never a guess — including when the command then fails, and
// including with --json. It goes to stderr: stdout belongs to the answer,
// and `keyring list --json | jq` must never have to strip a report line off
// the front of it (the same rule deps' operational notes follow).
//
// The use case reports a surface-neutral source ("explicit"); here it is named
// in this surface's own vocabulary, because an operator asking "why that key set?"
// wants the flag they typed.
func announce(cmd *cobra.Command, r operation.SetOut) {
	fmt.Fprintf(cmd.ErrOrStderr(), "keyring: %s (%s)\n", r.Dir, ringSourceName(r.Source))
}

// ringSourceName renders a use-case source as a CLI reason.
func ringSourceName(source string) string {
	if source == "explicit" {
		return "--keyring-dir"
	}
	return source
}

// renderEntries writes the human listing.
func renderEntries(out io.Writer, r operation.SetOut) error {
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
func renderEntry(out io.Writer, e operation.EntryOut) {
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
// assumed; a key set for wemix carrying BLS keys would carry keys nobody reads. An
// identity made without it has no BLS material at all, which is a different
// thing from having an empty one.
type blsFlag struct{ on bool }

func (f *blsFlag) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.on, "with-bls", false,
		"also derive BLS material (required for the wbft family: stablenet, wbft)")
}

// labelFlag names one identity in a key set.
type labelFlag struct{ name string }

func (f *labelFlag) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.name, "name", "", "identity label (e.g. node1)")
}

// require refuses a missing --name with the way out, instead of cobra's bare
// "required flag not set": the operator who forgot it usually wanted either
// one identity by name or the whole key set, and the error should offer both.
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
