package nodeconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// dialectChains maps a dialect to the chains built from that generation. A
// spelling has to exist in every one of them, since the dialect claims to
// describe them all.
var dialectChains = map[string][]string{
	"geth114":       {"gstable", "gwbft"},
	"geth110-wemix": {"gwemix"},
}

// TestDialectSpellingsExistInTheBinaries holds the flag table to the binaries.
//
// The table is a promise: every key in it is a knob this tool says it can set.
// A spelling that is wrong, or a flag that generation does not have, turns that
// promise into a launch that fails at the node — after provisioning, after the
// datadir is initialized, with an error from a program the operator did not
// invoke. So the table is checked against what the binaries actually print,
// captured by scripts/chain-analysis/capture-cli.sh.
//
// Skips when the captures are absent so a checkout without them still builds;
// the captures are committed, so that is not the normal case.
func TestDialectSpellingsExistInTheBinaries(t *testing.T) {
	for _, d := range []Dialect{Geth114(), Geth110Wemix()} {
		for _, chain := range dialectChains[d.ID] {
			surface, ok := readSurface(t, chain)
			if !ok {
				t.Skipf("no captured surface for %s — run scripts/chain-analysis/capture-cli.sh", chain)
			}
			for key, spec := range d.flags {
				kind, present := surface[spec.name]
				if !present {
					t.Errorf("%s: dialect %s claims %q (key %q), but the binary does not have it",
						chain, d.ID, spec.name, key)
					continue
				}
				if kind != spec.boolean {
					got, want := "takes a value", "takes a value"
					if kind {
						got = "is boolean"
					}
					if spec.boolean {
						want = "is boolean"
					}
					t.Errorf("%s: %q %s in the binary, but the dialect says it %s — a value passed to a boolean flag is a launch failure",
						chain, spec.name, got, want)
				}
			}
		}
	}
}

// TestDialectsDoNotClaimFlagsTheOtherGenerationLacks is the same check read the
// other way: a flag only one generation has must not sit in the shared table,
// because the shared table is what the other generation inherits.
func TestDialectsDoNotClaimFlagsTheOtherGenerationLacks(t *testing.T) {
	wemix, ok := readSurface(t, "gwemix")
	if !ok {
		t.Skip("no captured surface for gwemix")
	}
	for key, spec := range Geth110Wemix().flags {
		if _, present := wemix[spec.name]; !present {
			t.Errorf("the wemix dialect keeps %q (key %q) which that binary does not have — it should be deleted from the inherited table",
				spec.name, key)
		}
	}
}

// flagDefinition matches a flag's definition line in a captured help surface:
// indentation, the flag, an optional "value", then the description column.
// Prose mentions of a flag inside another's description have a single space
// after them and are deliberately not matched — reading one of those as a
// definition is how --cache.gc was first recorded as boolean.
var flagDefinition = regexp.MustCompile(`^\s{4,}(--[a-zA-Z0-9._-]+)(\s+value)?\s{2,}\S`)

// readSurface returns flag name → is-boolean for a captured chain surface.
func readSurface(t *testing.T, chain string) (map[string]bool, bool) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "chain-analysis", chain, "cli-surface.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }()

	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if m := flagDefinition.FindStringSubmatch(sc.Text()); m != nil {
			out[m[1]] = m[2] == ""
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return out, len(out) > 0
}
