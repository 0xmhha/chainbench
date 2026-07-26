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
		"network_id": 1, "miner_recommit": "duration",
		"bootstrap": {"type": "static"}, "consensus_family": "wbft"` + extra + `}`
}

func TestParseManifest_NewFields(t *testing.T) {
	m, err := ParseManifest([]byte(baseManifest(`,
		"upgrade": {"to_chain": "wbft", "at_fork": "croissant", "validator_source": "croissant_init"}`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.NetworkID != 1 || m.MinerRecommit != "duration" || m.Bootstrap.Type != "static" {
		t.Errorf("fields: %+v", m)
	}
	if m.Upgrade == nil || m.Upgrade.ToChain != "wbft" || m.Upgrade.AtFork != "croissant" {
		t.Errorf("upgrade: %+v", m.Upgrade)
	}
}

func TestParseManifest_Rejects(t *testing.T) {
	cases := []struct{ name, json, want string }{
		{"missing network_id",
			`{"id":"x","binary":"gx","chain_id":1,"miner_recommit":"nanos","bootstrap":{"type":"static"},"consensus_family":"poa"}`,
			"network_id"},
		{"bad miner_recommit",
			`{"id":"x","binary":"gx","chain_id":1,"network_id":1,"miner_recommit":"secs","bootstrap":{"type":"static"},"consensus_family":"poa"}`,
			"miner_recommit"},
		{"bad bootstrap type",
			`{"id":"x","binary":"gx","chain_id":1,"network_id":1,"miner_recommit":"nanos","bootstrap":{"type":"magic"},"consensus_family":"poa"}`,
			"bootstrap.type"},
		{"incomplete upgrade",
			baseManifest(`,"upgrade":{"to_chain":"wbft"}`),
			"upgrade requires"},
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
