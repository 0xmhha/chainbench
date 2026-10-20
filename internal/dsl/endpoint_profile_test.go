package dsl_test

import (
	"github.com/0xmhha/chainbench/internal/dsl"
	"testing"
)

func TestStructuredAttachEndpointProfileParses(t *testing.T) {
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"profile","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"external","chain":"wbft","attach":{"endpoints":[{"rpc":"http://127.0.0.1:8545","role":"bp","ws":"ws://127.0.0.1:8546","metrics":"http://127.0.0.1:6060/metrics","auth":{"type":"jwt","env":"CHAINBENCH_TEST_TOKEN"},"apis":["eth","net"]},{"rpc":"http://127.0.0.1:8555","role":"en"}]}},"steps":[{"expect":"blockNumber","on":"en1","compare":"GreaterOrEqual","is":"0"}]}`)
	spec, err := dsl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if spec.EnvAttach == nil {
		t.Fatal("endpoint profile was discarded")
	}
}

func TestStructuredAttachProfilesRefuseAmbiguousOrSecretInputs(t *testing.T) {
	for _, profile := range []string{
		`{"rpc":["http://one"],"endpoints":[{"rpc":"http://two"}]}`,
		`{"endpoints":[{"rpc":"http://one","role":"miner"}]}`,
		`{"endpoints":[{"rpc":"file:///fixture"}]}`,
		`{"endpoints":[{"rpc":"http://user:password@one"}]}`,
		`{"endpoints":[{"rpc":"http://one","ws":"http://one"}]}`,
		`{"endpoints":[{"rpc":"http://one","auth":{"type":"jwt","env":"TOKEN","value":"literal-secret"}}]}`,
		`{"endpoints":[{"rpc":"http://one","auth":{"type":"jwt"}}]}`,
		`{"endpoints":[{"rpc":"http://one","apis":[""]}]}`,
		`{"endpoints":[{"rpc":"http://one","auth":{"type":"jwt","env":"TOKEN","header":"Ignored"}}]}`,
		`{"endpoints":[{"rpc":"http://one","auth":{"type":"api-key","env":"TOKEN","header":"Invalid Header"}}]}`,
		`{"endpoints":[{"rpc":"http://one","auth":{"type":"jwt","env":"TOKEN=literal"}}]}`,
		`{"endpoints":[{"rpc":"http://one","httpPort":65536}]}`,
	} {
		raw := []byte(`{"schemaVersion":"2","kind":"case","id":"profile","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"external","chain":"wbft","attach":` + profile + `},"steps":[{"expect":"blockNumber","is":"0"}]}`)
		if _, err := dsl.Parse(raw); err == nil {
			t.Fatalf("accepted unsupported profile: %s", profile)
		}
	}
}
