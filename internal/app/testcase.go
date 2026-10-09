package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"github.com/0xmhha/chainbench/internal/testhelper"
)

type TestCaseContract struct {
	ContractVersion   string                       `json:"contractVersion"`
	GrammarContractID string                       `json:"grammarContractId"`
	Entries           []testhelper.VocabularyEntry `json:"entries"`
}

// DSLContract composes the grammar and builtin-owned parameter contracts.
func DSLContract() (TestCaseContract, map[string]any, error) {
	entries, params, err := testhelper.EditorVocabulary()
	if err != nil {
		return TestCaseContract{}, nil, err
	}
	var schema map[string]any
	if err = json.Unmarshal(dsl.SchemaV2, &schema); err != nil {
		return TestCaseContract{}, nil, err
	}
	defs := schema["$defs"].(map[string]any)
	for id, p := range params {
		defs[id] = p
	}
	// read and waitFor forward arguments to the selected reader. Each choice
	// exposes only that reader's arguments; changing source preserves the JSON
	// until validation so an imported value is never silently discarded.
	for _, action := range []string{"read", "waitFor"} {
		base := params["action-"+action]
		choices := []any{}
		for _, entry := range entries {
			if entry.Kind != "reader" {
				continue
			}
			reader := params["reader-"+entry.Name]
			properties := map[string]any{}
			for k, v := range reader["properties"].(map[string]any) {
				properties[k] = v
			}
			for k, v := range base["properties"].(map[string]any) {
				properties[k] = v
			}
			properties["source"] = map[string]any{"const": entry.Name}
			required := append(append([]string{}, base["required"].([]string)...), reader["required"].([]string)...)
			choice := map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
			if conditions, ok := reader["anyOf"]; ok {
				choice["anyOf"] = conditions
			}
			choices = append(choices, choice)
		}
		defs["action-"+action] = map[string]any{"oneOf": choices}
	}
	actions, assertions := []any{}, []any{}
	for _, entry := range entries {
		ref := map[string]any{"$ref": entry.SchemaRef}
		switch entry.Kind {
		case "action":
			actions = append(actions, ref)
		case "assertion":
			assertions = append(assertions, ref)
		}
	}
	// rpc is the parser's supported alias for rpcCall.
	alias := map[string]any{}
	for k, v := range params["assertion-rpcCall"] {
		alias[k] = v
	}
	aliasProps := map[string]any{}
	for k, v := range alias["properties"].(map[string]any) {
		aliasProps[k] = v
	}
	aliasProps["expect"] = map[string]any{"const": "rpc"}
	alias["properties"] = aliasProps
	defs["assertion-rpc"] = alias
	assertions = append(assertions, map[string]any{"$ref": "#/$defs/assertion-rpc"})
	defs["doStatement"] = map[string]any{"oneOf": actions}
	defs["expectStatement"] = map[string]any{"oneOf": assertions}
	legacyActions, legacyAssertions := []any{}, []any{}
	for _, entry := range entries {
		if entry.Kind == "reader" {
			continue
		}
		alternatives := []any{defs[entry.Kind+"-"+entry.Name]}
		if union, ok := alternatives[0].(map[string]any)["oneOf"].([]any); ok {
			alternatives = union
		}
		for _, alternative := range alternatives {
			old := alternative.(map[string]any)
			props := map[string]any{}
			for k, v := range old["properties"].(map[string]any) {
				if k == "do" || k == "is" || k == "isPerChain" || k == "expectPerChain" || (entry.Kind == "assertion" && k == "expect") {
					continue
				}
				props[k] = v
			}
			required := []string{}
			for _, k := range editorStrings(old["required"]) {
				if k != "do" && k != "expect" {
					required = append(required, k)
				}
			}
			arguments := map[string]any{"type": "object", "additionalProperties": false, "properties": props, "required": required}
			if conditions, ok := old["anyOf"]; ok {
				arguments["anyOf"] = conditions
			}
			if entry.Kind == "action" {
				legacyActions = append(legacyActions, map[string]any{"type": "object", "title": "do " + entry.Name, "additionalProperties": false, "required": []string{entry.Name}, "properties": map[string]any{entry.Name: arguments}})
			} else {
				props["assert"] = map[string]any{"const": entry.Name}
				// v1 writes the runtime spelling, and only where it is read.
				if _, compares := old["properties"].(map[string]any)["is"]; compares {
					props["expected"] = map[string]any{}
				}
				arguments["required"] = append(required, "assert")
				legacyAssertions = append(legacyAssertions, arguments)
			}
		}
	}
	defs["v1Do"] = map[string]any{"oneOf": legacyActions}
	defs["v1Assert"] = map[string]any{"oneOf": legacyAssertions}
	legacy := dsl.LegacyEditorSchema()
	legacyProps := legacy["properties"].(map[string]any)
	for _, field := range []string{"steps", "preActions", "postActions"} {
		legacyProps[field] = map[string]any{"type": "array", "items": map[string]any{"$ref": "#/$defs/v1Do"}}
	}
	legacyProps["assertions"] = map[string]any{"type": "array", "items": map[string]any{"$ref": "#/$defs/v1Assert"}}
	defs["v1Spec"] = legacy
	// Inline presets have optional headers in the parser; a shared preset has
	// the full envSpec header. Keep both forms in the editor contract.
	env := defs["envSpec"].(map[string]any)
	inline := map[string]any{}
	for k, v := range env {
		inline[k] = v
	}
	inline["required"] = []string{"chain"}
	defs["inlinePreset"] = inline
	defs["caseSpec"].(map[string]any)["properties"].(map[string]any)["chainPreset"] = map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"$ref": "#/$defs/inlinePreset"}}}
	return TestCaseContract{"2", "dsl", entries}, schema, nil
}

type TestCaseInput struct {
	Content json.RawMessage `json:"content"`
	// Presets are declarations supplied with the document, never filesystem paths.
	Presets map[string]json.RawMessage `json:"presets,omitempty"`
}
type TestCasePrepared struct {
	Content             json.RawMessage `json:"content"`
	Original            json.RawMessage `json:"original"`
	Migrated            bool            `json:"migrated"`
	SemanticFingerprint string          `json:"semanticFingerprint"`
	ContractVersion     string          `json:"contractVersion"`
	MigrationMessage    string          `json:"migrationMessage,omitempty"`
}

// PrepareTestCase imports or validates a document, keeping all declarations
// intact. v1 migration is accepted only if the complete executable projection
// is unchanged, including runtime-only environment fields.
func PrepareTestCase(in TestCaseInput) (TestCasePrepared, error) {
	content := append(json.RawMessage(nil), in.Content...)
	var probe map[string]any
	if err := json.Unmarshal(content, &probe); err != nil {
		return TestCasePrepared{}, err
	}
	legacyInput := !dsl.IsV2(content)
	migrated := false
	migrationMessage := ""
	var before dsl.Spec
	var err error
	if legacyInput {
		dec := json.NewDecoder(bytes.NewReader(content))
		dec.DisallowUnknownFields()
		if err = dec.Decode(new(dsl.Spec)); err != nil {
			return TestCasePrepared{}, fmt.Errorf("v1 import: %w", err)
		}
		if err = dec.Decode(new(any)); err != io.EOF {
			return TestCasePrepared{}, fmt.Errorf("expected one document")
		}
		before, err = dsl.Parse(content)
		if err != nil {
			return TestCasePrepared{}, err
		}
		candidate, migrationErr := dsl.MigrateV1(content)
		if migrationErr == nil {
			after, parseErr := dsl.Parse(candidate)
			if parseErr == nil && testCaseSemantics(after) == testCaseSemantics(before) {
				content = candidate
				migrated = true
			} else {
				migrationMessage = "Retained v1: migration would change executable declarations"
			}
		} else {
			migrationMessage = "Retained v1: this declaration has no equivalent v2 migration"
		}
	}
	_, schema, err := DSLContract()
	if err != nil {
		return TestCasePrepared{}, err
	}
	resolved, err := dsl.InlineChainPreset(content, func(id string) ([]byte, error) {
		raw, ok := in.Presets[id]
		if !ok {
			return nil, fmt.Errorf("missing preset %q", id)
		}
		return raw, nil
	})
	if err != nil {
		return TestCasePrepared{}, err
	}
	// Schema checks the structured parameter vocabulary, then the original
	// parser and precheck enforce cross-field constraints and references.
	defs := schema["$defs"].(map[string]any)
	caseSchema := defs["caseSpec"].(map[string]any)
	if !dsl.IsV2(content) {
		caseSchema = defs["v1Spec"].(map[string]any)
	}
	if err = validateEditorSchema(resolved, caseSchema, schema); err != nil {
		return TestCasePrepared{}, err
	}
	spec, err := dsl.Parse(resolved)
	if err != nil {
		return TestCasePrepared{}, err
	}
	if err = testengine.Precheck([]dsl.Spec{spec}); err != nil {
		return TestCasePrepared{}, err
	}
	fingerprint := testCaseSemantics(spec)
	if migrated && fingerprint != testCaseSemantics(before) {
		return TestCasePrepared{}, fmt.Errorf("v1 migration cannot preserve every engine declaration; import remains unchanged")
	}
	return TestCasePrepared{Content: content, Original: in.Content, Migrated: migrated, SemanticFingerprint: fingerprint, ContractVersion: "2", MigrationMessage: migrationMessage}, nil
}

func testCaseSemantics(s dsl.Spec) string {
	// v2 stores actions only in Sequence, while v1 stores them in Steps as well.
	// Normalize the duplicate representations, but include every exported field
	// (encoding Spec directly would omit runtime-only env fields).
	sequence := dsl.SequenceOf(s)
	s.Steps = nil
	s.Assertions = nil
	s.Sequence = nil
	fields := map[string]any{"Sequence": sequence}
	value := reflect.ValueOf(s)
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		v := value.Field(i)
		if v.IsZero() || ((v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.Len() == 0 && typ.Field(i).Name != "SkipsOn") {
			continue
		}
		fields[typ.Field(i).Name] = v.Interface()
	}
	for _, name := range []string{"EnvLaunch", "EnvConfig"} {
		if m, ok := fields[name].(map[string][]string); ok {
			copyMap := map[string][]string{}
			for k, values := range m {
				copyMap[k] = append([]string{}, values...)
				sort.Strings(copyMap[k])
			}
			fields[name] = copyMap
		}
	}
	raw, _ := json.Marshal(fields)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validateEditorSchema(raw []byte, schema, root map[string]any) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	return checkEditorValue(value, schema, root, "/content")
}
func editorStrings(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := []string{}
		for _, x := range s {
			if text, ok := x.(string); ok {
				out = append(out, text)
			}
		}
		return out
	}
	return nil
}
func checkEditorValue(value any, s, root map[string]any, path string) error {
	if ref, ok := s["$ref"].(string); ok {
		def, ok := root["$defs"].(map[string]any)[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: unresolved schema %s", path, ref)
		}
		return checkEditorValue(value, def, root, path)
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if choices, ok := s[key].([]any); ok {
			matches := 0
			var details, addressed []string
			for _, choice := range choices {
				err := checkEditorValue(value, choice.(map[string]any), root, path)
				if err == nil {
					matches++
					continue
				}
				details = append(details, err.Error())
				if editorChoiceNamed(value, choice.(map[string]any), root) {
					addressed = append(addressed, err.Error())
				}
			}
			if matches == 0 && len(addressed) == 1 {
				// The statement names its builtin (do, expect, source), so
				// the other builtins' complaints are noise.
				return errors.New(addressed[0])
			}
			if matches == 0 || (key == "oneOf" && matches != 1) {
				return fmt.Errorf("%s: no unique supported %s variant (%s)", path, key, strings.Join(details, "; "))
			}
		}
	}
	if all, ok := s["allOf"].([]any); ok {
		for _, part := range all {
			if err := checkEditorValue(value, part.(map[string]any), root, path); err != nil {
				return err
			}
		}
	}
	if not, ok := s["not"].(map[string]any); ok && checkEditorValue(value, not, root, path) == nil {
		return fmt.Errorf("%s: fields %s cannot be given together", path, strings.Join(editorStrings(not["required"]), " and "))
	}
	if c, exists := s["const"]; exists && !reflect.DeepEqual(value, c) {
		return fmt.Errorf("%s: expected %v", path, c)
	}
	if choices, ok := s["enum"]; ok {
		found := false
		for _, c := range editorStrings(choices) {
			if reflect.DeepEqual(value, c) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("%s: unsupported value", path)
		}
	}
	switch s["type"] {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s: expected string", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: expected boolean", path)
		}
	case "number", "integer":
		n, ok := value.(float64)
		if !ok || (s["type"] == "integer" && n != float64(int64(n))) {
			return fmt.Errorf("%s: expected %s", path, s["type"])
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("%s: expected object", path)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("%s: expected array", path)
		}
	}
	if text, ok := value.(string); ok {
		if pattern, ok := s["pattern"].(string); ok {
			re, err := regexp.Compile(pattern)
			if err != nil || !re.MatchString(text) {
				return fmt.Errorf("%s: invalid value", path)
			}
		}
		if min, ok := editorSchemaNumber(s["minLength"]); ok && float64(len(text)) < min {
			return fmt.Errorf("%s: empty value", path)
		}
		if s["format"] == "duration" {
			if d, err := time.ParseDuration(text); err != nil || d <= 0 {
				return fmt.Errorf("%s: expected positive duration", path)
			}
		}
	}
	if n, ok := value.(float64); ok {
		if min, ok := editorSchemaNumber(s["minimum"]); ok && n < min {
			return fmt.Errorf("%s: below minimum", path)
		}
		if max, ok := editorSchemaNumber(s["maximum"]); ok && n > max {
			return fmt.Errorf("%s: above maximum", path)
		}
	}
	if m, ok := value.(map[string]any); ok {
		if min, ok := editorSchemaNumber(s["minProperties"]); ok && float64(len(m)) < min {
			return fmt.Errorf("%s: too few properties", path)
		}
		props, _ := s["properties"].(map[string]any)
		for _, key := range editorStrings(s["required"]) {
			if _, exists := m[key]; !exists {
				return fmt.Errorf("%s/%s: required", path, key)
			}
		}
		for key, v := range m {
			if names, ok := s["propertyNames"].(map[string]any); ok {
				if err := checkEditorValue(key, names, root, path+"/"+key); err != nil {
					return err
				}
			}
			child, exists := props[key].(map[string]any)
			if !exists {
				switch a := s["additionalProperties"].(type) {
				case bool:
					if !a {
						return fmt.Errorf("%s/%s: unsupported field", path, key)
					}
				case map[string]any:
					child = a
				}
			}
			if child != nil {
				if err := checkEditorValue(v, child, root, path+"/"+key); err != nil {
					return err
				}
			}
		}
	}
	if values, ok := value.([]any); ok {
		if min, ok := editorSchemaNumber(s["minItems"]); ok && float64(len(values)) < min {
			return fmt.Errorf("%s: too few items", path)
		}
		if s["uniqueItems"] == true {
			seen := map[string]bool{}
			for _, v := range values {
				raw, _ := json.Marshal(v)
				key := string(raw)
				if seen[key] {
					return fmt.Errorf("%s: duplicate items", path)
				}
				seen[key] = true
			}
		}
		if child, ok := s["items"].(map[string]any); ok {
			for i, v := range values {
				if err := checkEditorValue(v, child, root, fmt.Sprintf("%s/%d", path, i)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// editorChoiceNamed reports whether value carries every constant a choice
// discriminates on (its "do", "expect" or "source"), so the choice is the one
// the author meant even though it failed.
func editorChoiceNamed(value any, choice, root map[string]any) bool {
	if ref, ok := choice["$ref"].(string); ok {
		def, ok := root["$defs"].(map[string]any)[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
		if !ok {
			return false
		}
		choice = def
	}
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if nested, ok := choice["oneOf"].([]any); ok {
		for _, c := range nested {
			if editorChoiceNamed(value, c.(map[string]any), root) {
				return true
			}
		}
		return false
	}
	props, _ := choice["properties"].(map[string]any)
	named := false
	for key, prop := range props {
		c, ok := prop.(map[string]any)["const"]
		if !ok {
			continue
		}
		if !reflect.DeepEqual(m[key], c) {
			return false
		}
		named = true
	}
	return named
}

func editorSchemaNumber(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
