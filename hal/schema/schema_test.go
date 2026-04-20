package schema

import (
	"path/filepath"
	"testing"
)

func TestNetworkSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"network-local.json",
		"network-remote.json",
		"network-hybrid.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("network", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestNetworkSchema_RejectsMissingChainType(t *testing.T) {
	doc := []byte(`{
		"name": "no-chain-type",
		"chain_id": 1,
		"nodes": [{"id":"n1","provider":"local","http":"http://127.0.0.1:8545"}]
	}`)
	if err := ValidateBytes("network", doc); err == nil {
		t.Fatal("expected validation error for missing chain_type")
	}
}
