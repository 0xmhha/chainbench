package testhelper

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestDotPath_Wildcard: "*" maps the rest of the path over an array, so a
// case can ask whether an id is among a node's peers without knowing where
// in the unordered list it sits; indexes and "#" keep working as before.
func TestDotPath_Wildcard(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(`[{"id":"a","net":{"x":1}},{"id":"b"},{"name":"no id"}]`), &v); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]any{
		"*.id":    []any{"a", "b"},
		"*":       v,
		"1.id":    "b",
		"#":       "3",
		"*.net.x": []any{float64(1)},
	} {
		got, ok := dotPath(v, path)
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("dotPath(%q) = %v, %v; want %v", path, got, ok, want)
		}
	}
	if _, ok := dotPath(map[string]any{"a": 1}, "*"); ok {
		t.Error(`"*" on an object must not match — it maps over arrays only`)
	}
}
