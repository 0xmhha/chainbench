package app

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/testhelper"
)

func TestEditorDelegationFieldsAndConflicts(t *testing.T) {
	_, schemas, err := testhelper.EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"sendTx", "deployContract"} {
		schema := schemas["action-"+action]
		for _, c := range []struct {
			extra map[string]any
			valid bool
		}{
			{map[string]any{"feePayer": "sponsor"}, true},
			{map[string]any{"feePayerKey": "$key"}, true},
			{map[string]any{"feePayer": "sponsor", "maxFeePerGas": "3"}, false},
			{map[string]any{"feePayerKey": "$key", "maxPriorityFeePerGas": "1"}, false},
			{map[string]any{"feePayer": "sponsor", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, true},
			{map[string]any{"feePayer": ""}, false},
			{map[string]any{"feePayer": "sponsor", "feePayerKey": "$key"}, false},
			{map[string]any{"feePayer": "sponsor", "gasPrice": "3"}, false},
			{map[string]any{"feePayerKey": "$key", "gasPrice": "3"}, false},
		} {
			value := map[string]any{"do": action, "bytecode": "0x60006000f3"}
			if action == "sendTx" {
				delete(value, "bytecode")
			}
			for k, v := range c.extra {
				value[k] = v
			}
			raw, _ := json.Marshal(value)
			err := validateEditorSchema(raw, schema, schema)
			if (err == nil) != c.valid {
				t.Errorf("%s %v: err=%v valid=%v", action, c.extra, err, c.valid)
			}
		}
	}
}
