package nodeconfig

import (
	"slices"
	"strings"
	"testing"
)

func TestOptionValuesAreCheckedAgainstWhatTheBinaryAccepts(t *testing.T) {
	ok := []struct {
		d Dialect
		k OptionKey
		v string
	}{
		{Geth114(), KeyHTTPPort, "8545"},
		{Geth114(), KeyPort, "0"},
		{Geth114(), KeySyncMode, "snap"},
		{Geth110Wemix(), KeySyncMode, "light"},
		{Geth110Wemix(), KeyConsensusMethod, "2"},
		{Geth114(), KeyMinerRecommit, "3s"},
		{Geth114(), KeyMinerRecommit, "1000000000"},
		{Geth114(), KeyNAT, "extip:10.0.0.1"},
		{Geth114(), KeyRPCTxFeeCap, "0.5"},
		{Geth114(), KeyHTTPAPI, "eth,net,web3"},
	}
	for _, c := range ok {
		if err := CheckValue(c.d, c.k, c.v); err != nil {
			t.Errorf("%s %s=%q refused: %v", c.d.ID, c.k, c.v, err)
		}
	}
	bad := []struct {
		d Dialect
		k OptionKey
		v string
	}{
		{Geth114(), KeyHTTPPort, "abc"},
		{Geth114(), KeyHTTPPort, "70000"},
		{Geth114(), KeyMaxPeers, "-1"},
		{Geth114(), KeySyncMode, "light"},
		{Geth114(), KeyGCMode, "partial"},
		{Geth110Wemix(), KeyConsensusMethod, "5"},
		{Geth114(), KeyMinerRecommit, "soon"},
		{Geth114(), KeyNAT, "extip:"},
		{Geth114(), KeyCacheGC, "150"},
	}
	for _, c := range bad {
		if err := CheckValue(c.d, c.k, c.v); err == nil {
			t.Errorf("%s %s=%q accepted", c.d.ID, c.k, c.v)
		}
	}
}

func TestBuilderRefusesAnOverrideOutsideItsRange(t *testing.T) {
	_, err := New(Geth114()).WithOverrides(Override{Key: KeyHTTPPort, Value: "abc", Layer: LayerEnv}).Build()
	if err == nil || !strings.Contains(err.Error(), "http.port") {
		t.Fatalf("an invalid port was assembled: %v", err)
	}
}

func TestOptionSchemasCarryTheSameConstraints(t *testing.T) {
	port := Geth114().OptionSchemas(true)[string(KeyHTTPPort)].(map[string]any)
	if port["x-minimum"] != int64(0) || port["x-maximum"] != int64(65535) || port["pattern"] == nil {
		t.Errorf("port schema %v", port)
	}
	sync := Geth110Wemix().OptionSchemas(false)[string(KeySyncMode)].(map[string]any)
	if enum, _ := sync["enum"].([]string); !slices.Contains(enum, "light") {
		t.Errorf("wemix syncmode schema %v", sync)
	}
	if enum, _ := Geth114().OptionSchemas(false)[string(KeySyncMode)].(map[string]any)["enum"].([]string); slices.Contains(enum, "light") {
		t.Error("geth114 offers light sync")
	}
	method := Geth110Wemix().OptionSchemas(false)[string(KeyConsensusMethod)].(map[string]any)
	if method["x-minimum"] != int64(1) || method["x-maximum"] != int64(4) {
		t.Errorf("consensusmethod schema %v", method)
	}
}
