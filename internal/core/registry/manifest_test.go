package registry

import (
	"strings"
	"testing"
)

// baseManifest is a minimal valid manifest JSON with a hole for the field under
// test, so validation cases stay focused.
func baseManifest(extra string) string {
	return `{
		"id": "x", "binary": "gx", "chain_id": 1,
		"miner_recommit": "duration", "dialect": "geth114",
		"bootstrap": {"type": "static"}, "consensus_family": "wbft"` + extra + `}`
}

func TestParseManifest_NewFields(t *testing.T) {
	m, err := ParseManifest([]byte(baseManifest(``)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.MinerRecommit != "duration" || m.Bootstrap.Type != "static" {
		t.Errorf("fields: %+v", m)
	}
	// An unset devp2p id is the ordinary case and means "it is the chain id".
	if m.NetworkID != 0 {
		t.Errorf("network_id = %d, want it unset when the chain does not differ", m.NetworkID)
	}
}

func TestParseManifest_Rejects(t *testing.T) {
	cases := []struct{ name, json, want string }{
		{"a network_id that repeats the chain id",
			baseManifest(`,"network_id":1`),
			"already means"},
		{"a negative network_id",
			baseManifest(`,"network_id":-1`),
			"negative network_id"},
		{"bad miner_recommit",
			`{"id":"x","binary":"gx","chain_id":1,"dialect":"geth114","miner_recommit":"secs","bootstrap":{"type":"static"},"consensus_family":"poa"}`,
			"miner_recommit"},
		{"bad bootstrap type",
			`{"id":"x","binary":"gx","chain_id":1,"dialect":"geth114","miner_recommit":"nanos","bootstrap":{"type":"magic"},"consensus_family":"poa"}`,
			"bootstrap.type"},
		{"missing dialect",
			`{"id":"x","binary":"gx","chain_id":1,"miner_recommit":"nanos","bootstrap":{"type":"static"},"consensus_family":"poa"}`,
			"missing dialect"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(c.json))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}
