package testhelper

import "testing"

// TestComparator_InDelta pins WA16: compare:"InDelta" resolves to a tolerance
// comparison, reading the tolerance from the spec's delta (or tol) arg. Before
// the fix InDelta was defined but unreachable — assert.Lookup did not carry it.
func TestComparator_InDelta(t *testing.T) {
	cmp, ok := comparator("InDelta", map[string]any{"delta": 2})
	if !ok {
		t.Fatal("InDelta must resolve as a comparator")
	}
	if pass, detail := cmp(9, 10); !pass {
		t.Fatalf("9 within 2 of 10 should pass: %s", detail)
	}
	if pass, _ := cmp(7, 10); pass {
		t.Fatal("7 is not within 2 of 10; should fail")
	}

	// The "tol" alias works too.
	if cmp, ok := comparator("InDelta", map[string]any{"tol": 1}); !ok || func() bool { p, _ := cmp(10, 10); return !p }() {
		t.Fatal("tol alias: exact match within tolerance 1 should pass")
	}

	// A real assert primitive still resolves through the same helper.
	if _, ok := comparator("Equal", nil); !ok {
		t.Fatal("Equal must still resolve")
	}
	if _, ok := comparator("Nonexistent", nil); ok {
		t.Fatal("an unknown comparator must not resolve")
	}
}
