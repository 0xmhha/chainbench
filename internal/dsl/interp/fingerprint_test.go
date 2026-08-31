package interp_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
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
