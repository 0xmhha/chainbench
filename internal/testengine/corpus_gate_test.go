package testengine_test

import (
	"encoding/json"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// corpusRoot is the committed case tree. The rule below is about those
// documents, not about this package, but there is nowhere else a test reads the
// whole corpus and this is where the engine's own corpus tests live.
const corpusRoot = "../../tests/tc"

// systemContractLow and systemContractHigh bound the addresses a chain puts its
// own contracts at.
//
// The low bound skips the precompiles (0x01 through 0x0a): those are the EVM's,
// every chain has them, and a case calling one says nothing about which chain it
// is on. The first survey of this rule counted 0x…0001 as a system contract and
// reported one more offender than there was.
var (
	systemContractLow  = big.NewInt(0x0a)
	systemContractHigh = big.NewInt(0x1000000)
)

// TestCorpus_ACaseNamingASystemContractSaysWhereItRuns.
//
// A chain's own contracts live at fixed low addresses, and which contract sits
// at which address is that chain's decision. A case that writes one of those
// addresses has named a fact about one chain, so it has to say which — otherwise
// it claims to run everywhere and breaks on the second chain that tries.
//
// Nine cases were in exactly that state, and nothing caught them because until
// the declaration could be swapped at run time nobody ran a stablenet case
// against wbft. This is the guard for that.
func TestCorpus_ACaseNamingASystemContractSaysWhereItRuns(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir(corpusRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") || strings.HasSuffix(p, ".env.json") {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		var doc map[string]any
		if json.Unmarshal(raw, &doc) != nil {
			return nil
		}
		if kind, _ := doc["kind"].(string); kind != "case" {
			return nil
		}
		if chains, _ := doc["applicableChains"].(string); strings.TrimSpace(chains) != "" {
			return nil
		}
		if addr := findSystemContract(doc); addr != "" {
			offenders = append(offenders, p+" names "+addr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range offenders {
		t.Errorf("%s but declares no applicableChains", o)
	}
}

// findSystemContract returns the first system-contract address the document
// carries, or "".
//
// It walks the decoded document rather than matching the raw text: event topics,
// four-byte selectors padded to a word, and deployment bytecode all contain runs
// of 40 hex digits, and matching text counts those as addresses.
func findSystemContract(node any) string {
	switch n := node.(type) {
	case map[string]any:
		for _, k := range sortedKeys(n) {
			if a := findSystemContract(n[k]); a != "" {
				return a
			}
		}
	case []any:
		for _, v := range n {
			if a := findSystemContract(v); a != "" {
				return a
			}
		}
	case string:
		if isSystemContract(n) {
			return n
		}
	}
	return ""
}

// isSystemContract reports whether s is an address in the range a chain puts its
// own contracts at. The 0x…c0ffee** addresses are excluded: those are what the
// cases deploy for themselves, and they belong to the test rather than to a
// chain.
func isSystemContract(s string) bool {
	if len(s) != 42 || !strings.HasPrefix(s, "0x") {
		return false
	}
	v, ok := new(big.Int).SetString(s[2:], 16)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(s), "c0ffee") {
		return false
	}
	return v.Cmp(systemContractLow) > 0 && v.Cmp(systemContractHigh) < 0
}

// sortedKeys keeps the reported offender stable across runs; map order is not.
func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
