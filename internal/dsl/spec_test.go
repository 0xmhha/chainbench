package dsl_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/dsl"
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

func TestParse_Valid(t *testing.T) {
	s, err := dsl.Parse([]byte(validSpec))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.SchemaVersion != "1" || s.ID != "GOV-005" || s.Chain.Name != "wbft" || s.Chain.Binary != "go-wbft" {
		t.Fatalf("parsed fields wrong: %+v", s)
	}
	if len(s.Assertions) != 1 || s.Topology["bp"].(float64) != 7 {
		t.Fatalf("topology/assertions wrong: %+v", s)
	}
}

func TestParse_MissingRequired(t *testing.T) {
	cases := map[string]string{
		"schemaVersion": `{"id":"X","chain":{"name":"wbft","binary":"b"},"assertions":[{}]}`,
		"id":            `{"schemaVersion":"1","chain":{"name":"wbft","binary":"b"},"assertions":[{}]}`,
		"chain.name":    `{"schemaVersion":"1","id":"X","chain":{"binary":"b"},"assertions":[{}]}`,
		"binary":        `{"schemaVersion":"1","id":"X","chain":{"name":"wbft"},"assertions":[{}]}`,
		"assertions":    `{"schemaVersion":"1","id":"X","chain":{"name":"wbft","binary":"b"}}`,
	}
	for field, raw := range cases {
		_, err := dsl.Parse([]byte(raw))
		if err == nil {
			t.Fatalf("missing %s: expected error", field)
		}
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("missing %s: error %q should name the field", field, err)
		}
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	if _, err := dsl.Parse([]byte("{not json")); err == nil {
		t.Fatal("invalid JSON must error")
	}
}

func TestParse_UnsupportedSchemaVersion(t *testing.T) {
	raw := `{"schemaVersion":"99","id":"X","chain":{"name":"wbft","binary":"b"},"assertions":[{"assert":"True"}]}`
	_, err := dsl.Parse([]byte(raw))
	if err == nil {
		t.Fatal("unsupported schemaVersion must be rejected")
	}
	if !strings.Contains(err.Error(), "schemaVersion") || !strings.Contains(err.Error(), "99") {
		t.Fatalf("error should name the rejected version: %v", err)
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	s, _ := dsl.Parse([]byte(validSpec))
	cfg := nodeconfig.Values{"nodes.validators": "7", "chain.id": "111133"}

	fp1 := s.Fingerprint(cfg)
	fp2 := s.Fingerprint(cfg)
	if fp1 != fp2 {
		t.Fatalf("fingerprint not deterministic: %q vs %q", fp1, fp2)
	}
	if len(string(fp1)) != 64 {
		t.Fatalf("fingerprint len = %d, want 64 hex", len(string(fp1)))
	}
	// A different resolved config -> different fingerprint.
	if s.Fingerprint(nodeconfig.Values{"chain.id": "999"}) == fp1 {
		t.Fatal("different config must change fingerprint")
	}
	// A different placement -> different fingerprint.
	s2 := s
	s2.Placement = "remote"
	if s2.Fingerprint(cfg) == fp1 {
		t.Fatal("different placement must change fingerprint")
	}
}

func TestFingerprint_MapOrderIndependent(t *testing.T) {
	// Two configs with the same entries added in different order must hash the
	// same (json.Marshal sorts map keys).
	a := nodeconfig.Values{"x": "1", "y": "2", "z": "3"}
	b := nodeconfig.Values{"z": "3", "y": "2", "x": "1"}
	s, _ := dsl.Parse([]byte(validSpec))
	if s.Fingerprint(a) != s.Fingerprint(b) {
		t.Fatal("fingerprint must be independent of map insertion order")
	}
}

func TestGet_DotPath(t *testing.T) {
	s, _ := dsl.Parse([]byte(validSpec))

	if v, ok := s.Get("chain.name"); !ok || v != "wbft" {
		t.Fatalf("Get chain.name = %v ok=%v", v, ok)
	}
	if v, ok := s.Get("topology.bp"); !ok || v.(float64) != 7 {
		t.Fatalf("Get topology.bp = %v ok=%v", v, ok)
	}
	if _, ok := s.Get("chain.missing"); ok {
		t.Fatal("missing path must return ok=false")
	}
	if _, ok := s.Get("nope.nope"); ok {
		t.Fatal("absent top-level path must return ok=false")
	}
}
