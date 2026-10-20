package testengine_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestPreparedSequenceRefusesMissingForwardAndDuplicateProducers(t *testing.T) {
	producer := dsl.Spec{ID: "prepare", Prepared: &dsl.PreparedV2{Exports: map[string]string{"contract": "$address"}}, Steps: []map[string]any{{"read": map[string]any{"source": "rpcCall", "method": "eth_coinbase", "save": "address"}}}}
	consumer := dsl.Spec{ID: "consume", Prepared: &dsl.PreparedV2{Imports: map[string]dsl.PreparedRef{"address": {Case: "prepare", Name: "contract"}}}, Assertions: []map[string]any{{"assert": "codeAt", "address": "$address", "expected": "0x6000"}}}
	for _, c := range []struct {
		name  string
		specs []dsl.Spec
	}{
		{"missing", []dsl.Spec{consumer}}, {"forward", []dsl.Spec{consumer, producer}}, {"duplicate", []dsl.Spec{producer, producer, consumer}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := testengine.Precheck(c.specs); err == nil || !strings.Contains(err.Error(), "prepared") {
				t.Fatalf("invalid preparation sequence passed preflight: %v", err)
			}
		})
	}
	if err := testengine.Precheck([]dsl.Spec{producer, consumer}); err != nil {
		t.Fatal("valid ordered preparation rejected", err)
	}
}
