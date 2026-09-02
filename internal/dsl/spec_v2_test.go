package dsl

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const v2Case = `{
  "schemaVersion": "2",
  "kind": "case",
  "id": "V2-001",
  "env": {
    "schemaVersion": "2",
    "kind": "env",
    "id": "wbft-4",
    "target": "local:.chainbench/work",
    "chain": "wbft",
    "binaries": { "default": "gwbft" },
    "keys": { "nodekeys": { "source": "generate", "ref": "keys/gen1", "bootnode": "/bin/bootnode" } },
    "genesis": { "set": { "config.chainId": 8284 }, "overlay": { "config": { "bohoBlock": 100 } } },
    "topology": { "validators": 4 },
    "launch": { "all": { "networkid": 8284, "nodiscover": true } },
    "capabilities": ["rpc", "ws"]
  },
  "on": "node1",
  "timeouts": { "case": "10m" },
  "hooks": {
    "pre":  [ { "do": "waitBlock", "n": 1 } ],
    "post": [ { "do": "waitBlock", "n": 1 } ],
    "onFail": [ { "do": "waitBlock", "n": 1 } ]
  },
  "steps": [
    { "do": "sendTx", "on": "node1", "save": "h1", "expect": "receipt" },
    { "expect": "txStatus", "hash": "$h1", "is": "0x1" },
    { "do": "waitBlock", "n": 2 },
    { "expect": "metric", "name": "chain_head_block", "compare": "GreaterOrEqual", "is": 2 },
    { "expect": "rpc", "method": "eth_blockNumber", "is": "0x2" }
  ]
}`

func TestParseV2CaseLowering(t *testing.T) {
	s, err := Parse([]byte(v2Case))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "V2-001" || s.Chain.Name != "wbft" || s.Chain.Binary != "gwbft" {
		t.Fatalf("header lowered wrong: %+v", s)
	}
	if s.Placement != "local:.chainbench/work" || s.DefaultOn != "node1" {
		t.Fatalf("target/on lowered wrong: %q %q", s.Placement, s.DefaultOn)
	}
	if len(s.Requires) != 2 {
		t.Fatalf("capabilities -> requires: %v", s.Requires)
	}

	// genesis.set dot-path merges into the overlay.
	cfg, _ := s.Chain.GenesisOverlay["config"].(map[string]any)
	if cfg == nil || cfg["chainId"] != float64(8284) || cfg["bohoBlock"] != float64(100) {
		t.Fatalf("genesis overlay = %#v", s.Chain.GenesisOverlay)
	}

	// keys/launch declarations carry for the surface to fold.
	if s.EnvKeys == nil || s.EnvKeys.Source != "generate" || s.EnvKeys.Ref != "keys/gen1" {
		t.Fatalf("EnvKeys = %+v", s.EnvKeys)
	}
	if len(s.EnvLaunch["all"]) != 2 {
		t.Fatalf("EnvLaunch = %+v", s.EnvLaunch)
	}

	// Sequence: 5 statements, interleaved, with the "is" -> "expected" rename
	// and the rpc alias applied.
	if len(s.Sequence) != 5 {
		t.Fatalf("sequence = %d statements", len(s.Sequence))
	}
	if s.Sequence[0].Do != "sendTx" || s.Sequence[0].Args["expect"] != "receipt" {
		t.Fatalf("stmt0 = %+v", s.Sequence[0])
	}
	if s.Sequence[1].Expect != "txStatus" || s.Sequence[1].Args["expected"] != "0x1" {
		t.Fatalf("stmt1 = %+v", s.Sequence[1])
	}
	if s.Sequence[3].Expect != "metric" {
		t.Fatalf("stmt3 = %+v", s.Sequence[3])
	}
	if s.Sequence[4].Expect != "rpcCall" {
		t.Fatalf("rpc alias not applied: %+v", s.Sequence[4])
	}

	// Hooks lowered to v1 action maps; onFail is runtime-only.
	if len(s.PreActions) != 1 || len(s.PostActions) != 1 || len(s.OnFailActions) != 1 {
		t.Fatalf("hooks = pre %d post %d onFail %d", len(s.PreActions), len(s.PostActions), len(s.OnFailActions))
	}
}

func TestParseV2Strictness(t *testing.T) {
	cases := map[string]string{
		"unknown case field": `{"schemaVersion":"2","kind":"case","id":"x","typo":1,
			"env":{"chain":"wbft"},"steps":[{"expect":"blockNumber","is":1}]}`,
		"unknown env field": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","nope":1},"steps":[{"expect":"blockNumber","is":1}]}`,
		"env alone": `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft"}`,
		"no kind":   `{"schemaVersion":"2","id":"x"}`,
		"no expects": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft"},"steps":[{"do":"waitBlock","n":1}]}`,
		"unsupported genesis mode": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","genesis":{"mode":"inherit"}},"steps":[{"expect":"blockNumber","is":1}]}`,
		"unknown launch scope": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","launch":{"bp1":{"mine":true}}},"steps":[{"expect":"blockNumber","is":1}]}`,
		"override hook": `{"schemaVersion":"2","kind":"case","id":"x","env":{"chain":"wbft"},
			"steps":[{"override":{"env.launch":{}}},{"expect":"blockNumber","is":1}]}`,
	}
	for name, raw := range cases {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("%s: must fail", name)
		}
	}
}

func TestParseV2ScopedLaunch(t *testing.T) {
	raw := `{"schemaVersion":"2","kind":"case","id":"x","env":{"chain":"wbft",
	  "binaries":{"default":"gwbft"},
	  "launch":{"all":{"metrics":true},"bp":{"mine":true},"node1":{"verbosity":5}}},
	  "steps":[{"expect":"blockNumber","is":1}]}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := s.EnvLaunch["all"]; len(got) != 1 || got[0] != "metrics=true" {
		t.Errorf("all scope = %v", got)
	}
	if got := s.EnvLaunch["bp"]; len(got) != 1 || got[0] != "mine=true" {
		t.Errorf("bp scope = %v", got)
	}
	if got := s.EnvLaunch["node1"]; len(got) != 1 || got[0] != "verbosity=5" {
		t.Errorf("node1 scope = %v", got)
	}
}

func TestInlineEnv(t *testing.T) {
	caseRef := `{"schemaVersion":"2","kind":"case","id":"c1","env":"wbft-4",
		"steps":[{"expect":"blockNumber","is":1}]}`
	envDoc := `{"schemaVersion":"2","kind":"env","id":"wbft-4","chain":"wbft"}`

	out, err := InlineEnv([]byte(caseRef), func(id string) ([]byte, error) {
		if id != "wbft-4" {
			return nil, fmt.Errorf("unexpected id %s", id)
		}
		return []byte(envDoc), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if s.Chain.Name != "wbft" {
		t.Fatalf("inlined env not used: %+v", s.Chain)
	}

	// Unresolvable ref names the id.
	if _, err := InlineEnv([]byte(caseRef), nil); err == nil || !strings.Contains(err.Error(), "wbft-4") {
		t.Fatalf("nil resolver: %v", err)
	}
	// A v1 spec passes through untouched.
	v1 := []byte(`{"schemaVersion":"1","id":"x"}`)
	if out, err := InlineEnv(v1, nil); err != nil || string(out) != string(v1) {
		t.Fatalf("v1 passthrough: %v", err)
	}
	// A case parsed without inlining reports the pending reference.
	if _, err := Parse([]byte(caseRef)); err == nil || !strings.Contains(err.Error(), "InlineEnv") {
		t.Fatalf("unresolved ref parse: %v", err)
	}
}

// TestMigrateV1RoundTrip pins the §3.6 property: migrating a v1 spec and
// parsing the result yields the same executable content as parsing the v1
// spec directly.
func TestMigrateV1RoundTrip(t *testing.T) {
	v1 := []byte(`{
	  "schemaVersion": "1", "id": "RT-1",
	  "chain": {"name": "wbft", "binary": "gwbft", "genesisOverlay": {"config": {"x": 1}}},
	  "topology": {"validators": 2},
	  "defaultOn": "node1",
	  "preActions": [{"waitBlock": {"n": 1}}],
	  "steps": [{"sendTx": {"on": "node1", "save": "h"}}],
	  "assertions": [{"assert": "txStatus", "hash": "$h", "expected": "0x1", "compare": "Equal"}],
	  "postActions": [{"waitBlock": {"n": 1}}]
	}`)

	orig, err := Parse(v1)
	if err != nil {
		t.Fatal(err)
	}
	migrated, err := MigrateV1(v1)
	if err != nil {
		t.Fatal(err)
	}
	conv, err := Parse(migrated)
	if err != nil {
		t.Fatalf("migrated spec does not parse: %v\n%s", err, migrated)
	}

	if conv.ID != orig.ID || conv.Chain.Name != orig.Chain.Name || conv.Chain.Binary != orig.Chain.Binary {
		t.Fatalf("header drift: %+v vs %+v", conv.Chain, orig.Chain)
	}
	// The executable sequences must be identical.
	os, cs := SequenceOf(orig), SequenceOf(conv)
	ob, _ := json.Marshal(os)
	cb, _ := json.Marshal(cs)
	if string(ob) != string(cb) {
		t.Fatalf("sequence drift:\n v1: %s\n v2: %s", ob, cb)
	}
	if len(conv.PreActions) != 1 || len(conv.PostActions) != 1 {
		t.Fatalf("hooks drift: %+v %+v", conv.PreActions, conv.PostActions)
	}
}

func TestSchemaV2Embedded(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(SchemaV2, &doc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if doc["$id"] != "chainbench/testspec/v2" {
		t.Fatalf("schema $id = %v", doc["$id"])
	}
}

// TestSchemaV2MatchesParsedFields keeps the schema's field set from drifting
// from the strict parser's. The schema is documentation, not enforcement, so a
// field added to EnvV2/CaseV2 without a schema entry (or the reverse) goes
// unnoticed until a reader trusts the wrong one — which is how "config" came to
// say string while the parser read an object. It checks names, the drift that
// actually happens; types stay a manual review.
func TestSchemaV2MatchesParsedFields(t *testing.T) {
	cases := []struct {
		def  string
		typ  reflect.Type
		skip map[string]bool // struct fields the schema folds elsewhere
	}{
		{"envSpec", reflect.TypeOf(EnvV2{}), nil},
		// CaseV2.Env is json.RawMessage in Go (resolved after a first pass); the
		// schema spells out its string|envSpec shape.
		{"caseSpec", reflect.TypeOf(CaseV2{}), nil},
	}
	var doc struct {
		Defs map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(SchemaV2, &doc); err != nil {
		t.Fatalf("schema: %v", err)
	}
	for _, c := range cases {
		schemaFields := map[string]bool{}
		for k := range doc.Defs[c.def].Properties {
			schemaFields[k] = true
		}
		structFields := map[string]bool{}
		for i := 0; i < c.typ.NumField(); i++ {
			tag := c.typ.Field(i).Tag.Get("json")
			name, _, _ := strings.Cut(tag, ",")
			if name == "" || name == "-" {
				continue
			}
			structFields[name] = true
		}
		for f := range structFields {
			if !schemaFields[f] && !c.skip[f] {
				t.Errorf("%s: parser field %q has no schema property — add it to v2.schema.json", c.def, f)
			}
		}
		for f := range schemaFields {
			if !structFields[f] {
				t.Errorf("%s: schema property %q has no parser field — remove it from v2.schema.json", c.def, f)
			}
		}
	}
}

// TestSchemaV2MatchesParsedTypes catches the drift the field-set test cannot:
// a property whose declared JSON type disagrees with the parser field's Go type.
// This is what happened to "config" — schema "string" while the parser reads an
// object. Each env/case field's Go kind is mapped to the JSON type it decodes
// from and checked against the schema's declared type (resolving const, enum,
// oneOf, and $ref). Fields the parser reads as raw/any JSON are skipped, since
// the schema legitimately spells those as a union.
func TestSchemaV2MatchesParsedTypes(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(SchemaV2, &doc); err != nil {
		t.Fatalf("schema: %v", err)
	}
	defs, _ := doc["$defs"].(map[string]any)
	for _, c := range []struct {
		def string
		typ reflect.Type
	}{
		{"envSpec", reflect.TypeOf(EnvV2{})},
		{"caseSpec", reflect.TypeOf(CaseV2{})},
	} {
		def, _ := defs[c.def].(map[string]any)
		props, _ := def["properties"].(map[string]any)
		for i := 0; i < c.typ.NumField(); i++ {
			f := c.typ.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				continue
			}
			want := jsonKind(f.Type)
			if want == "" {
				continue // raw/any JSON — the schema may spell it as a union
			}
			prop, ok := props[name].(map[string]any)
			if !ok {
				continue // the field-set test reports a missing property
			}
			got := schemaTypesOf(prop, defs)
			if len(got) > 0 && !got[want] {
				t.Errorf("%s.%s: the parser reads a %s but the schema declares %v — v2.schema.json drifted from the parser type",
					c.def, name, want, sortedKeys(got))
			}
		}
	}
}

// jsonKind is the JSON type a Go type decodes from, or "" for raw/any JSON.
func jsonKind(t reflect.Type) string {
	if t == reflect.TypeOf(json.RawMessage(nil)) {
		return ""
	}
	switch t.Kind() {
	case reflect.Pointer:
		return jsonKind(t.Elem())
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return ""
	}
}

// schemaTypesOf resolves the JSON type(s) a schema property declares, following
// const/enum/oneOf and one level of $ref into $defs.
func schemaTypesOf(prop, defs map[string]any) map[string]bool {
	out := map[string]bool{}
	switch {
	case prop["$ref"] != nil:
		if ref, ok := prop["$ref"].(string); ok {
			const p = "#/$defs/"
			if d, ok := defs[strings.TrimPrefix(ref, p)].(map[string]any); ok && strings.HasPrefix(ref, p) {
				for k := range schemaTypesOf(d, defs) {
					out[k] = true
				}
			}
		}
	case prop["type"] != nil:
		if s, ok := prop["type"].(string); ok {
			out[s] = true
		}
	case prop["const"] != nil:
		out[valueKind(prop["const"])] = true
	case prop["enum"] != nil:
		if e, ok := prop["enum"].([]any); ok && len(e) > 0 {
			out[valueKind(e[0])] = true
		}
	case prop["oneOf"] != nil:
		if alts, ok := prop["oneOf"].([]any); ok {
			for _, alt := range alts {
				if m, ok := alt.(map[string]any); ok {
					for k := range schemaTypesOf(m, defs) {
						out[k] = true
					}
				}
			}
		}
	}
	delete(out, "")
	return out
}

// valueKind is the JSON type of a decoded literal (a const or enum value).
func valueKind(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	default:
		return ""
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestV2_UpgradeEnvNamesItsBinariesByRole: a handoff declaration carries
// through to the executable spec, and one that leaves a role out is refused
// rather than composed as a single-binary network.
func TestV2_UpgradeEnvNamesItsBinariesByRole(t *testing.T) {
	good := `{"schemaVersion":"2","kind":"case","id":"h","env":{
	  "schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"producer":"gwemix","validator":"gwbft"},
	  "upgrade":{"profile":"p.yaml","template":"t.json"}},
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	s, err := Parse([]byte(good))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.EnvUpgrade == nil || s.EnvUpgrade.Profile != "p.yaml" || s.EnvUpgrade.Template != "t.json" {
		t.Fatalf("upgrade not lowered: %+v", s.EnvUpgrade)
	}
	if s.Chain.Binaries[RoleProducer] != "gwemix" || s.Chain.Binaries[RoleValidator] != "gwbft" || s.Chain.Binary != "" {
		t.Fatalf("binaries = %v / %q", s.Chain.Binaries, s.Chain.Binary)
	}

	bad := map[string]string{
		"missing validator":    `"binaries":{"producer":"gwemix"},"upgrade":{"profile":"p","template":"t"}`,
		"default with upgrade": `"binaries":{"producer":"gwemix","validator":"gwbft","default":"x"},"upgrade":{"profile":"p","template":"t"}`,
		"no template":          `"binaries":{"producer":"gwemix","validator":"gwbft"},"upgrade":{"profile":"p"}`,
	}
	for name, env := range bad {
		raw := `{"schemaVersion":"2","kind":"case","id":"h","env":{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",` + env + `},
		  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestParseEnv_StandsOnItsOwn(t *testing.T) {
	env, err := ParseEnv([]byte(`{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"}}`))
	if err != nil || env.ID != "e" || env.Chain != "stablenet" {
		t.Fatalf("ParseEnv: %+v (%v)", env, err)
	}
	if !IsEnv([]byte(`{"schemaVersion":"2","kind":"env"}`)) || IsEnv([]byte(`{"schemaVersion":"2","kind":"case"}`)) {
		t.Fatal("IsEnv misreads the kind")
	}
	for name, raw := range map[string]string{
		"typo field": `{"schemaVersion":"2","kind":"env","id":"e","chain":"x","binaris":{}}`,
		"no chain":   `{"schemaVersion":"2","kind":"env","id":"e"}`,
		"wrong kind": `{"schemaVersion":"2","kind":"case","id":"e","chain":"x"}`,
	} {
		if _, err := ParseEnv([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestV2_ConfigScopesLowerToEnvConfig(t *testing.T) {
	raw := `{"schemaVersion":"2","kind":"case","id":"c","env":{
	  "schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"},
	  "config":{"all":{"metricsHost":"0.0.0.0"},"node2":{"syncMode":"snap"}}},
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := s.EnvConfig["all"]; len(got) != 1 || got[0] != "metricsHost=0.0.0.0" {
		t.Errorf("config.all lowered to %v", got)
	}
	if got := s.EnvConfig["node2"]; len(got) != 1 || got[0] != "syncMode=snap" {
		t.Errorf("config.node2 lowered to %v", got)
	}

	// A scope that is neither "all" nor node<N> is refused.
	bad := `{"schemaVersion":"2","kind":"case","id":"c","env":{
	  "schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"},
	  "config":{"bp1":{"syncMode":"snap"}}},
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	if _, err := Parse([]byte(bad)); err == nil || !strings.Contains(err.Error(), "config scope") {
		t.Fatalf("a non-all, non-node scope must be refused: %v", err)
	}
}
