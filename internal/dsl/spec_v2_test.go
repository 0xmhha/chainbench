package dsl

import (
	"encoding/json"
	"errors"
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
		"existing genesis without ref": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","genesis":{"mode":"existing"}},"steps":[{"expect":"blockNumber","is":1}]}`,
		"existing genesis with overlay": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","genesis":{"mode":"existing","ref":"g.json","set":{"config.chainId":9}}},"steps":[{"expect":"blockNumber","is":1}]}`,
		"unknown launch scope": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","launch":{"bp1":{"mine":true}}},"steps":[{"expect":"blockNumber","is":1}]}`,
		"override hook": `{"schemaVersion":"2","kind":"case","id":"x","env":{"chain":"wbft"},
			"steps":[{"override":{"env.launch":{}}},{"expect":"blockNumber","is":1}]}`,
		// A typo in a do step's expect adjunct must be refused, not silently
		// treated as the default success (WA8) — "revrt" is not "revert".
		"typo expect adjunct": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","binaries":{"default":"gwbft"}},
			"steps":[{"do":"sendTx","from":"0xa","expect":"revrt"},{"expect":"blockNumber","is":1}]}`,
		// A timeouts value that is not a duration must be refused, not silently
		// ignored at run time (WA15).
		"bad timeout duration": `{"schemaVersion":"2","kind":"case","id":"x",
			"env":{"chain":"wbft","binaries":{"default":"gwbft"}},"timeouts":{"case":"tenminutes"},
			"steps":[{"expect":"blockNumber","is":1}]}`,
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

// TestGenesisExistingMode pins W4's finished-genesis form: mode "existing" with
// a ref lowers to Spec.Chain.GenesisExisting and does not build a template.
func TestGenesisExistingMode(t *testing.T) {
	raw := `{"schemaVersion":"2","kind":"case","id":"x",
		"env":{"chain":"wbft","binaries":{"default":"gwbft"},
		       "genesis":{"mode":"existing","ref":"genesis-regression.json"}},
		"steps":[{"expect":"blockNumber","is":1}]}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if s.Chain.GenesisExisting != "genesis-regression.json" {
		t.Fatalf("GenesisExisting = %q, want the ref", s.Chain.GenesisExisting)
	}
	if len(s.Chain.GenesisOverlay) != 0 {
		t.Fatalf("existing mode must not build an overlay: %v", s.Chain.GenesisOverlay)
	}
}

// TestInlineEnv_NonObjectBaseIsAnError pins S6's override form: a case extends a canonical env
// and names only what differs. The named fields replace the canonical env's
// whole; everything else is inherited.
// TestInlineEnv_NonObjectBaseIsAnError: a referenced env that is not an object
// must be reported, not crash the check that is supposed to report it. JSON null
// is the case that mattered — it unmarshals into a nil map without error, and
// the override loop then assigned into that nil map and panicked.
func TestInlineEnv_NonObjectBaseIsAnError(t *testing.T) {
	cases := []struct {
		name string
		base string
	}{
		{"null", `null`},
		{"array", `[1,2]`},
		{"string", `"env"`},
		{"number", `7`},
		{"malformed", `{`},
	}
	// The case must name at least one override field: that is what reaches the
	// merge loop.
	caseRef := `{"schemaVersion":"2","kind":"case","id":"c1",
		"env":{"extends":"base","topology":{"bp":3}},
		"steps":[{"expect":"blockNumber","is":1}]}`
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := InlineEnv([]byte(caseRef), func(string) ([]byte, error) {
				return []byte(tc.base), nil
			})
			if err == nil {
				t.Fatalf("a %s env must be refused", tc.name)
			}
		})
	}
}

// TestInlineEnv_NullBaseWithNoOverrideIsAlsoRefused: without an override field
// the merge loop never runs, so this path used to survive by accident. It is
// still not a usable env, and the failure should come from the same check.
func TestInlineEnv_NullBaseWithNoOverrideIsAlsoRefused(t *testing.T) {
	caseRef := `{"schemaVersion":"2","kind":"case","id":"c1",
		"env":{"extends":"base"},
		"steps":[{"expect":"blockNumber","is":1}]}`
	if _, err := InlineEnv([]byte(caseRef), func(string) ([]byte, error) {
		return []byte(`null`), nil
	}); err == nil {
		t.Fatal("a null env must be refused even with no override field")
	}
}

// TestInlineEnv_Extends pins S6's override form on the merge rule that replaced
// the shallow one: a case names what differs and inherits the rest, key by key.
func TestInlineEnv_Extends(t *testing.T) {
	canonical := `{"schemaVersion":"2","kind":"env","id":"stablenet-15","chain":"stablenet",
		"binaries":{"default":"gstable"},"topology":{"bp":"max","pn":1,"en":1}}`
	caseRef := `{"schemaVersion":"2","kind":"case","id":"c1",
		"env":{"extends":"stablenet-15","topology":{"bp":3,"en":1}},
		"steps":[{"expect":"blockNumber","is":1}]}`

	out, err := InlineEnv([]byte(caseRef), func(id string) ([]byte, error) {
		if id != "stablenet-15" {
			return nil, fmt.Errorf("unexpected id %s", id)
		}
		return []byte(canonical), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	// Inherited from the canonical env.
	if s.Chain.Name != "stablenet" {
		t.Fatalf("chain not inherited: %+v", s.Chain)
	}
	// Merged by key: the case's bp replaces the shared env's "max" form, and
	// what the case did not name stays. A case that wanted no proxy tier used
	// to get it by naming the whole field; it says so now.
	bp, ok := s.Topology["bp"]
	if !ok || bp != float64(3) {
		t.Fatalf("topology.bp = %v, want the overriding 3", s.Topology["bp"])
	}
	if pn, hasPN := s.Topology["pn"]; !hasPN || pn != float64(1) {
		t.Fatalf("topology.pn = %v, want the shared env's 1 to survive: %v", s.Topology["pn"], s.Topology)
	}

	// And the way to drop it.
	dropped := `{"schemaVersion":"2","kind":"case","id":"c1",
		"env":{"extends":"stablenet-15","topology":{"bp":3,"pn":null}},
		"steps":[{"expect":"blockNumber","is":1}]}`
	out, err = InlineEnv([]byte(dropped), func(string) ([]byte, error) { return []byte(canonical), nil })
	if err != nil {
		t.Fatal(err)
	}
	s, err = Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, hasPN := s.Topology["pn"]; hasPN {
		t.Fatalf("a null must remove the key: %v", s.Topology)
	}

	// extends with a non-string id is rejected.
	bad := `{"schemaVersion":"2","kind":"case","id":"c1","env":{"extends":5},"steps":[]}`
	if _, err := InlineEnv([]byte(bad), func(string) ([]byte, error) { return nil, nil }); err == nil {
		t.Fatal("a non-string extends id was accepted")
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

// TestV2_UpgradeEnvNamesOneBinaryPerSideOfTheFork: a hardfork declaration
// carries through to the executable spec, and one that leaves a side out is
// refused rather than composed as a single-binary network.
func TestV2_UpgradeEnvNamesOneBinaryPerSideOfTheFork(t *testing.T) {
	table := `,"topology":{"nodes":[{"index":1,"role":"en","binary":"to"},{"index":2,"role":"bp"}]}`
	good := `{"schemaVersion":"2","kind":"case","id":"h","env":{
	  "schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"from":"gwemix","to":"gwbft"}` + table + `,
	  "upgrade":{"profile":"p.yaml"}},
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	s, err := Parse([]byte(good))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.EnvUpgrade == nil || s.EnvUpgrade.Profile != "p.yaml" {
		t.Fatalf("upgrade not lowered: %+v", s.EnvUpgrade)
	}
	if s.Chain.Binaries[BinaryFrom] != "gwemix" || s.Chain.Binaries[BinaryTo] != "gwbft" || s.Chain.Binary != "" {
		t.Fatalf("binaries = %v / %q", s.Chain.Binaries, s.Chain.Binary)
	}

	bad := map[string]string{
		"missing the to side":  `"binaries":{"from":"gwemix"}` + table + `,"upgrade":{"profile":"p"}`,
		"default with upgrade": `"binaries":{"from":"gwemix","to":"gwbft","default":"x"}` + table + `,"upgrade":{"profile":"p"}`,
		"no node table":        `"binaries":{"from":"gwemix","to":"gwbft"},"upgrade":{"profile":"p"}`,
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

// TestParseV2_RequiresAndCapabilitiesUnion pins WA22: a case's own requires and
// the env's capabilities union rather than the env's being dropped whenever the
// case lists any of its own. The shared entry is not duplicated.
func TestParseV2_RequiresAndCapabilitiesUnion(t *testing.T) {
	raw := `{"schemaVersion":"2","kind":"case","id":"x",
	  "requires":["account-extra"],
	  "env":{"chain":"wbft","binaries":{"default":"gwbft"},"capabilities":["short-expiry","account-extra"]},
	  "steps":[{"expect":"blockNumber","is":1}]}`
	s, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"account-extra", "short-expiry"}
	if !reflect.DeepEqual(s.Requires, want) {
		t.Fatalf("requires = %v, want %v (case requires and env capabilities must union without duplicates)", s.Requires, want)
	}
}

// TestSchemaV2StatementOnEachIsArray guards WA17: the do/expect statement
// onEach selector is a list, matching how the parser and validator read it
// ([]any). It drifted to "string" once, silently, because the top-level type
// test cannot see per-statement args.
func TestSchemaV2StatementOnEachIsArray(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(SchemaV2, &doc); err != nil {
		t.Fatalf("schema: %v", err)
	}
	defs, _ := doc["$defs"].(map[string]any)
	for _, name := range []string{"doStatement", "expectStatement"} {
		def, _ := defs[name].(map[string]any)
		props, _ := def["properties"].(map[string]any)
		oe, ok := props["onEach"].(map[string]any)
		if !ok {
			t.Fatalf("%s has no onEach property", name)
		}
		if oe["type"] != "array" {
			t.Errorf("%s.onEach type = %v, want array (the parser reads []any)", name, oe["type"])
		}
	}
}

// TestV2_BinaryReferenceMustBeAName is the rule that keeps a definition
// portable.
//
// Five specs carried /data/chainbench/bin/... and ran on one docker environment
// and nowhere else, while the environment file passed on the same command line
// already produced that exact path from the binary's name. A definition says
// WHICH binary; a workspace-config says WHERE binaries live.
func TestV2_BinaryReferenceMustBeAName(t *testing.T) {
	caseWith := func(binaries string) []byte {
		return []byte(`{"schemaVersion":"2","kind":"case","id":"b","env":{
		  "schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
		  "binaries":` + binaries + `,"topology":{"bp":4}},
		  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`)
	}

	for _, ok := range []string{
		`{"default":"gstable"}`,
		`{"default":"${GSTABLE_BIN:-gstable}"}`, // a name with a machine-local escape hatch
		`{"default":"$GSTABLE_BIN"}`,            // only the machine knows; placeBinary judges it
		`{"default":"${GSTABLE_BIN}"}`,          // same
		`{"default":"gstable-2.1.0"}`,           // a version in the name is still a name
	} {
		if _, err := Parse(caseWith(ok)); err != nil {
			t.Errorf("binaries %s must parse: %v", ok, err)
		}
	}

	for _, bad := range []struct{ binaries, why string }{
		{`{"default":"/data/chainbench/bin/gstable"}`, "an absolute path"},
		{`{"default":"build/gstable"}`, "a relative path"},
		{`{"default":"~/bin/gstable"}`, "a home directory"},
		{`{"default":"${GSTABLE_BIN:-/opt/gstable}"}`, "a path wearing a variable"},
		{`{"default":""}`, "empty"},
	} {
		_, err := Parse(caseWith(bad.binaries))
		if err == nil {
			t.Errorf("binaries %s (%s) must be refused", bad.binaries, bad.why)
			continue
		}
		// The refusal has to say where the path belongs instead, or the author
		// has nowhere to put it.
		if bad.binaries != `{"default":""}` && !strings.Contains(err.Error(), "workspace-config") &&
			!strings.Contains(err.Error(), "machine") {
			t.Errorf("binaries %s: error %q says nothing about where a path belongs", bad.binaries, err)
		}
	}
}

// mergeCase builds a case whose env extends "base" with the given override.
func mergeCase(override string) []byte {
	return []byte(`{"schemaVersion":"2","kind":"case","id":"m","env":` + override + `,
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`)
}

// sharedEnv is the shape a case extends in these tests: two scopes of config
// and a capability list, which is where the old shallow rule lost things.
const sharedEnv = `{"schemaVersion":"2","kind":"env","id":"base","chain":"stablenet",
  "topology":{"bp":4,"en":1},
  "config":{"all":{"metricsHost":"0.0.0.0"},"node1":{"syncMode":"archive"}},
  "capabilities":["rpc","consensus"]}`

func lookupShared(id string) ([]byte, error) {
	if id != "base" {
		return nil, errors.New("no such env")
	}
	return []byte(sharedEnv), nil
}

// TestInlineEnv_DeepMergeKeepsWhatTheCaseDidNotName is the defect the rule
// change exists to remove.
//
// The override used to replace a whole field. A case that wanted one node to
// differ wrote config.node2, and the shared env's "all" scope and node1's went
// with it — the case ran, and only the result was different.
func TestInlineEnv_DeepMergeKeepsWhatTheCaseDidNotName(t *testing.T) {
	raw, err := InlineEnv(mergeCase(`{"extends":"base","config":{"node2":{"syncMode":"snap"}}}`), lookupShared)
	if err != nil {
		t.Fatalf("inline: %v", err)
	}
	var got struct {
		Env struct {
			Config   map[string]map[string]any `json:"config"`
			Topology map[string]any            `json:"topology"`
		} `json:"env"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"all", "node1", "node2"} {
		if _, ok := got.Env.Config[scope]; !ok {
			t.Errorf("config scope %q was dropped: %v", scope, got.Env.Config)
		}
	}
	if got.Env.Config["node2"]["syncMode"] != "snap" {
		t.Errorf("the case's own value did not land: %v", got.Env.Config["node2"])
	}
	// A field the case did not mention is untouched.
	if got.Env.Topology["bp"] != float64(4) {
		t.Errorf("topology = %v, want the shared env's", got.Env.Topology)
	}
}

// TestInlineEnv_NullRemoves: deep merge alone can only add and overwrite, so
// without this there is no way to switch off what the shared env turned on. The
// commonest difference among the inline envs is exactly an absent capability
// list.
func TestInlineEnv_NullRemoves(t *testing.T) {
	raw, err := InlineEnv(mergeCase(`{"extends":"base","capabilities":null,"config":{"node1":null}}`), lookupShared)
	if err != nil {
		t.Fatalf("inline: %v", err)
	}
	var got struct {
		Env map[string]any `json:"env"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if _, present := got.Env["capabilities"]; present {
		t.Errorf("capabilities survived a null: %v", got.Env)
	}
	cfg, _ := got.Env["config"].(map[string]any)
	if _, present := cfg["node1"]; present {
		t.Errorf("config.node1 survived a null: %v", cfg)
	}
	if _, present := cfg["all"]; !present {
		t.Errorf("removing one scope took another with it: %v", cfg)
	}
}

// TestInlineEnv_ArraysReplace: unioning reads as generous and takes away the
// only way to drop an entry, which is the same hole as having no delete.
func TestInlineEnv_ArraysReplace(t *testing.T) {
	raw, err := InlineEnv(mergeCase(`{"extends":"base","capabilities":["rpc"]}`), lookupShared)
	if err != nil {
		t.Fatalf("inline: %v", err)
	}
	var got struct {
		Env struct {
			Capabilities []string `json:"capabilities"`
		} `json:"env"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Env.Capabilities) != 1 || got.Env.Capabilities[0] != "rpc" {
		t.Errorf("capabilities = %v, want the case's list alone", got.Env.Capabilities)
	}
}

// TestInlineEnv_RefusesAChainOfExtends: a shared env is the base, not a step in
// a chain. Following one would make "what does this case actually declare"
// answerable only by reading a sequence of files, which is the thing the shared
// env exists to avoid.
func TestInlineEnv_RefusesAChainOfExtends(t *testing.T) {
	chained := func(string) ([]byte, error) {
		return []byte(`{"schemaVersion":"2","kind":"env","id":"base","chain":"stablenet","extends":"other"}`), nil
	}
	if _, err := InlineEnv(mergeCase(`{"extends":"base"}`), chained); err == nil {
		t.Error("an env that extends another must be refused")
	}
}

// TestInlineEnv_MergedResultIsStillParsedStrictly: the merge does not check for
// keys neither side should have, because the parse that follows does. A typo
// that survives the merge has to fail there.
func TestInlineEnv_MergedResultIsStillParsedStrictly(t *testing.T) {
	raw, err := InlineEnv(mergeCase(`{"extends":"base","toplogy":{"bp":9}}`), lookupShared)
	if err != nil {
		t.Fatalf("inline: %v", err)
	}
	if _, err := Parse(raw); err == nil {
		t.Error("a misspelled field must be refused by the parse")
	}
}

// TestV2_UpgradeSaysWhichFileCarriesTheFork.
//
// A hardfork reaches the post-fork build either through a second genesis
// document or through that build's own config file. Both are real: a chain team
// ships one or the other, and a case that pins which one is testing what they
// will actually do. Empty means the genesis, which is what every declaration
// written before this meant.
func TestV2_UpgradeSaysWhichFileCarriesTheFork(t *testing.T) {
	spec := func(carry string) string {
		return `{"schemaVersion":"2","kind":"case","id":"h","env":{
		  "schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
		  "binaries":{"from":"gwemix","to":"gwbft"},"topology":{"nodes":[{"index":1,"role":"en","binary":"to"},{"index":2,"role":"bp"}]},
		  "upgrade":{"profile":"p.yaml"` + carry + `}},
		  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	}
	for _, want := range []string{"", CarryGenesis, CarryConfig} {
		field := ""
		if want != "" {
			field = `,"carry":"` + want + `"`
		}
		s, err := Parse([]byte(spec(field)))
		if err != nil {
			t.Fatalf("carry %q: %v", want, err)
		}
		if s.EnvUpgrade.Carry != want {
			t.Errorf("carry = %q, want %q", s.EnvUpgrade.Carry, want)
		}
	}
	// Refused by name rather than accepted and quietly carried the default way,
	// which would run a different hardfork from the one the case asked for.
	if _, err := Parse([]byte(spec(`,"carry":"sidecar"`))); err == nil {
		t.Error("an unknown carry was accepted")
	}
}

// TestV2_ACaseThatCrossesTheForkItselfMustHaveOneToCross.
//
// Naming the step is how a case says the moment is its own: it acts before the
// fork and crosses when it is ready. On a network that crosses no fork the step
// has nothing to do, and finding that out at run time means finding it out
// after the network is up and the case has already acted.
func TestV2_ACaseThatCrossesTheForkItselfMustHaveOneToCross(t *testing.T) {
	spec := func(env, steps string) string {
		return `{"schemaVersion":"2","kind":"case","id":"h","env":{
		  "schemaVersion":"2","kind":"env","id":"e","chain":"wbft"` + env + `},
		  "steps":[` + steps + `]}`
	}
	fork := `,"binaries":{"from":"gwemix","to":"gwbft"},"topology":{"nodes":[{"index":1,"role":"en","binary":"to"},{"index":2,"role":"bp"}]},
	  "upgrade":{"profile":"p.yaml"}`
	cross := `{"do":"crossFork","timeout":"120s"}`
	check := `{"expect":"blockNumber","compare":"Greater","is":"0"}`

	if _, err := Parse([]byte(spec(fork, cross+","+check))); err != nil {
		t.Fatalf("a case crossing its own declared fork was refused: %v", err)
	}
	if _, err := Parse([]byte(spec("", cross+","+check))); err == nil {
		t.Error("a case crossed a fork its env never declared")
	}
	// Twice reads as though the network crossed twice. The step is idempotent,
	// so the second would report "already crossed" and the case would pass
	// while saying something that did not happen.
	if _, err := Parse([]byte(spec(fork, cross+","+cross+","+check))); err == nil {
		t.Error("a case crossed the same fork twice")
	}
	// And a case that says nothing is untouched: the composition crosses.
	if _, err := Parse([]byte(spec(fork, check))); err != nil {
		t.Fatalf("a case that leaves the fork to the composition was refused: %v", err)
	}
}

// TestV2_AHardforkNeedsANodeTableAndNoTemplate.
//
// A hardfork says which build each node runs, and the node table is the only
// place that says it. It used to be optional: an upgrade env without one went
// to a composer of its own, which sized the network from its preset and
// generated the pre-fork genesis by running the producer's binary against a
// template the case named. That composer is gone, so both of those are now
// errors rather than a path.
func TestV2_AHardforkNeedsANodeTableAndNoTemplate(t *testing.T) {
	spec := func(env string) string {
		return `{"schemaVersion":"2","kind":"case","id":"h","env":{
		  "schemaVersion":"2","kind":"env","id":"e","chain":"wemix",
		  "binaries":{"default":"gwemix","to":{"binary":"gwbft","chain":"wbft"}}` + env + `},
		  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	}
	table := `,"topology":{"nodes":[{"index":1,"role":"en","binary":"to"},{"index":2,"role":"bp"}]}`
	upgrade := `,"upgrade":{"preset":"wemix-upgrade","from":"default"}`

	if _, err := Parse([]byte(spec(table + upgrade))); err != nil {
		t.Fatalf("a hardfork with a node table was refused: %v", err)
	}
	if _, err := Parse([]byte(spec(upgrade))); err == nil {
		t.Error("a hardfork with no node table was accepted")
	}
	withTemplate := `,"upgrade":{"preset":"wemix-upgrade","from":"default","template":"t.json"}`
	if _, err := Parse([]byte(spec(table + withTemplate))); err == nil {
		t.Error("a hardfork naming a genesis template was accepted")
	}
}
