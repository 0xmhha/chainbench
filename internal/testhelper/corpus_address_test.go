package testhelper

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"github.com/0xmhha/chainbench/internal/preset"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

// updateGolden rewrites the record instead of checking it. Run with
// `go test ./internal/testhelper -run TestCorpus_Addresses -update` when an
// address change is intended, and read the diff before committing it.
var updateGolden = flag.Bool("update", false, "rewrite the corpus address record")

const (
	corpusDir     = "../../tests/tc"
	presetKeysDir = "../../presets/keys"
	addressGolden = "testdata/corpus-addresses.golden"
)

// TestCorpus_AddressesResolveToWhatTheyDidBefore is the guard for replacing the
// addresses in the committed cases with names.
//
// The substitution is invisible to everything else that checks this corpus. A
// compose plan carries no steps, so it cannot see an address at all; validate
// resolves no accounts, so a name that means the wrong thing still parses. Both
// ways of getting it wrong — picking the wrong contract name, or putting a name
// where the resolver does not reach — end with the case addressing something
// else and nothing saying so.
//
// So the resolved value of every address position is recorded, and a change to
// the record is the finding. The record also keeps its worth afterwards: it is
// what this corpus addresses, and a typo'd label or contract name fails here
// rather than against a running chain.
//
// The values are a BASELINE, not a prediction. They are resolved against
// presets/keys for every case, while a case whose env generates its keys gets
// different addresses at run time. That does not weaken the comparison — the
// same key set is used on both sides, so a line that moves means the name
// changed meaning, which is the only question being asked.
func TestCorpus_AddressesResolveToWhatTheyDidBefore(t *testing.T) {
	ring := presetRing(t)
	var lines []string

	err := filepath.WalkDir(corpusDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") || strings.HasSuffix(p, ".env.json") {
			return err
		}
		raws, rerr := dsl.ReadFiles([]string{p})
		if rerr != nil {
			return fmt.Errorf("%s: %w", p, rerr)
		}
		spec, perr := dsl.Parse(raws[0])
		if perr != nil {
			return fmt.Errorf("%s: %w", p, perr)
		}
		deps := &interp.Deps{Keys: ring, Contracts: contractsOf(spec.Chain.Name)}

		var doc any
		if uerr := json.Unmarshal(raws[0], &doc); uerr != nil {
			return fmt.Errorf("%s: %w", p, uerr)
		}
		rel := strings.TrimPrefix(filepath.ToSlash(p), corpusDir+"/")
		found, ferr := addressesIn(deps, doc, declaredLabels(spec))
		if ferr != nil {
			return fmt.Errorf("%s: %w", p, ferr)
		}
		for _, f := range found {
			lines = append(lines, rel+"\t"+f)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"

	if *updateGolden {
		if werr := os.WriteFile(addressGolden, []byte(got), 0o644); werr != nil {
			t.Fatal(werr)
		}
		t.Logf("rewrote %s with %d positions", addressGolden, len(lines))
		return
	}
	want, rerr := os.ReadFile(addressGolden)
	if rerr != nil {
		t.Fatalf("%s is missing; regenerate it with -update and read the diff: %v", addressGolden, rerr)
	}
	if got != string(want) {
		t.Errorf("the corpus addresses something other than what %s records.\n%s",
			addressGolden, firstDifference(string(want), got))
	}
}

// presetRing loads the committed key set, which is what a label resolves
// against. It needs no network.
func presetRing(t *testing.T) *store.KeySet {
	t.Helper()
	set, err := preset.LoadKeyPresetWithKeys(presetKeysDir)
	if err != nil {
		t.Fatalf("load %s: %v", presetKeysDir, err)
	}
	ring := store.NewKeySet(presetKeysDir)
	if err := ring.Register(context.Background(), set, len(set.Nodes)); err != nil {
		t.Fatalf("register key set: %v", err)
	}
	return ring
}

// contractsOf is the chain's contract table, or nil for a chain that declares
// none — the same answer the run gets.
func contractsOf(chain string) map[string]string {
	p, err := registry.Get(chain)
	if err != nil {
		return nil
	}
	return p.Manifest().SystemContracts
}

// stepKeys mark a map as a step. Only steps are resolved, because only steps
// are what the resolver is handed at run time.
var stepKeys = []string{"do", "expect"}

// opaqueKeys are the values the resolver never walks into, so neither does this.
//
// "params", "is" and "expected" used to be here, on the ground that an address
// inside raw JSON-RPC stays a literal. That stopped being true when the
// comparison side was opened to labels, and a record that still skipped them
// would not have noticed the substitution it exists to check.
var opaqueKeys = []string{"args", "env", "hooks"}

// addressesIn resolves every address position in the document and returns them
// as "<step path>\t<key>\t<address>" lines.
func addressesIn(d *interp.Deps, node any, declared map[string]bool) ([]string, error) {
	var out []string
	var walk func(n any, path string) error
	walk = func(n any, path string) error {
		switch v := n.(type) {
		case map[string]any:
			if isStep(v) {
				resolved, err := resolveAddressArgs(d, withoutBindings(v, declared))
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				out = append(out, positionsOf(v, resolved, path, declared)...)
			}
			for _, k := range sortedKeys(v) {
				if contains(opaqueKeys, k) {
					continue
				}
				if err := walk(v[k], path+"."+k); err != nil {
					return err
				}
			}
		case []any:
			for i, e := range v {
				if err := walk(e, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(node, ""); err != nil {
		return nil, err
	}
	return out, nil
}

// bindingRef marks a value the interpreter substitutes before an action runs.
// The resolver never sees one, so neither does this walk: it records the
// reference verbatim instead, which still moves if the step stops using it.
func bindingRef(s string) bool { return strings.HasPrefix(s, "$") }

// declaredLabels are the accounts a case's own env creates. They exist only
// once a run has funded them, so they are recorded by name for the same reason
// binding references are.
func declaredLabels(s dsl.Spec) map[string]bool {
	out := make(map[string]bool, len(s.EnvAccounts))
	for label := range s.EnvAccounts {
		out[label] = true
	}
	return out
}

// deferred reports a value this walk records verbatim rather than resolving: a
// binding the interpreter fills in later, or an account the run creates.
func deferred(v any, declared map[string]bool) bool {
	s, ok := v.(string)
	return ok && (bindingRef(s) || declared[s])
}

// withoutBindings drops the address positions holding a deferred value, so the
// resolver is handed the same shape it gets at run time.
func withoutBindings(spec map[string]any, declared map[string]bool) map[string]any {
	out := make(map[string]any, len(spec))
	for k, v := range spec {
		if deferred(v, declared) {
			continue
		}
		if list, ok := v.([]any); ok {
			kept := make([]any, 0, len(list))
			for _, e := range list {
				if deferred(e, declared) {
					kept = append(kept, "")
					continue
				}
				kept = append(kept, e)
			}
			out[k] = kept
			continue
		}
		out[k] = v
	}
	return out
}

// positionsOf reports the resolved value of each address position a step
// carries, whether it was written as a name or as an address.
func positionsOf(raw, resolved map[string]any, path string, declared map[string]bool) []string {
	var out []string
	for _, key := range append(append([]string{}, signerArgs...), addressArgs...) {
		s, ok := raw[key].(string)
		if !ok || s == "" {
			continue
		}
		if deferred(s, declared) {
			out = append(out, fmt.Sprintf("%s\t%s\t%s", path, key, s))
			continue
		}
		out = append(out, fmt.Sprintf("%s\t%s\t%s", path, key, normalizeAddr(resolved[key])))
	}
	for _, key := range append(append([]string{}, addressListArgs...), valueArgs...) {
		v, ok := resolved[key]
		if !ok {
			continue
		}
		out = append(out, addressesUnder(v, path+"\t"+key)...)
	}
	return out
}

// addressesUnder records every address anywhere inside a value, keyed by the
// path that reaches it. The resolver walks lists and objects, so this does too:
// recording only the top level would miss the one inside the transaction object
// eth_createAccessList takes.
func addressesUnder(v any, path string) []string {
	switch t := v.(type) {
	case string:
		// An address, not merely something starting with 0x. These positions
		// also carry block numbers and calldata blobs, and recording those
		// would churn the file on every unrelated edit while saying nothing
		// about which account a name points at.
		if !addressLiteral.MatchString(t) || len(t) != 42 {
			return nil
		}
		return []string{path + "\t" + normalizeAddr(t)}
	case []any:
		var out []string
		for i, e := range t {
			out = append(out, addressesUnder(e, fmt.Sprintf("%s[%d]", path, i))...)
		}
		return out
	case map[string]any:
		var out []string
		for _, k := range sortedKeys(t) {
			out = append(out, addressesUnder(t[k], path+"."+k)...)
		}
		return out
	default:
		return nil
	}
}

// normalizeAddr lowercases an address so the record compares identity rather
// than spelling. A case may write 0x…B00003 where the manifest writes
// 0x…b00003; the two are the same address, and a record that called them
// different would report every substitution as a change.
func normalizeAddr(v any) string {
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	if addressLiteral.MatchString(s) {
		return strings.ToLower(s)
	}
	return s
}

func isStep(m map[string]any) bool {
	for _, k := range stepKeys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// firstDifference names the first line that moved, so a failure points at one
// case rather than at a wall of identical text.
func firstDifference(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("line %d:\n  recorded: %s\n  now:      %s\n(%d recorded, %d now)", i+1, wl, gl, len(w)-1, len(g)-1)
		}
	}
	return "(lengths differ with no differing line)"
}

// TestCorpus_ANamedContractIsNotWrittenAsAnAddress is the ratchet that keeps
// the addresses from coming back.
//
// It is deliberately not "no address literals anywhere". Plenty of addresses in
// this corpus have no name to use instead: the markers a case deploys for
// itself, the EVM's precompiles, an arbitrary recipient. What it forbids is
// narrower and exact — writing the address of a contract the chain DOES name,
// in a position where the name would have worked.
//
// The rule tightens on its own. Every contract added to a manifest brings its
// address under this check without the check changing, which is what makes it a
// ratchet rather than a list someone has to remember to extend.
func TestCorpus_ANamedContractIsNotWrittenAsAnAddress(t *testing.T) {
	err := filepath.WalkDir(corpusDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") || strings.HasSuffix(p, ".env.json") {
			return err
		}
		raws, rerr := dsl.ReadFiles([]string{p})
		if rerr != nil {
			return fmt.Errorf("%s: %w", p, rerr)
		}
		spec, perr := dsl.Parse(raws[0])
		if perr != nil {
			return fmt.Errorf("%s: %w", p, perr)
		}
		named := byAddress(contractsOf(spec.Chain.Name))
		if len(named) == 0 {
			return nil
		}
		var doc any
		if uerr := json.Unmarshal(raws[0], &doc); uerr != nil {
			return fmt.Errorf("%s: %w", p, uerr)
		}
		rel := strings.TrimPrefix(filepath.ToSlash(p), corpusDir+"/")
		for _, w := range writtenOutContracts(doc, named, "") {
			t.Errorf("%s%s", rel, w)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// byAddress inverts a contract table so a literal can be looked up.
func byAddress(contracts map[string]string) map[string]string {
	out := make(map[string]string, len(contracts))
	for name, addr := range contracts {
		out[strings.ToLower(addr)] = name
	}
	return out
}

// writtenOutContracts reports every address position holding the address of a
// contract the chain names.
func writtenOutContracts(node any, named map[string]string, path string) []string {
	var out []string
	switch v := node.(type) {
	case map[string]any:
		if isStep(v) {
			for _, key := range append(append([]string{}, signerArgs...), addressArgs...) {
				if s, ok := v[key].(string); ok {
					if name, found := named[strings.ToLower(s)]; found {
						out = append(out, fmt.Sprintf("%s %s: writes %s, which this chain calls %q", path, key, s, name))
					}
				}
			}
			for _, key := range addressListArgs {
				list, _ := v[key].([]any)
				for i, e := range list {
					s, ok := e.(string)
					if !ok {
						continue
					}
					if name, found := named[strings.ToLower(s)]; found {
						out = append(out, fmt.Sprintf("%s %s[%d]: writes %s, which this chain calls %q", path, key, i, s, name))
					}
				}
			}
		}
		for _, k := range sortedKeys(v) {
			if contains(opaqueKeys, k) {
				continue
			}
			out = append(out, writtenOutContracts(v[k], named, path+"."+k)...)
		}
	case []any:
		for i, e := range v {
			out = append(out, writtenOutContracts(e, named, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return out
}
