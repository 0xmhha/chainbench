package app

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/testhelper"
)

func TestEditorDurationControlsRejectInvalidValuesAndPreserveGoLiterals(t *testing.T) {
	_, schemas, err := testhelper.EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	fields := 0
	for name, definition := range schemas {
		for field, rawSchema := range definition["properties"].(map[string]any) {
			schema := rawSchema.(map[string]any)
			if schema["format"] != "duration" {
				continue
			}
			fields++
			for _, c := range []struct {
				value any
				valid bool
			}{
				{"0s", false}, {"-1ms", false}, {"", false}, {"invalid", false}, {true, false}, {nil, false}, {1.5, false}, {"9223372036854775808ns", false},
				{"1ns", true}, {"1.5ms", true}, {"250µs", true}, {"+2s", true}, {"1m0.5s", true}, {"9223372036854775807ns", true},
			} {
				value, _ := json.Marshal(c.value)
				err := validateEditorSchema(value, schema, schema)
				if (err == nil) != c.valid {
					t.Errorf("%s.%s %v: valid=%v error=%v", name, field, c.value, c.valid, err)
				}
			}
		}
	}
	if fields != 22 {
		t.Fatalf("duration fields changed without updating the runtime consumer audit: %d", fields)
	}
}
