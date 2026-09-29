package dsl

import (
	"strings"
	"testing"
)

// A do step's outcome can differ by chain, and both the map and the fallback
// have to be written.
//
// CT-FEE-001 and CT-FEE-002 are the cases that need it. A price under the base
// fee is refused at submit on StableNet and WEMIX3.0 and accepted but never
// mined on WEMIX4.0, and each chain's manner is definite — saying "either" for
// all three would be a weaker test, not a more tolerant one.

func TestExpectPerChain_NamedChainTakesItsOwnOutcome(t *testing.T) {
	st, err := lowerStatement(map[string]any{
		"do":             "sendTx",
		"expect":         "reject",
		"expectPerChain": map[string]any{"wbft": "keptOut"},
	}, "wbft")
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Args["expect"]; got != "keptOut" {
		t.Errorf("expect = %v, want the outcome the map names for this chain", got)
	}
	if _, left := st.Args[outcomePerChainKey]; left {
		t.Error("the map reached the runtime; it is resolved at lowering")
	}
}

func TestExpectPerChain_UnnamedChainTakesTheFallback(t *testing.T) {
	st, err := lowerStatement(map[string]any{
		"do":             "sendTx",
		"expect":         "reject",
		"expectPerChain": map[string]any{"wbft": "keptOut"},
	}, "stablenet")
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Args["expect"]; got != "reject" {
		t.Errorf("expect = %v, want the fallback for a chain the map does not name", got)
	}
}

// Without the fallback a chain the map forgets would expect nothing, and a step
// that expects nothing takes any outcome as success — the failure this whole
// mechanism exists to stop.
func TestExpectPerChain_NeedsItsFallbackWritten(t *testing.T) {
	_, err := lowerStatement(map[string]any{
		"do":             "sendTx",
		"expectPerChain": map[string]any{"wbft": "keptOut"},
	}, "stablenet")
	if err == nil || !strings.Contains(err.Error(), "expect") {
		t.Errorf("a map without its fallback was accepted: %v", err)
	}
}

// A per-chain outcome is held to the same vocabulary as a plain one, so a typo
// cannot fall through to "must succeed" on one chain only.
func TestExpectPerChain_HoldsTheOutcomeVocabulary(t *testing.T) {
	_, err := lowerStatement(map[string]any{
		"do":             "sendTx",
		"expect":         "reject",
		"expectPerChain": map[string]any{"wbft": "keptOutt"},
	}, "wbft")
	if err == nil || !strings.Contains(err.Error(), "keptOutt") {
		t.Errorf("a misspelled per-chain outcome was accepted: %v", err)
	}
}

// An empty map is a spec that meant to vary the outcome and said nothing, which
// is likelier a mistake than an intent.
func TestExpectPerChain_RefusesAnEmptyMap(t *testing.T) {
	_, err := lowerStatement(map[string]any{
		"do":             "sendTx",
		"expect":         "reject",
		"expectPerChain": map[string]any{},
	}, "wbft")
	if err == nil || !strings.Contains(err.Error(), outcomePerChainKey) {
		t.Errorf("an empty map was accepted: %v", err)
	}
}

// The vocabulary check ignores case, so an outcome whose name is written in
// mixed case is reachable. Keeping the keys canonical for the schema ratchet
// and lowering the written name cost keptOut exactly this: the check refused
// every case that named it.
func TestExpectAdjunct_IgnoresCase(t *testing.T) {
	for _, name := range []string{"keptOut", "KEPTOUT", "keptout", "Reject"} {
		if !isExpectAdjunct(name) {
			t.Errorf("%q is in the vocabulary but the check refused it", name)
		}
	}
	if isExpectAdjunct("kept out") {
		t.Error("a name outside the vocabulary was accepted")
	}
}
