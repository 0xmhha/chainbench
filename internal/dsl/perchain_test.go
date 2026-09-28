package dsl

import "testing"

// The per-chain expectation, checked where it is resolved.
//
// It is resolved at lowering because that is where "is" becomes "expected",
// one place; the five sites that read "expected" afterwards never learn that a
// chain was involved. These hold that resolution to its rules.

// stmt lowers one statement on the named chain and returns its expected value.
func stmt(t *testing.T, chain string, m map[string]any) (any, bool) {
	t.Helper()
	st, err := lowerStatement(m, chain)
	if err != nil {
		t.Fatalf("lowerStatement on %s: %v", chain, err)
	}
	v, ok := st.Args["expected"]
	return v, ok
}

// TestIsPerChain_TheNamedChainGetsItsOwnAnswer.
//
// CT-FEE-002 is the case this exists for: a gas price under the floor is
// refused on StableNet, and on the WEMIX chains may sit unmined instead. One
// procedure, two right answers.
func TestIsPerChain_TheNamedChainGetsItsOwnAnswer(t *testing.T) {
	step := map[string]any{
		"expect": "txStatus", "hash": "$h",
		"is":         "0x1",
		"isPerChain": map[string]any{"wemix": "0x0"},
	}
	if got, ok := stmt(t, "wemix", step); !ok || got != "0x0" {
		t.Errorf("on wemix expected = %v (set %v), want 0x0", got, ok)
	}
	// A chain the map does not name keeps the default.
	if got, ok := stmt(t, "stablenet", step); !ok || got != "0x1" {
		t.Errorf("on stablenet expected = %v (set %v), want 0x1", got, ok)
	}
}

// TestIsPerChain_TheKeyDoesNotSurviveLowering.
//
// Whatever reads the statement afterwards knows only "expected". A leftover
// isPerChain would travel on as an argument and reach an assertion that has no
// idea what to do with it.
func TestIsPerChain_TheKeyDoesNotSurviveLowering(t *testing.T) {
	st, err := lowerStatement(map[string]any{
		"expect": "blockNumber", "is": "0x1",
		"isPerChain": map[string]any{"wemix": "0x2"},
	}, "wemix")
	if err != nil {
		t.Fatal(err)
	}
	if _, leaked := st.Args["isPerChain"]; leaked {
		t.Error("isPerChain reached the arguments")
	}
	if _, leaked := st.Args["is"]; leaked {
		t.Error("is reached the arguments; lowering renames it to expected")
	}
}

// TestIsPerChain_ItNeedsADefaultBesideIt.
//
// A statement that lists only some chains and no default would assert nothing
// on the rest and pass — which is the exact failure the per-chain expectation
// was added to stop. So it is refused instead of defaulted.
func TestIsPerChain_ItNeedsADefaultBesideIt(t *testing.T) {
	_, err := lowerStatement(map[string]any{
		"expect": "txStatus", "hash": "$h",
		"isPerChain": map[string]any{"wemix": "0x0"},
	}, "stablenet")
	if err == nil {
		t.Fatal("isPerChain without is was accepted")
	}
	if got := err.Error(); got == "" {
		t.Error("the refusal says nothing")
	}
}

// TestIsPerChain_AnEmptyOrMisshapenMapIsRefused: an empty object reads as "the
// answer differs" and then names no difference, which is a half-written step
// rather than a statement about any chain.
func TestIsPerChain_AnEmptyOrMisshapenMapIsRefused(t *testing.T) {
	for name, bad := range map[string]any{
		"empty object": map[string]any{},
		"a string":     "wemix",
		"a list":       []any{"wemix"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := lowerStatement(map[string]any{
				"expect": "blockNumber", "is": "0x1", "isPerChain": bad,
			}, "wemix")
			if err == nil {
				t.Fatalf("isPerChain = %v was accepted", bad)
			}
		})
	}
}

// TestIsPerChain_AStatementWithoutItIsUnchanged keeps the ordinary path honest:
// the great majority of statements name no per-chain answer, and lowering must
// treat them exactly as before.
func TestIsPerChain_AStatementWithoutItIsUnchanged(t *testing.T) {
	if got, ok := stmt(t, "wemix", map[string]any{"expect": "blockNumber", "is": "0x1"}); !ok || got != "0x1" {
		t.Errorf("expected = %v (set %v), want 0x1", got, ok)
	}
	// A statement that expects nothing still expects nothing.
	if _, ok := stmt(t, "wemix", map[string]any{"do": "waitBlock", "target": 5}); ok {
		t.Error("a do step with no is gained an expected value")
	}
}
