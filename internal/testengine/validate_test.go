package testengine

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestSelectorWellFormed(t *testing.T) {
	cases := []struct {
		sel  string
		want bool
	}{
		{"node1", true},
		{"node12", true},
		{"bp", true},
		{"bp2", true},
		{"en1", true},
		{"pn", true},
		{"boot", true},
		{"validator", true}, // legacy spelling
		{"bp:any", true},
		{"en:0", true},
		{"", false},
		{"node0", false},  // 1-based
		{"node", false},   // needs an index
		{"nodeX", false},  // non-numeric
		{"xyz", false},    // unknown role
		{"bp:two", false}, // non-numeric suffix
		{"bp:-1", false},  // negative suffix
	}
	for _, tc := range cases {
		if got := selectorWellFormed(tc.sel); got != tc.want {
			t.Errorf("selectorWellFormed(%q) = %v, want %v", tc.sel, got, tc.want)
		}
	}
}

// caseJSON is a v2 case with an inline env, so it parses without resolving an
// env reference. steps and on are spliced in.
func caseJSON(steps string) []byte {
	return []byte(`{"schemaVersion":"2","kind":"case","id":"c",` +
		`"env":{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet"},` +
		`"steps":[` + steps + `]}`)
}

func TestValidateContent(t *testing.T) {
	cases := []struct {
		name       string
		raw        []byte
		wantOK     bool
		wantSubstr string
	}{
		{"well-formed", caseJSON(`{"do":"waitBlock","target":1},{"expect":"chainId","compare":"Greater","is":"0"}`), true, "OK"},
		{"typo action", caseJSON(`{"do":"waitBlok","target":1},{"expect":"chainId","is":"0"}`), false, "UNRESOLVED"},
		{"malformed selector", caseJSON(`{"do":"waitBlock","target":1,"on":"node0"},{"expect":"chainId","is":"0"}`), false, "INVALID SELECTOR"},
		{"unknown assertion", caseJSON(`{"do":"waitBlock","target":1},{"expect":"nosuchcheck","is":"0"}`), false, "UNRESOLVED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ValidateContent([][]byte{tc.raw}, []string{tc.name}, "")
			if err != nil {
				t.Fatalf("ValidateContent: %v", err)
			}
			if res[0].OK != tc.wantOK {
				t.Fatalf("OK = %v, want %v (result %q)", res[0].OK, tc.wantOK, res[0].Result)
			}
			if !strings.Contains(res[0].Result, tc.wantSubstr) {
				t.Fatalf("result = %q, want substring %q", res[0].Result, tc.wantSubstr)
			}
		})
	}
}

// TestValidateContent_ApplicableChainsSkip: a case that does not apply to the
// target chain is SKIP, not invalid.
func TestValidateContent_ApplicableChainsSkip(t *testing.T) {
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"c",` +
		`"env":{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet"},` +
		`"applicableChains":"wbft",` +
		`"steps":[{"do":"waitBlock","target":1},{"expect":"chainId","is":"0"}]}`)
	res, err := ValidateContent([][]byte{raw}, []string{"c"}, "stablenet")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if !res[0].OK || !strings.Contains(res[0].Result, "SKIP") {
		t.Fatalf("applicableChains mismatch should be OK+SKIP, got OK=%v result=%q", res[0].OK, res[0].Result)
	}
}

// TestPrecheck_FailsBeforeCompose: Precheck rejects a bad spec so the run path
// never composes it.
func TestPrecheck_FailsBeforeCompose(t *testing.T) {
	good, err := dsl.Parse(caseJSON(`{"do":"waitBlock","target":1},{"expect":"chainId","is":"0"}`))
	if err != nil {
		t.Fatalf("parse good: %v", err)
	}
	if err := Precheck([]dsl.Spec{good}); err != nil {
		t.Fatalf("Precheck(good) = %v, want nil", err)
	}
	badSel, err := dsl.Parse(caseJSON(`{"do":"waitBlock","target":1,"on":"node0"},{"expect":"chainId","is":"0"}`))
	if err != nil {
		t.Fatalf("parse badSel: %v", err)
	}
	if err := Precheck([]dsl.Spec{badSel}); err == nil {
		t.Fatal("Precheck(malformed selector) = nil, want error")
	}
}
