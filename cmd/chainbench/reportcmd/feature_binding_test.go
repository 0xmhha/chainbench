package reportcmd_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0xmhha/chainbench/cmd/chainbench/reportcmd"
	"github.com/0xmhha/chainbench/internal/feature"
)

// TestReportFeatures_TagsMatchTheCommands is S1's check applied to the report
// stage: what the tags would render has to be what the commands already offer,
// so switching them over later is a deletion rather than a change of surface.
//
// One-directional, as before: every derived flag must exist on the command, and
// the command may carry more.
func TestReportFeatures_TagsMatchTheCommands(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"report.session": reportcmd.NewReport(),
		"report.log":     reportcmd.NewLog(),
	}
	checked := 0
	for name, cmd := range cmds {
		reg, ok := feature.Lookup(name)
		if !ok {
			t.Errorf("%s is not registered", name)
			continue
		}
		derived := pflag.NewFlagSet(name, pflag.ContinueOnError)
		if err := feature.Flags(reg.Input(), derived); err != nil {
			t.Errorf("%s: derive flags: %v", name, err)
			continue
		}
		derived.VisitAll(func(f *pflag.Flag) {
			checked++
			got := cmd.Flags().Lookup(f.Name)
			if got == nil {
				t.Errorf("%s declares --%s, which %s does not offer", name, f.Name, cmd.Name())
				return
			}
			if got.Value.Type() != f.Value.Type() {
				t.Errorf("%s --%s is %s in the tags and %s on the command", name, f.Name, f.Value.Type(), got.Value.Type())
			}
			if got.Usage != f.Usage {
				t.Errorf("%s --%s reads differently:\n  tags:    %q\n  command: %q", name, f.Name, f.Usage, got.Usage)
			}
			if got.DefValue != f.DefValue {
				t.Errorf("%s --%s defaults to %q in the tags and %q on the command", name, f.Name, f.DefValue, got.DefValue)
			}
		})
	}
	if checked == 0 {
		t.Fatal("no derived flag was compared, so this test asserts nothing")
	}
	t.Logf("%d derived flags across %d report features match the commands", checked, len(cmds))
}
