package arch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// envDir holds the shared chain declarations the test cases extend.
const envDir = "../../tests/tc/env"

// twoChainEnvs is every env that names a chain besides its own, and so cannot
// be run against a different chain.
//
// Why this is pinned. A chain declaration is a network SHAPE plus a pointer to
// the chain it runs on: measured 2026-09-20, the stablenet, wbft and wemix bp4
// declarations differ only in "id" and "chain", because everything chain-specific
// already lives in internal/chains/<id>/manifest.json. That is what lets one
// common test definition run against three chains — swap the pointer, keep the
// shape.
//
// A handoff env breaks that. It declares one chain and a second binary that runs
// ANOTHER chain, because crossing a fork from one build to another is the thing
// under test. There is no chain to swap it to: the pair IS the subject. So the
// refusal belongs here, to the declaration, and not to the cases that extend it.
//
// The set is derived, never declared — an env is a handoff exactly when one of
// its binaries names a different chain, which the document already says. A new
// entry here means someone added a two-chain declaration, and whoever adds the
// chain-swap flag has to decide what it does with this one.
var twoChainEnvs = []string{
	"wemix-to-wbft",
	"wemix-to-wbft-bp2",
}

func TestTwoChainEnvsAreListed(t *testing.T) {
	entries, err := os.ReadDir(envDir)
	if err != nil {
		t.Fatalf("read %s: %v", envDir, err)
	}
	var got []string
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".env.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(envDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		var env struct {
			ID       string                     `json:"id"`
			Chain    string                     `json:"chain"`
			Binaries map[string]json.RawMessage `json:"binaries"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		seen++
		if other := secondChain(t, env.Chain, env.Binaries); other != "" {
			got = append(got, env.ID)
			t.Logf("%s runs %s and %s", env.ID, env.Chain, other)
		}
	}
	if seen == 0 {
		t.Fatalf("no env files under %s, so the walk is wrong rather than the set empty", envDir)
	}
	slices.Sort(got)
	want := slices.Clone(twoChainEnvs)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("envs naming a second chain: got %v, want %v\n"+
			"a new one cannot take a chain swap — say what the swap does with it before adding it here", got, want)
	}
}

// secondChain reports the chain an env names besides its own. A binaries entry
// is either a bare name (this env's chain) or {binary, chain}.
func secondChain(t *testing.T, chain string, binaries map[string]json.RawMessage) string {
	t.Helper()
	for _, name := range slices.Sorted(mapKeys(binaries)) {
		var ref struct {
			Chain string `json:"chain"`
		}
		if err := json.Unmarshal(binaries[name], &ref); err != nil {
			continue // a bare string: this env's chain
		}
		if ref.Chain != "" && ref.Chain != chain {
			return ref.Chain
		}
	}
	return ""
}

func mapKeys(m map[string]json.RawMessage) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}
