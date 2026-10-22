package nodeconfig

import (
	"strings"
	"testing"
)

// A node's log level has to be set when it starts: what happens in the first
// seconds (peers connecting and dropping, the consensus engine coming up) is
// gone by the time debug_verbosity can raise it over RPC.
func TestVerbosityIsALaunchOptionOnEveryDialect(t *testing.T) {
	for _, d := range []Dialect{Geth114(), Geth110Wemix()} {
		for _, level := range []string{"0", "4", "5"} {
			argv, err := New(d, Storage{DataDir: "/d"}).
				WithOverrides(Override{Key: KeyVerbosity, Value: level, Layer: LayerEnv}).
				Build()
			if err != nil {
				t.Fatalf("%s verbosity %s: %v", d.ID, level, err)
			}
			if got := String(argv); !strings.Contains(got, "--verbosity "+level) {
				t.Errorf("%s: argv has no --verbosity %s:\n%s", d.ID, level, got)
			}
		}
	}
}

// The binaries take 0 (silent) to 5 (detail); anything else fails at assembly
// instead of at a node that exits on start.
func TestVerbosityRejectsLevelsTheBinariesDoNotHave(t *testing.T) {
	for _, value := range []string{"-1", "6", "debug", ""} {
		_, err := New(Geth114(), Storage{DataDir: "/d"}).
			WithOverrides(Override{Key: KeyVerbosity, Value: value, Layer: LayerEnv}).
			Build()
		if err == nil {
			t.Errorf("verbosity %q was accepted", value)
		}
	}
}
