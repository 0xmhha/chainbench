package chaincmd_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0xmhha/chainbench/cmd/chainbench/chaincmd"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
)

// TestNetGenesis_TagsMatchTheCommand proves the derivation before anything
// depends on it.
//
// S0 builds the binding; S1 moves the commands onto it. In between, the tags on
// NetGenesisIn are inert — nothing reads them at run time — so the only way to
// know they would reproduce the shipped surface is to derive the flags and
// compare them against the ones `chain genesis` declares by hand.
//
// The comparison is name, type and help text, because those are what a person
// sees. If they match, the migration is a deletion of the hand-written half
// rather than a change of surface; if they drift, this says so while both
// halves still exist to be compared.
func TestNetGenesis_TagsMatchTheCommand(t *testing.T) {
	var genesis *cobra.Command
	for _, c := range chaincmd.New().Commands() {
		if c.Name() == "genesis" {
			genesis = c
		}
	}
	if genesis == nil {
		t.Fatal("chain genesis is gone")
	}

	derived := pflag.NewFlagSet("derived", pflag.ContinueOnError)
	if err := feature.Flags(&app.NetGenesisIn{}, derived); err != nil {
		t.Fatalf("derive flags from the tags: %v", err)
	}

	got := describe(derived)
	want := describe(genesis.Flags())
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the tags would render a different surface than the command declares\n derived:\n  %s\n command:\n  %s",
			strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	if len(got) == 0 {
		t.Fatal("no flag was derived, so this test asserts nothing")
	}
	t.Logf("%d flags derived from tags match what the command declares by hand", len(got))
}

// describe renders a flag set as sorted "name type usage" lines.
func describe(fs *pflag.FlagSet) []string {
	var out []string
	fs.VisitAll(func(f *pflag.Flag) {
		out = append(out, f.Name+" "+f.Value.Type()+" "+f.Usage)
	})
	sort.Strings(out)
	return out
}
