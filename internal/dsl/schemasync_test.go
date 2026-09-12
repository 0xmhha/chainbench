package dsl

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// The schema/parser agreement is already guarded for field names, field types and
// the onEach shape (TestSchemaV2MatchesParsedFields, …ParsedTypes,
// …StatementOnEachIsArray). This file adds only what those do not cover, because
// a second copy of a check drifts from the first one.
//
// What they did not cover is where the schema was actually weaker than the parser:
// a do statement's expect adjunct was an unconstrained string. The parser
// restricts it to four outcomes, and a value outside the set would otherwise fall
// through to the default "must succeed" — the hole WA8 closed in the parser and
// left open in the document. Anything validating against this file (an editor,
// external tooling, the $schema reference) called a typo valid.

// schemaDefsRaw returns the schema's $defs with their properties, required list
// and strictness flag, which is the shape both tests below read.
func schemaDefsRaw(t *testing.T) map[string]struct {
	Properties           map[string]json.RawMessage `json:"properties"`
	Required             []string                   `json:"required"`
	AdditionalProperties *bool                      `json:"additionalProperties"`
} {
	t.Helper()
	var doc struct {
		Defs map[string]struct {
			Properties           map[string]json.RawMessage `json:"properties"`
			Required             []string                   `json:"required"`
			AdditionalProperties *bool                      `json:"additionalProperties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(SchemaV2, &doc); err != nil {
		t.Fatalf("the embedded schema does not parse: %v", err)
	}
	if len(doc.Defs) == 0 {
		t.Fatal("the embedded schema declares no $defs")
	}
	return doc.Defs
}

// TestSchemaV2ConstrainsTheExpectAdjunct is the drift this file was written for.
// A do statement's expect names an outcome, not an assertion, and a value outside
// the set turns a negative case positive. The parser refuses it; the document has
// to refuse it too, with the same words — so the enum is compared against
// expectAdjuncts rather than against a copy of the list.
func TestSchemaV2ConstrainsTheExpectAdjunct(t *testing.T) {
	raw, ok := schemaDefsRaw(t)["doStatement"].Properties["expect"]
	if !ok {
		t.Fatal("the schema's doStatement has no expect property")
	}
	var prop struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(raw, &prop); err != nil {
		t.Fatalf("doStatement.expect does not parse: %v", err)
	}
	if len(prop.Enum) == 0 {
		t.Fatal("doStatement.expect declares no enum, so the schema accepts an outcome the parser refuses")
	}
	want := make([]string, 0, len(expectAdjuncts))
	for k := range expectAdjuncts {
		want = append(want, k)
	}
	got := append([]string(nil), prop.Enum...)
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the schema's outcome vocabulary differs from the parser's\n  schema: %v\n  parser: %v", got, want)
	}
}

// TestSchemaV2IsAsStrictAsTheParser pins the two ways the document can be looser
// than the code even while every field name agrees.
//
// additionalProperties: the parser is strict, so a spec with an unknown key is
// refused. A schema that allows extra keys calls that spec valid, and the refusal
// then reads as a tooling bug rather than as the spec's own mistake.
//
// required: a field the parser rejects a spec for omitting has to be required
// here, or the document invites a spec that cannot run.
func TestSchemaV2IsAsStrictAsTheParser(t *testing.T) {
	defs := schemaDefsRaw(t)
	for def, wantRequired := range map[string][]string{
		"envSpec":  {"chain", "id", "kind", "schemaVersion"},
		"caseSpec": {"env", "id", "kind", "schemaVersion", "steps"},
	} {
		t.Run(def, func(t *testing.T) {
			d, ok := defs[def]
			if !ok {
				t.Fatalf("the schema has no $defs/%s", def)
			}
			if d.AdditionalProperties == nil || *d.AdditionalProperties {
				t.Errorf("additionalProperties must be false to match the strict parser")
			}
			got := append([]string(nil), d.Required...)
			sort.Strings(got)
			sort.Strings(wantRequired)
			if !reflect.DeepEqual(got, wantRequired) {
				t.Errorf("required = %v, want %v", got, wantRequired)
			}
		})
	}
}
