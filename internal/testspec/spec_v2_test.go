package testspec

import (
	"encoding/json"
	"fmt"
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
	if len(s.EnvLaunch) != 2 {
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
		"role-scoped launch": `{"schemaVersion":"2","kind":"case","id":"x",
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
	os, cs := sequenceOf(orig), sequenceOf(conv)
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
