package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestCapabilityGateRecordsExactMissingRequirementsBeforeExecution(t *testing.T) {
	for _, c := range []struct {
		name, chains, status, reason string
		caps, requires, missing      []string
		skips                        []string
	}{
		{"missing", "", "skip", "missing required capabilities: ws,process-control", nil, []string{"rpc", "ws", "process-control"}, []string{"ws", "process-control"}, nil},
		{"provided", "", "pass", "", []string{"ws"}, []string{"rpc", "ws"}, nil, nil},
		{"wrong-chain", "wbft", "skip", "chain stablenet is outside applicableChains wbft", nil, []string{"rpc"}, nil, nil},
		{"unexpected", "", "fail", "missing required capabilities: ws", nil, []string{"rpc", "ws"}, []string{"ws"}, []string{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			server := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"})
			eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: []string{server.URL}, Caps: c.caps, ArtifactRoot: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			doc := map[string]any{"schemaVersion": "2", "kind": "case", "id": c.name, "chainPreset": map[string]any{"chain": "stablenet"}, "requires": c.requires, "steps": []any{map[string]any{"expect": "blockNumber", "is": 16}}}
			if c.chains != "" {
				doc["applicableChains"] = c.chains
			}
			if c.skips != nil {
				doc["skipsOn"] = c.skips
			}
			raw, _ := json.Marshal(doc)
			root, err := eng.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "tests", "001_"+c.name)
			data, err := os.ReadFile(filepath.Join(dir, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var verdict struct{ Result, Reason string }
			if err := json.Unmarshal(data, &verdict); err != nil {
				t.Fatal(err)
			}
			if verdict.Result != c.status || !strings.Contains(verdict.Reason, c.reason) {
				t.Errorf("missing precise verdict reason: %s", data)
			}
			data, err = os.ReadFile(filepath.Join(dir, "observations", "capability-gate.json"))
			if err != nil {
				t.Fatal(err)
			}
			var probe struct {
				Required, Provided, Missing []string
				Runs                        bool
			}
			if err := json.Unmarshal(data, &probe); err != nil {
				t.Fatal(err)
			}
			if strings.Join(probe.Missing, ",") != strings.Join(c.missing, ",") || strings.Join(probe.Required, ",") != strings.Join(c.requires, ",") || strings.Join(probe.Provided, ",") != strings.Join(append([]string{"rpc"}, c.caps...), ",") || probe.Runs != (c.status == "pass") {
				t.Fatalf("incorrect gate provenance: %s", data)
			}
			if c.status != "pass" {
				if _, err := os.Stat(filepath.Join(dir, "steps.json")); err == nil {
					t.Fatal("rejected case executed steps")
				}
			}
		})
	}
}
