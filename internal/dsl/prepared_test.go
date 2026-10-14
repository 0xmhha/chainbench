package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestPreparedDeclarationsValidateAndMigrateWithoutChangingComposition(t *testing.T) {
	base := map[string]any{"schemaVersion": "2", "kind": "case", "id": "consumer", "chainPreset": map[string]any{"chain": "wbft"}, "steps": []any{map[string]any{"expect": "blockNumber", "is": 1}}}
	for _, p := range []map[string]any{
		{"imports": map[string]any{"bad-alias": map[string]any{"case": "prepare", "name": "address"}}},
		{"imports": map[string]any{"address": map[string]any{"case": "consumer", "name": "address"}}},
		{"imports": map[string]any{"address": map[string]any{"case": "prepare", "name": ""}}},
		{"exports": map[string]any{"address": "prefix-$address"}},
		{"exports": map[string]any{"address": "$address.field"}},
	} {
		base["prepared"] = p
		raw, _ := json.Marshal(base)
		if _, err := dsl.Parse(raw); err == nil {
			t.Fatal("invalid prepared declaration accepted", p)
		}
	}
	legacy := []byte(`{"schemaVersion":"1","id":"prepare","chain":{"name":"wbft","binary":"gwbft"},"prepared":{"exports":{"address":"${address}"}},"steps":[{"read":{"source":"rpcCall","method":"eth_coinbase","save":"address"}}],"assertions":[{"assert":"blockNumber","expected":1}]}`)
	v2, err := dsl.MigrateV1(legacy)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := dsl.Parse(v2)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Prepared == nil || spec.Prepared.Exports["address"] != "${address}" || spec.Chain.Binary != "gwbft" {
		t.Fatal("migration lost the shared declaration or composition", spec)
	}
}
