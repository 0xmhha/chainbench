package app

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// The Go response structs are the source of the Web HTTP contract; this test
// fails when docs/dev/web-ui.openapi.yaml no longer describes them. It checks
// field names, which fields are always present, and the JSON kind of each.
var webContractSchemas = map[string]any{
	"Node":              WebNode{},
	"Network":           WebNetwork{},
	"Phase":             WebJobPhase{},
	"Job":               WebJob{},
	"Coverage":          WebCoverage{},
	"Metrics":           WebMetrics{},
	"Logs":              WebLogs{},
	"Run":               WebRun{},
	"RunPage":           WebRunPage{},
	"Document":          DeploymentDocument{},
	"Workspace":         DeploymentWorkspace{},
	"User":              WebUser{},
	"Session":           WebSession{},
	"Credential":        DeploymentCredential{},
	"Asset":             WebAsset{},
	"AuditRecord":       WebAuditRecord{},
	"AuditRecordPage":   WebAuditPage{},
	"Plan":              WebPlan{},
	"ResourceConflict":  WebResourceConflict{},
	"Comparison":        WebComparison{},
	"Vocabulary":        TestCaseContract{},
	"ChainSurface":      WebChainSurface{},
	"ChainSurfaceRef":   WebChainSurfaceRef{},
	"ChainCommand":      WebChainCommand{},
	"ChainOption":       WebChainOption{},
	"SurfaceSource":     WebSurfaceSource{},
	"SurfaceSupport":    WebSurfaceSupport{},
	"BinaryFingerprint": WebBinaryFingerprint{},
	"Validation":        DocumentValidation{},
	"ImportPreview":     DocumentImportPreview{},
	"ImportCommit":      DocumentImportCommit{},
	"ObservationEvent":  WebObservationEvent{},
}

// webInputOnly names fields a response struct inherits from its input struct
// but never fills, so the response schema rightly leaves them out.
var webInputOnly = map[string]bool{
	"DeploymentDocument.contractDigest": true, // checked on save, never stored
}

type openAPISchemas map[string]map[string]any

func loadOpenAPISchemas(t *testing.T) openAPISchemas {
	t.Helper()
	raw, err := os.ReadFile("../../docs/dev/web-ui.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas openAPISchemas `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Components.Schemas
}

func (s openAPISchemas) resolve(schema map[string]any) map[string]any {
	for i := 0; i < 8; i++ {
		ref, ok := schema["$ref"].(string)
		if !ok {
			break
		}
		// A ref may point inside a schema: #/components/schemas/Name/properties/x.
		parts := strings.Split(strings.TrimPrefix(ref, "#/components/schemas/"), "/")
		schema = s[parts[0]]
		for _, part := range parts[1:] {
			next, _ := schema[part].(map[string]any)
			schema = next
		}
	}
	parts, ok := schema["allOf"].([]any)
	if !ok {
		return schema
	}
	// allOf adds constraints: the union of the parts' properties and
	// required lists describes what the value must carry.
	merged := map[string]any{"properties": map[string]any{}, "required": []any{}}
	for _, part := range parts {
		p, _ := part.(map[string]any)
		p = s.resolve(p)
		for k, v := range p {
			switch k {
			case "properties":
				props, _ := v.(map[string]any)
				for name, prop := range props {
					merged["properties"].(map[string]any)[name] = prop
				}
			case "required":
				list, _ := v.([]any)
				merged["required"] = append(merged["required"].([]any), list...)
			default:
				merged[k] = v
			}
		}
	}
	return merged
}

func schemaTypes(schema map[string]any) []string {
	switch v := schema["type"].(type) {
	case string:
		return []string{v}
	case []any:
		out := []string{}
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

type jsonField struct {
	name   string
	typ    reflect.Type
	always bool
}

// jsonFields lists the fields encoding/json writes for a struct, following
// embedded structs the way the encoder does.
func jsonFields(t reflect.Type) []jsonField {
	out := []jsonField{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" || (!f.IsExported() && !f.Anonymous) {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
			out = append(out, jsonFields(f.Type)...)
			continue
		}
		if name == "" {
			name = f.Name
		}
		// omitempty never drops a struct value such as time.Time; omitzero does.
		omitEmpty := strings.Contains(","+opts+",", ",omitempty,") && f.Type.Kind() != reflect.Struct
		omit := omitEmpty || strings.Contains(","+opts+",", ",omitzero,")
		out = append(out, jsonField{name: name, typ: f.Type, always: !omit})
	}
	return out
}

var timeType = reflect.TypeOf(time.Time{})
var marshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

func jsonKind(t reflect.Type) string {
	if t == timeType {
		return "string"
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	}
	return ""
}

func (s openAPISchemas) compare(path string, t reflect.Type, schema map[string]any, problems *[]string) {
	schema = s.resolve(schema)
	if schema == nil {
		*problems = append(*problems, path+": no schema")
		return
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	types := schemaTypes(schema)
	custom := t != timeType && (t.Implements(marshalerType) || reflect.PointerTo(t).Implements(marshalerType))
	if kind := jsonKind(t); kind != "" && len(types) > 0 && !custom {
		ok := false
		for _, ty := range types {
			ok = ok || ty == kind || (ty == "number" && kind == "integer")
		}
		if !ok {
			*problems = append(*problems, path+": Go writes "+kind+", schema says "+strings.Join(types, "|"))
		}
	}
	switch {
	case t == timeType || custom:
		return
	case t.Kind() == reflect.Slice:
		if items, ok := schema["items"].(map[string]any); ok {
			s.compare(path+"[]", t.Elem(), items, problems)
		}
	case t.Kind() == reflect.Map:
		if values, ok := schema["additionalProperties"].(map[string]any); ok && t.Elem().Kind() != reflect.Interface {
			s.compare(path+"{}", t.Elem(), values, problems)
		}
	case t.Kind() == reflect.Struct:
		props, _ := schema["properties"].(map[string]any)
		required := map[string]bool{}
		if list, ok := schema["required"].([]any); ok {
			for _, r := range list {
				required[r.(string)] = true
			}
		}
		seen := map[string]bool{}
		for _, f := range jsonFields(t) {
			seen[f.name] = true
			if webInputOnly[t.Name()+"."+f.name] {
				continue
			}
			prop, ok := props[f.name].(map[string]any)
			if !ok {
				*problems = append(*problems, path+"."+f.name+": written by Go, missing from schema")
				continue
			}
			if f.always && !required[f.name] {
				*problems = append(*problems, path+"."+f.name+": always written, schema does not require it")
			}
			if !f.always && required[f.name] {
				*problems = append(*problems, path+"."+f.name+": schema requires it, Go omits it when empty")
			}
			s.compare(path+"."+f.name, f.typ, prop, problems)
		}
		for name := range props {
			if !seen[name] {
				*problems = append(*problems, path+"."+name+": in schema, never written by Go")
			}
		}
	}
}

func TestWebOpenAPIMatchesGoResponses(t *testing.T) {
	schemas := loadOpenAPISchemas(t)
	problems := []string{}
	for name, value := range webContractSchemas {
		schema, ok := schemas[name]
		if !ok {
			problems = append(problems, name+": schema missing")
			continue
		}
		schemas.compare(name, reflect.TypeOf(value), schema, &problems)
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Errorf("docs/dev/web-ui.openapi.yaml disagrees with the Go response structs:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestWebJobPhaseOmitsUnsetTimes keeps a running phase from claiming it
// finished in year 1, which encoding/json writes for a zero time.Time.
func TestWebJobPhaseOmitsUnsetTimes(t *testing.T) {
	raw, err := json.Marshal(WebJobPhase{Name: "deploy", State: "running", StartedAt: time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(raw), `{"name":"deploy","state":"running","startedAt":"2026-10-25T08:00:00Z"}`; got != want {
		t.Fatalf("running phase = %s, want %s", got, want)
	}
}

// TestWebOpenAPIComparisonCatchesDrift proves the comparison is not vacuous:
// a renamed field, a field made optional, and a wrong kind are all reported.
func TestWebOpenAPIComparisonCatchesDrift(t *testing.T) {
	type sample struct {
		Kept    string    `json:"kept"`
		Renamed string    `json:"renamed"`
		Count   int       `json:"count,omitempty"`
		At      time.Time `json:"at,omitempty"`
	}
	schemas := openAPISchemas{}
	var schema map[string]any
	if err := yaml.Unmarshal([]byte(`
type: object
required: [kept, count]
properties:
  kept: {type: string}
  oldName: {type: string}
  count: {type: string}
  at: {type: string}
`), &schema); err != nil {
		t.Fatal(err)
	}
	problems := []string{}
	schemas.compare("Sample", reflect.TypeOf(sample{}), schema, &problems)
	sort.Strings(problems)
	want := []string{
		"Sample.at: always written, schema does not require it",
		"Sample.count: Go writes integer, schema says string",
		"Sample.count: schema requires it, Go omits it when empty",
		"Sample.oldName: in schema, never written by Go",
		"Sample.renamed: written by Go, missing from schema",
	}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("problems = %q, want %q", problems, want)
	}
}

// TestWebOpenAPIComparisonReadsAllOf: a required list added through allOf
// binds the fields its other parts declare.
func TestWebOpenAPIComparisonReadsAllOf(t *testing.T) {
	type sample struct {
		Refs []string `json:"refs,omitempty"`
	}
	var schema map[string]any
	if err := yaml.Unmarshal([]byte(`
allOf:
- type: object
  properties:
    refs: {type: array, items: {type: string}}
- required: [refs]
`), &schema); err != nil {
		t.Fatal(err)
	}
	problems := []string{}
	openAPISchemas{}.compare("Sample", reflect.TypeOf(sample{}), schema, &problems)
	if want := []string{"Sample.refs: schema requires it, Go omits it when empty"}; !reflect.DeepEqual(problems, want) {
		t.Fatalf("problems = %q, want %q", problems, want)
	}
}
