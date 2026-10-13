package app

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/testhelper"
)

func TestEditorTransactionQuantitiesRejectLossyNumbers(t *testing.T) {
	_, schemas, err := testhelper.EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"value", "gas", "nonce", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas"} {
		schema := schemas["action-sendTx"]["properties"].(map[string]any)[field].(map[string]any)
		for _, c := range []struct {
			value any
			valid bool
		}{
			{float64(1.5), false}, {float64(9007199254740992), false}, {float64(9007199254740991), true}, {"18446744073709551615", true}, {"0XFFFFFFFFFFFFFFFF", true}, {"$quantity", true}, {nil, false},
		} {
			raw, err := json.Marshal(c.value)
			if err != nil {
				t.Fatal(err)
			}
			validation := validateEditorSchema(raw, schema, schema)
			if (validation == nil) != c.valid {
				t.Errorf("%s %v: validation=%v, valid=%v", field, c.value, validation, c.valid)
			}
		}
	}
}
