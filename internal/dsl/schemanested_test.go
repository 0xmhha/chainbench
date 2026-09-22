package dsl

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The field-set check stopped at the top level, and one nested block rotted
// whole beneath it.
//
// TestSchemaV2MatchesParsedFields compares $defs.envSpec and $defs.caseSpec
// property by property against ChainPresetV2 and CaseV2. It never descends, so the
// "upgrade" object underneath went years without being compared to UpgradeV2:
// measured 2026-09-21 it required "profile" and "template", allowed nothing
// else, and carried none of fork, at, from, to, style or carry. Had the schema
// been enforced it would have rejected every upgrade declaration in the tree.
// Nothing failed because nothing looked.
//
// So the nested objects are paired with their Go types here, and the pairing is
// itself ratcheted: a new object with properties has to be declared before the
// suite goes green, which is what stops the next one rotting the same way.

// nestedObjects pairs each object-with-properties reachable from envSpec or
// caseSpec with the Go type that parses it. The key is the path the walk
// builds, so a rename on either side shows up as an unpaired object rather than
// as silence.
var nestedObjects = map[string]reflect.Type{
	"envSpec.keys":                           reflect.TypeOf(KeysV2{}),
	"envSpec.genesis":                        reflect.TypeOf(GenesisV2{}),
	"envSpec.upgrade":                        reflect.TypeOf(UpgradeV2{}),
	"envSpec.accounts[additionalProperties]": reflect.TypeOf(AccountV2{}),
	"envSpec.attach":                         reflect.TypeOf(AttachV2{}),
	"envSpec.genesis.perBinary[additionalProperties]": reflect.TypeOf(GenesisSideV2{}),
	"caseSpec.hooks": reflect.TypeOf(HooksV2{}),
}

// schemaObject is one object the walk found.
type schemaObject struct {
	path  string
	props map[string]bool
}

// walkObjects collects every object with a "properties" map below root.
func walkObjects(node any, path string, depth int, out *[]schemaObject) {
	m, ok := node.(map[string]any)
	if !ok {
		return
	}
	props, _ := m["properties"].(map[string]any)
	if props != nil && depth > 0 {
		set := map[string]bool{}
		for k := range props {
			set[k] = true
		}
		*out = append(*out, schemaObject{path: path, props: set})
	}
	for k, v := range props {
		walkObjects(v, path+"."+k, depth+1, out)
	}
	for _, k := range []string{"items", "additionalProperties"} {
		if v, ok := m[k]; ok {
			walkObjects(v, path+"["+k+"]", depth+1, out)
		}
	}
	for _, k := range []string{"oneOf", "anyOf", "allOf"} {
		if arr, ok := m[k].([]any); ok {
			for i, v := range arr {
				walkObjects(v, path+"."+k+itoa(i), depth+1, out)
			}
		}
	}
}

func itoa(i int) string { return string(rune('0' + i)) }

// jsonFields is the set of json names a struct declares.
func jsonFields(t reflect.Type) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		out[name] = true
	}
	return out
}

func TestSchemaV2NestedObjectsMatchTheirTypes(t *testing.T) {
	var doc struct {
		Defs map[string]json.RawMessage `json:"$defs"`
	}
	if err := json.Unmarshal(SchemaV2, &doc); err != nil {
		t.Fatalf("schema: %v", err)
	}
	var found []schemaObject
	for _, root := range []string{"envSpec", "caseSpec"} {
		raw, ok := doc.Defs[root]
		if !ok {
			t.Fatalf("schema has no $defs.%s, so the walk is wrong rather than the schema flat", root)
		}
		var node any
		if err := json.Unmarshal(raw, &node); err != nil {
			t.Fatalf("%s: %v", root, err)
		}
		walkObjects(node, root, 0, &found)
	}
	if len(found) == 0 {
		t.Fatal("no nested objects found, so the walk is wrong rather than the schema flat")
	}

	seen := map[string]bool{}
	for _, obj := range found {
		typ, ok := nestedObjects[obj.path]
		if !ok {
			t.Errorf("schema object %q is compared to nothing — pair it with the type that parses it in nestedObjects", obj.path)
			continue
		}
		seen[obj.path] = true
		fields := jsonFields(typ)
		for f := range fields {
			if !obj.props[f] {
				t.Errorf("%s: parser field %q has no schema property — add it to v2.schema.json", obj.path, f)
			}
		}
		for p := range obj.props {
			if !fields[p] {
				t.Errorf("%s: schema property %q has no parser field — remove it from v2.schema.json", obj.path, p)
			}
		}
	}
	// A pairing for an object the schema no longer has reads as coverage that
	// is not happening.
	var ghosts []string
	for path := range nestedObjects {
		if !seen[path] {
			ghosts = append(ghosts, path)
		}
	}
	sort.Strings(ghosts)
	if len(ghosts) > 0 {
		t.Errorf("nestedObjects names objects the schema does not have:\n  %s", strings.Join(ghosts, "\n  "))
	}
	t.Logf("%d nested objects compared", len(found))
}
