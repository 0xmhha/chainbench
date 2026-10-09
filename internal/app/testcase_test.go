package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/dsl"
)

func editorCase(steps string) []byte {
	return []byte(`{"schemaVersion":"2","kind":"case","id":"editor","chainPreset":{"chain":"stablenet","binaries":{"default":"gstable"},"topology":{"bp":4}},"steps":` + steps + `}`)
}
func TestTestCaseRoundTripAndReferences(t *testing.T) {
	raw := editorCase(`[{"do":"read","source":"blockNumber","on":"node1","save":"head"},{"expect":"blockNumber","onEach":["node1","node2"],"compare":"GreaterOrEqual","is":"$head"}]`)
	first, err := PrepareTestCase(TestCaseInput{Content: raw})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PrepareTestCase(TestCaseInput{Content: first.Content})
	if err != nil {
		t.Fatal(err)
	}
	if first.SemanticFingerprint != second.SemanticFingerprint {
		t.Fatal("executable meaning changed")
	}
	if string(first.Content) != string(raw) || string(first.Original) != string(raw) {
		t.Fatal("import changed")
	}
	for _, steps := range []string{
		`[{"expect":"blockNumber","is":"$missing"}]`,
		`[{"expect":"unknownExtension","is":1}]`,
		`[{"expect":"blockNumber","is":1,"extra":true}]`,
		`[{"do":"waitBlock","target":{},"timeout":"bad"},{"expect":"blockNumber","is":1}]`,
		`[{"do":"read","source":"unknownReader"},{"expect":"blockNumber","is":1}]`,
		`[{"do":"read","source":"balanceAt"},{"expect":"blockNumber","is":1}]`,
		`[{"expect":"blockNumber","on":"node0","is":1}]`,
	} {
		if _, err := PrepareTestCase(TestCaseInput{Content: editorCase(steps)}); err == nil {
			t.Fatalf("accepted invalid %s", steps)
		}
	}
}
func TestTestCaseV1Migration(t *testing.T) {
	raw := []byte(`{"schemaVersion":"1","id":"legacy","chain":{"name":"stablenet","binary":"gstable"},"steps":[{"read":{"source":"blockNumber","save":"head"}}],"assertions":[{"assert":"blockNumber","compare":"GreaterOrEqual","expected":"$head"}]}`)
	migrated, err := PrepareTestCase(TestCaseInput{Content: raw})
	if err != nil {
		t.Fatal(err)
	}
	if !migrated.Migrated || string(migrated.Original) != string(raw) {
		t.Fatal("original not preserved")
	}
	next, err := PrepareTestCase(TestCaseInput{Content: migrated.Content})
	if err != nil {
		t.Fatal(err)
	}
	if migrated.SemanticFingerprint != next.SemanticFingerprint {
		t.Fatal("migration changed execution")
	}
	unknown := []byte(strings.Replace(string(raw), `"id":"legacy"`, `"id":"legacy","extension":true`, 1))
	if _, err = PrepareTestCase(TestCaseInput{Content: unknown}); err == nil {
		t.Fatal("dropped v1 extension")
	}
}
func TestTestCasePresetReference(t *testing.T) {
	preset := json.RawMessage(`{"schemaVersion":"2","kind":"chain-preset","id":"shared","chain":"stablenet","binaries":{"default":"gstable"},"topology":{"bp":4}}`)
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"reference","chainPreset":{"extends":"shared","description":"override"},"steps":[{"expect":"blockNumber","is":1}]}`)
	in := TestCaseInput{Content: raw, Presets: map[string]json.RawMessage{"shared": preset}}
	out, err := PrepareTestCase(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Content) != string(raw) {
		t.Fatal("reference flattened on export")
	}
	if _, err := PrepareTestCase(TestCaseInput{Content: raw}); err == nil {
		t.Fatal("unresolved reference accepted")
	}
}
func TestTestCaseExamples(t *testing.T) {
	root := filepath.Join("..", "..")
	presets := map[string]json.RawMessage{}
	paths, _ := filepath.Glob(filepath.Join(root, "presets", "chain", "*.json"))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var head struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &head) == nil {
			presets[head.ID] = raw
		}
	}
	paths, _ = filepath.Glob(filepath.Join(root, "examples", "specs", "*.json"))
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			_, err = PrepareTestCase(TestCaseInput{Content: raw, Presets: presets})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestV1MigrationPreservesSkipsAndGenesisMetadata(t *testing.T) {
	raw := []byte(`{"schemaVersion":"1","id":"metadata","skipsOn":[],"chain":{"name":"stablenet","binary":"gstable","genesisProvides":["rpc"],"genesisHaltsAt":10,"genesisOverlay":{"config":{"chainId":8283}}},"assertions":[{"assert":"blockNumber","expected":1}]}`)
	out, err := PrepareTestCase(TestCaseInput{Content: raw})
	if err != nil {
		t.Fatal(err)
	}
	var migrated map[string]any
	if err = json.Unmarshal(out.Content, &migrated); err != nil {
		t.Fatal(err)
	}
	if skips, exists := migrated["skipsOn"]; !exists || len(skips.([]any)) != 0 {
		t.Fatal("explicit no-skip expectation lost")
	}
	env := migrated["chainPreset"].(map[string]any)
	genesis := env["genesis"].(map[string]any)
	if genesis["haltsAt"] != float64(10) || len(genesis["provides"].([]any)) != 1 {
		t.Fatal("genesis metadata lost")
	}
}

// Every case the CLI runs must import into the editor unchanged in meaning.
func TestTestCaseCorpusImports(t *testing.T) {
	root := filepath.Join("..", "..")
	presets := map[string]json.RawMessage{}
	var cases []string
	err := filepath.WalkDir(filepath.Join(root, "tests", "tc"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".json" {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if dsl.IsEnv(raw) {
			var head struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(raw, &head) == nil {
				presets[head.ID] = raw
			}
			return nil
		}
		cases = append(cases, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := filepath.Glob(filepath.Join(root, "presets", "chain", "*.json"))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var head struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &head) == nil {
			presets[head.ID] = raw
		}
	}
	if len(cases) < 100 {
		t.Fatalf("corpus has %d cases; the walk is wrong", len(cases))
	}
	for _, p := range cases {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareTestCase(TestCaseInput{Content: raw, Presets: presets}); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

// An argument the builtin never reads is refused by name rather than kept
// as a setting that does nothing.
func TestEditorRefusesIgnoredArguments(t *testing.T) {
	for steps, field := range map[string]string{
		`[{"expect":"chainId","is":"1","timeout":"5s"}]`:                                      "timeout",
		`[{"do":"restartNode","on":"node1","expectFail":true},{"expect":"chainId","is":"1"}]`: "expectFail",
		`[{"expect":"blockStalled","onEach":["node1","node2"]}]`:                              "onEach",
	} {
		_, err := PrepareTestCase(TestCaseInput{Content: editorCase(steps)})
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("%s: err = %v, want %s refused", steps, err, field)
		}
		// The reason names the field, not every other builtin's requirements.
		if err != nil && strings.Contains(err.Error(), "variant") {
			t.Errorf("%s: refusal lists unrelated builtins: %v", steps, err)
		}
	}
}

func TestEditorSchemaRejectsInvalidParameterTypesAndDurations(t *testing.T) {
	for _, steps := range []string{
		`[{"do":"waitBlock","target":-1},{"expect":"blockNumber","is":1}]`,
		`[{"do":"waitBlock","target":"tomorrow"},{"expect":"blockNumber","is":1}]`,
		`[{"expect":"blockNumber","is":1,"timeout":"0s"}]`,
		`[{"expect":"blockNumber","is":1,"compare":"Approximately"}]`,
		`[{"expect":"derive","op":"unknown","of":[1,2],"is":3}]`,
	} {
		if _, err := PrepareTestCase(TestCaseInput{Content: editorCase(steps)}); err == nil {
			t.Errorf("invalid parameter accepted: %s", steps)
		}
	}
}

func TestV1UnrepresentableMigrationRetainsEditableLegacyDeclaration(t *testing.T) {
	raw := []byte(`{"schemaVersion":"1","id":"legacy-config","chain":{"name":"stablenet","binary":"gstable","config":"site.toml"},"assertions":[{"assert":"blockNumber","expected":1}]}`)
	out, err := PrepareTestCase(TestCaseInput{Content: raw})
	if err != nil {
		t.Fatal(err)
	}
	if out.Migrated || out.MigrationMessage == "" || string(out.Content) != string(raw) {
		t.Fatal("unrepresentable migration changed declaration")
	}
	next, err := PrepareTestCase(TestCaseInput{Content: out.Content})
	if err != nil {
		t.Fatal(err)
	}
	if out.SemanticFingerprint != next.SemanticFingerprint {
		t.Fatal("legacy round-trip changed meaning")
	}
}

func TestEditorEnforcesBuiltinRequiredAlternatives(t *testing.T) {
	for _, statement := range []string{
		`{"do":"deployContract"}`, `{"do":"load"}`, `{"do":"swapNode","on":"bp1"}`,
		`{"do":"read","source":"balanceAt","address":""}`, `{"expect":"contractChecksum","is":"anything"}`,
		`{"do":"newAccount","saveKey":""}`, `{"expect":"derive","op":"sum","is":1}`,
	} {
		if _, err := PrepareTestCase(TestCaseInput{Content: editorCase(`[` + statement + `,{"expect":"blockNumber","is":1}]`)}); err == nil {
			t.Errorf("missing required argument accepted: %s", statement)
		}
	}
	raw := editorCase(`[{"expect":"derive","op":"abiCall","selector":"12345678","is":"0x12345678"}]`)
	if _, err := PrepareTestCase(TestCaseInput{Content: raw}); err != nil {
		t.Fatal("ABI calls with no arguments must remain valid:", err)
	}
}

func TestEditorPrivateKeyArgumentsRequireBindings(t *testing.T) {
	raw := editorCase(`[{"do":"sendTx","key":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","to":"node2"},{"expect":"blockNumber","is":1}]`)
	if _, err := PrepareTestCase(TestCaseInput{Content: raw}); err == nil {
		t.Fatal("literal private key reached shared API response")
	}
	raw = editorCase(`[{"do":"newAccount","save":"sender","saveKey":"senderKey"},{"do":"sendTx","key":"$senderKey","to":"node2"},{"expect":"blockNumber","is":1}]`)
	if _, err := PrepareTestCase(TestCaseInput{Content: raw}); err != nil {
		t.Fatal(err)
	}
}
