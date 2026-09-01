package chaincmd_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/cmd/chainbench/chaincmd"
)

// TestChainCommandSurface pins the chain group's command set to the
// consolidation dictionary (plan section 3) plus the sanctioned operational
// verbs, so a command outside it — a parallel compose path, a name that drifts
// from the dictionary — fails here rather than shipping. Each name is
// categorized so the reason it is allowed is on the record.
func TestChainCommandSurface(t *testing.T) {
	// The seven compose stages of the dictionary (keys -> place -> enode ->
	// genesis -> config -> build -> deploy). Stage 8, run/alive, is init+start
	// (kept as atomic steps, C2) and up (the whole flow).
	dictionary := []string{"keys", "place", "enode", "genesis", "config", "build", "deploy"}
	bringUp := []string{"init", "start", "up"}
	// Operational verbs the C decision keeps under chain rather than forcing
	// into the dictionary.
	operational := []string{"new", "status", "show", "stop", "restart", "resume", "rm", "logs", "health"}

	allowed := map[string]string{}
	for _, n := range dictionary {
		allowed[n] = "compose stage"
	}
	for _, n := range bringUp {
		allowed[n] = "bring-up"
	}
	for _, n := range operational {
		allowed[n] = "operational"
	}

	got := map[string]bool{}
	for _, c := range chaincmd.New().Commands() {
		got[c.Name()] = true
	}

	// Every registered command must be sanctioned.
	var unsanctioned []string
	for name := range got {
		if _, ok := allowed[name]; !ok {
			unsanctioned = append(unsanctioned, name)
		}
	}
	sort.Strings(unsanctioned)
	if len(unsanctioned) > 0 {
		t.Errorf("chain has commands outside the dictionary + operational allowlist: %s\n"+
			"add each to the dictionary (and docs) or to the operational list, or remove it.",
			strings.Join(unsanctioned, ", "))
	}

	// Every dictionary compose stage must exist as a command — the surface may
	// not silently drop a stage.
	for _, name := range dictionary {
		if !got[name] {
			t.Errorf("dictionary compose stage %q has no chain command", name)
		}
	}
	// The renamed stages must be present under their new names, and the old
	// names gone (guards a half-done rename).
	for _, gone := range []string{"allocate", "launchopts", "provision", "net"} {
		if got[gone] {
			t.Errorf("retired command name %q is still registered", gone)
		}
	}
}
