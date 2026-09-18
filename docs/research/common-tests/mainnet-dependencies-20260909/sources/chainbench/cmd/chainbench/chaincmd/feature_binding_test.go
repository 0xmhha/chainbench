package chaincmd_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0xmhha/chainbench/cmd/chainbench/chaincmd"
	"github.com/0xmhha/chainbench/internal/feature"
)

// composeCommands pairs a registered feature with the chain subcommand that
// renders it today. Only the pairs that exist on the CLI are here: a feature
// with no command yet is not a disagreement.
var composeCommands = map[string]string{
	"chain.place":   "place",
	"chain.keys":    "keys",
	"chain.genesis": "genesis",
	"chain.config":  "config",
	"chain.build":   "build",
	"chain.deploy":  "deploy",
	"chain.init":    "init",
	"chain.start":   "start",
	"chain.status":  "status",
	"chain.health":  "health",
	"chain.enode":   "enode",
	"chain.logs":    "logs",
}

// TestComposeFeatures_TagsMatchTheCommands proves the derivation before
// anything depends on it.
//
// S0 built the binding and S1 registered the compose features; the tags are
// still inert, so the only way to know they would reproduce the shipped surface
// is to derive the flags and compare them against what each command declares by
// hand.
//
// The comparison is one-directional on purpose. Every flag the tags derive must
// exist on the command, with the same type and the same help text. The command
// may carry MORE — the shared binders add --server, --server-set and the target
// flags, which belong to no single input — and requiring those in the tags
// would mean copying a shared thing into every struct that uses it.
//
// If they match, S1's follow-up is a deletion of the hand-written half rather
// than a change of surface. If they drift, this says so while both halves still
// exist to be compared.
func TestComposeFeatures_TagsMatchTheCommands(t *testing.T) {
	byName := map[string]*cobra.Command{}
	for _, c := range chaincmd.New().Commands() {
		byName[c.Name()] = c
	}
	checked := 0
	for _, reg := range feature.Registered() {
		name, ok := composeCommands[reg.Name]
		if !ok {
			continue
		}
		cmd := byName[name]
		if cmd == nil {
			t.Errorf("%s is registered but chain %s is gone", reg.Name, name)
			continue
		}
		derived := pflag.NewFlagSet(reg.Name, pflag.ContinueOnError)
		if err := feature.Flags(reg.Input(), derived); err != nil {
			t.Errorf("%s: derive flags: %v", reg.Name, err)
			continue
		}
		derived.VisitAll(func(f *pflag.Flag) {
			checked++
			got := cmd.Flags().Lookup(f.Name)
			if got == nil {
				t.Errorf("%s declares --%s, which chain %s does not offer", reg.Name, f.Name, name)
				return
			}
			if got.Value.Type() != f.Value.Type() {
				t.Errorf("%s --%s is %s in the tags and %s on chain %s", reg.Name, f.Name, f.Value.Type(), got.Value.Type(), name)
			}
			if got.Usage != f.Usage {
				t.Errorf("%s --%s reads differently:\n  tags:    %q\n  command: %q", reg.Name, f.Name, f.Usage, got.Usage)
			}
			if got.DefValue != f.DefValue {
				t.Errorf("%s --%s defaults to %q in the tags and %q on chain %s", reg.Name, f.Name, f.DefValue, got.DefValue, name)
			}
		})
	}
	if checked == 0 {
		t.Fatal("no derived flag was compared, so this test asserts nothing")
	}
	t.Logf("%d derived flags across %d features match what the commands declare by hand", checked, len(composeCommands))
}
