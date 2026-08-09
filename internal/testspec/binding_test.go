package testspec

import (
	"reflect"
	"testing"
)

func TestResolveRefs_WholeStringKeepsType(t *testing.T) {
	b := Bindings{"supply": "1000", "count": float64(7), "ok": true}
	cases := []struct {
		name string
		in   any
		want any
	}{
		{"string binding", "$supply", "1000"},
		{"number binding keeps float64", "$count", float64(7)},
		{"bool binding keeps bool", "$ok", true},
		{"no reference passes through", "plain", "plain"},
		{"escaped dollar is literal", "$$supply", "$supply"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRefs(tc.in, b)
			if err != nil {
				t.Fatalf("resolveRefs(%v): %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("resolveRefs(%v) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveRefs_BracedInterpolatesIntoString(t *testing.T) {
	b := Bindings{"addr": "0xabc", "n": float64(12)}
	got, err := resolveRefs("0x70a08231${addr}", b)
	if err != nil {
		t.Fatalf("resolveRefs: %v", err)
	}
	if got != "0x70a082310xabc" {
		t.Fatalf("got %q, want %q", got, "0x70a082310xabc")
	}
	// A number interpolates as its decimal form, not Go's float syntax.
	got, err = resolveRefs("block-${n}", b)
	if err != nil {
		t.Fatalf("resolveRefs: %v", err)
	}
	if got != "block-12" {
		t.Fatalf("got %q, want %q", got, "block-12")
	}
}

func TestResolveRefs_Nested(t *testing.T) {
	b := Bindings{"hash": "0xdead"}
	in := map[string]any{
		"assert":   "txStatus",
		"hash":     "$hash",
		"nested":   map[string]any{"deep": []any{"$hash", "literal"}},
		"expected": "0x1",
	}
	got, err := resolveRefs(in, b)
	if err != nil {
		t.Fatalf("resolveRefs: %v", err)
	}
	want := map[string]any{
		"assert":   "txStatus",
		"hash":     "0xdead",
		"nested":   map[string]any{"deep": []any{"0xdead", "literal"}},
		"expected": "0x1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestResolveRefs_DoesNotMutateInput(t *testing.T) {
	b := Bindings{"hash": "0xdead"}
	in := map[string]any{"hash": "$hash"}
	if _, err := resolveRefs(in, b); err != nil {
		t.Fatalf("resolveRefs: %v", err)
	}
	if in["hash"] != "$hash" {
		t.Fatalf("input mutated: %#v", in)
	}
}

func TestResolveRefs_UnboundIsAnError(t *testing.T) {
	if _, err := resolveRefs("$nope", Bindings{}); err == nil {
		t.Fatal("expected an error for an unbound reference")
	}
	if _, err := resolveRefs("x${nope}y", Bindings{}); err == nil {
		t.Fatal("expected an error for an unbound braced reference")
	}
}

func TestSaveName(t *testing.T) {
	if got := saveName(map[string]any{"save": "h"}); got != "h" {
		t.Fatalf("saveName = %q, want %q", got, "h")
	}
	if got := saveName(map[string]any{}); got != "" {
		t.Fatalf("saveName = %q, want empty", got)
	}
	if got := saveName(map[string]any{"save": 3}); got != "" {
		t.Fatalf("non-string save should be ignored, got %q", got)
	}
}

func TestRefNames(t *testing.T) {
	in := map[string]any{
		"a": "$one",
		"b": []any{"x${two}y", "plain", "$$notaref"},
		"c": map[string]any{"d": "$three"},
	}
	got := refNames(in)
	want := []string{"one", "three", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("refNames = %v, want %v", got, want)
	}
}
