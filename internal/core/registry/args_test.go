package registry_test

import (
	"math/big"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Every capability and MCP tool call decodes its arguments through these, and
// they were uncovered. Their failure mode is the quiet one: a value that does not
// decode becomes the DEFAULT, so an argument the caller gave is neither used nor
// refused — the tool runs, and does something else.
//
// The decoders cannot refuse; returning def is the contract, and the schema is
// what rejects a malformed argument. That makes what they DO understand the whole
// of their behaviour, and worth writing down.

func TestArgString(t *testing.T) {
	m := map[string]any{"s": "x", "n": 1.0}
	if got := registry.ArgString(m, "s", "def"); got != "x" {
		t.Errorf("ArgString = %q", got)
	}
	// A wrong type is the default, not a rendering of the value: "1" would be a
	// guess at what the caller meant.
	if got := registry.ArgString(m, "n", "def"); got != "def" {
		t.Errorf("a non-string became %q; it should be the default", got)
	}
	if got := registry.ArgString(m, "missing", "def"); got != "def" {
		t.Errorf("an absent key = %q", got)
	}
}

func TestArgInt(t *testing.T) {
	m := map[string]any{
		"json":   3.0, // a JSON number decodes as float64
		"native": 4,
		"str":    "5", // a numeric string is accepted on purpose
		"bad":    "abc",
		"bool":   true,
	}
	for _, c := range []struct {
		key  string
		want int
	}{{"json", 3}, {"native", 4}, {"str", 5}, {"bad", 9}, {"bool", 9}, {"missing", 9}} {
		if got := registry.ArgInt(m, c.key, 9); got != c.want {
			t.Errorf("ArgInt(%q) = %d, want %d", c.key, got, c.want)
		}
	}
}

// TestArgBool_UnderstandsTheSameSpellingsArgIntDoes is the asymmetry this file
// used to carry. ArgInt accepts a numeric string; ArgBool did not accept
// "true"/"false", so a caller sending "all": "true" got the default — the flag
// ignored rather than refused.
func TestArgBool_UnderstandsTheSameSpellingsArgIntDoes(t *testing.T) {
	m := map[string]any{
		"yes": true, "no": false,
		"strYes": "true", "strNo": "false",
		"loud": "TRUE", "padded": " true ",
		"num": 1.0, "empty": "", "other": "yep",
	}
	for _, c := range []struct {
		key  string
		def  bool
		want bool
	}{
		{"yes", false, true},
		{"no", true, false},
		{"strYes", false, true},
		{"strNo", true, false},
		{"loud", false, true},
		{"padded", false, true},
		// A number is not a boolean in JSON, and which way 2 or "" leans would be
		// invented rather than read.
		{"num", false, false},
		{"empty", true, true},
		{"other", true, true},
		{"missing", true, true},
	} {
		if got := registry.ArgBool(m, c.key, c.def); got != c.want {
			t.Errorf("ArgBool(%q, def=%v) = %v, want %v", c.key, c.def, got, c.want)
		}
	}
}

func TestArgStrings(t *testing.T) {
	m := map[string]any{
		"native": []string{"a", "b"},
		"json":   []any{"a", "b"},
		// A non-string element is skipped rather than failing the call: the
		// schema rejects a malformed argument, and panicking on one bad element
		// would turn a validation problem into a crash.
		"mixed": []any{"a", 2, "c"},
		"empty": []any{},
		"wrong": "not a list",
	}
	for _, c := range []struct {
		key  string
		want []string
	}{
		{"native", []string{"a", "b"}},
		{"json", []string{"a", "b"}},
		{"mixed", []string{"a", "c"}},
		{"empty", []string{}},
		{"wrong", nil},
		{"missing", nil},
	} {
		got := registry.ArgStrings(m, c.key)
		if len(got) != len(c.want) {
			t.Errorf("ArgStrings(%q) = %v, want %v", c.key, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("ArgStrings(%q)[%d] = %q, want %q", c.key, i, got[i], c.want[i])
			}
		}
	}
}

func TestArgBigInt(t *testing.T) {
	m := map[string]any{
		"dec":  "1000000000000000000",
		"zero": "0",
		"neg":  "-5",
		// Decimal only. A hex amount read as decimal, or the other way round,
		// moves a different sum than the caller asked for.
		"hex":   "0x10",
		"words": "lots",
		"num":   1.0, // not a string: ArgBigInt reads the string form only
	}
	if got := registry.ArgBigInt(m, "dec"); got == nil || got.Cmp(mustBig("1000000000000000000")) != 0 {
		t.Errorf("ArgBigInt(dec) = %v", got)
	}
	if got := registry.ArgBigInt(m, "zero"); got == nil || got.Sign() != 0 {
		t.Errorf("ArgBigInt(zero) = %v", got)
	}
	if got := registry.ArgBigInt(m, "neg"); got == nil || got.Sign() >= 0 {
		t.Errorf("ArgBigInt(neg) = %v; the decoder reads the sign, callers judge it", got)
	}
	for _, key := range []string{"hex", "words", "num", "missing"} {
		if got := registry.ArgBigInt(m, key); got != nil {
			t.Errorf("ArgBigInt(%q) = %v, want nil so the caller can tell it was not given", key, got)
		}
	}
}

func mustBig(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}
