package interp_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

const validSpec = `{
  "schemaVersion": "1",
  "id": "GOV-005",
  "applicableChains": "wbft",
  "chain": {"name": "wbft", "binary": "go-wbft", "genesisOverlay": {"config": {"a": 1}}},
  "topology": {"bp": 7, "en": 5},
  "hardforks": {"croissant": 100, "brioche": 50},
  "placement": "local",
  "assertions": [{"on": "bp1", "assert": "Len", "expected": 7}]
}`

func TestFingerprint_Deterministic(t *testing.T) {
	s, _ := dsl.Parse([]byte(validSpec))
	cfg := nodeconfig.Values{"nodes.validators": "7", "chain.id": "111133"}

	fp1 := interp.Fingerprint(s, cfg)
	fp2 := interp.Fingerprint(s, cfg)
	if fp1 != fp2 {
		t.Fatalf("fingerprint not deterministic: %q vs %q", fp1, fp2)
	}
	if len(string(fp1)) != 64 {
		t.Fatalf("fingerprint len = %d, want 64 hex", len(string(fp1)))
	}
	// A different resolved config -> different fingerprint.
	if interp.Fingerprint(s, nodeconfig.Values{"chain.id": "999"}) == fp1 {
		t.Fatal("different config must change fingerprint")
	}
	// A different placement -> different fingerprint.
	s2 := s
	s2.Placement = "remote"
	if interp.Fingerprint(s2, cfg) == fp1 {
		t.Fatal("different placement must change fingerprint")
	}
}

func TestFingerprint_MapOrderIndependent(t *testing.T) {
	// Two configs with the same entries added in different order must hash the
	// same (json.Marshal sorts map keys).
	a := nodeconfig.Values{"x": "1", "y": "2", "z": "3"}
	b := nodeconfig.Values{"z": "3", "y": "2", "x": "1"}
	s, _ := dsl.Parse([]byte(validSpec))
	if interp.Fingerprint(s, a) != interp.Fingerprint(s, b) {
		t.Fatal("fingerprint must be independent of map insertion order")
	}
}

// TestFingerprint_HashesTheMergedValuesNotTheSharedEnv is the rule that keeps
// reuse honest once cases share an env.
//
// A shared env is resolved and merged before anything parses, so the
// fingerprint sees the values a case actually declared. Were it to key on the
// env's id instead, two cases extending one env with different overrides would
// share a key, and the second would reuse a network composed for the first —
// the wrong chain, reported as a reused one.
func TestFingerprint_HashesTheMergedValuesNotTheSharedEnv(t *testing.T) {
	const shared = `{"schemaVersion":"2","kind":"chain-preset","id":"base","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":4,"en":1}}`
	lookup := func(string) ([]byte, error) { return []byte(shared), nil }

	fp := func(t *testing.T, override string) session.Fingerprint {
		t.Helper()
		raw := []byte(`{"schemaVersion":"2","kind":"case","id":"c","chainPreset":` + override + `,
		  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`)
		inlined, err := dsl.InlineChainPreset(raw, lookup)
		if err != nil {
			t.Fatalf("inline: %v", err)
		}
		s, err := dsl.Parse(inlined)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		return interp.Fingerprint(s, nodeconfig.Values{})
	}

	base := fp(t, `{"extends":"base"}`)
	differs := fp(t, `{"extends":"base","topology":{"bp":7}}`)
	if base == differs {
		t.Error("two cases that declare different networks must not share a reuse key")
	}

	// And the other direction: the same declared values are the same key, no
	// matter whether they were written out or inherited.
	spelledOut := fp(t, `{"schemaVersion":"2","kind":"chain-preset","id":"base","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":4,"en":1}}`)
	if base != spelledOut {
		t.Error("inheriting values and writing them out must give the same reuse key")
	}
}
