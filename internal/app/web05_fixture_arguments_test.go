package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// This is a declaration guard, not execution evidence. WEB-05 additionally
// requires every field to occur in a fully recorded passing native Web job.
func TestWeb05FixtureArgumentsMatchServedContract(t *testing.T) {
	vocabulary, contract, err := DSLContract()
	if err != nil {
		t.Fatal(err)
	}
	definitions := contract["$defs"].(map[string]any)
	paths, err := filepath.Glob("../../tests/tc/go-stablenet/vocabulary/1[1-6]-arguments-*.json")
	if err != nil || len(paths) == 0 {
		t.Fatal("missing Web coverage fixture pool", err)
	}
	paths = append(paths, "../../tests/tc/go-wemix/hardfork/02-state-written-before-the-fork-survives-it.json")
	declared := map[string]map[string]bool{}
	add := func(kind, name string, step map[string]any) {
		key := kind + "-" + name
		if declared[key] == nil {
			declared[key] = map[string]bool{}
		}
		for field := range step {
			declared[key][field] = true
		}
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var spec struct{ Steps []map[string]any }
		if err := json.Unmarshal(data, &spec); err != nil {
			t.Fatal(err)
		}
		for i, step := range spec.Steps {
			if action, ok := step["do"].(string); ok {
				add("action", action, step)
				if action == "read" || action == "waitFor" {
					if reader, ok := step["source"].(string); ok {
						add("reader", reader, step)
					}
				}
			} else if assertion, ok := step["expect"].(string); ok {
				if assertion == "rpc" {
					assertion = "rpcCall"
				}
				add("assertion", assertion, step)
			} else {
				t.Fatalf("%s step%d: Web execution coverage requires flat recorded statements", path, i)
			}
		}
	}
	for _, entry := range vocabulary.Entries {
		t.Run(entry.Kind+":"+entry.Name, func(t *testing.T) {
			schema := definitions[strings.TrimPrefix(entry.SchemaRef, "#/$defs/")].(map[string]any)
			choices := []any{schema}
			if union, ok := schema["oneOf"].([]any); ok {
				choices = union
			}
			missing := map[string]bool{}
			for _, choice := range choices {
				for field, rule := range choice.(map[string]any)["properties"].(map[string]any) {
					if slices.Contains([]string{"do", "source", "isPerChain", "expectPerChain"}, field) {
						continue
					}
					if field == "expect" {
						if _, constant := rule.(map[string]any)["const"]; constant {
							continue
						}
					}
					if !declared[entry.Kind+"-"+entry.Name][field] {
						missing[field] = true
					}
				}
			}
			if len(missing) != 0 {
				var fields []string
				for field := range missing {
					fields = append(fields, field)
				}
				slices.Sort(fields)
				t.Errorf("Web native fixtures omit served argument fields: %v", fields)
			}
		})
	}
}
