package testengine_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeConsensusCapabilityProbes(t *testing.T) {
	samples := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, sample := range samples {
		if os.Getenv(sample.variable) == "" {
			t.Skip("native capability gate requires " + sample.variable)
		}
	}
	for _, sample := range samples {
		t.Run(sample.chain, func(t *testing.T) {
			root := os.Getenv("WEBUI_RUNTIME_ROOT")
			if root == "" {
				root = t.TempDir()
			}
			fixture, err := os.MkdirTemp(root, "capability-native-")
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(fixture, "w")
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				if _, err := verb.ChainStop(ctx, chainsetup.Deps{}, verb.ChainStopIn{DataDir: workspace}); err != nil {
					t.Error(err)
					return
				}
				if _, err := verb.ChainRm(ctx, chainsetup.Deps{}, verb.ChainRmIn{DataDir: workspace}); err != nil {
					t.Error(err)
				}
			})
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "runtime-consensus", "requires": []string{"rpc", "consensus"}, "chainPreset": map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "native", "chain": sample.chain, "binaries": map[string]any{"default": "native"}, "topology": map[string]any{"bp": 4}}, "steps": []map[string]any{{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "2"}}})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			output, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{DataDir: workspace, SpecContent: [][]byte{raw}, Binary: os.Getenv(sample.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, ArtifactRoot: filepath.Join(fixture, "reports"), WaitBlocks: 2})
			if err != nil || output.Summary.Summary.Pass != 1 || output.Summary.Failed() || output.Summary.Summary.Skip != 0 {
				t.Fatalf("native consensus probe: %v %+v", err, output)
			}
			observation := filepath.Join(output.SessionRoot, "tests", "001_runtime-consensus", "observations", "capability-probes.json")
			bytes, err := os.ReadFile(observation)
			if err != nil {
				t.Fatal(err)
			}
			var facts struct {
				Status, Source string
				ValidatorCount int `json:"validatorCount"`
			}
			if err := json.Unmarshal(bytes, &facts); err != nil || facts.Status != "supported" || facts.Source == "" || facts.ValidatorCount < 4 {
				t.Fatalf("native facts: %v %s", err, bytes)
			}
			sources := map[string]string{}
			for _, pattern := range []string{"capability_probe*.go", "attach.go", "engine_impl.go", "../core/registry/consensus.go", "../consensus/poa/validators*.go"} {
				files, err := filepath.Glob(pattern)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range files {
					content, err := os.ReadFile(name)
					if err != nil {
						t.Fatal(err)
					}
					sum := sha256.Sum256(content)
					sources[name] = hex.EncodeToString(sum[:])
				}
			}
			binary, err := os.ReadFile(os.Getenv(sample.variable))
			if err != nil {
				t.Fatal(err)
			}
			binarySHA := sha256.Sum256(binary)
			proof := map[string]any{"chain": sample.chain, "binary": os.Getenv(sample.variable), "binarySHA256": hex.EncodeToString(binarySHA[:]), "sources": sources, "observation": observation, "facts": facts, "session": output.SessionRoot}
			content, err := json.MarshalIndent(proof, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(fixture, "proof.json")
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("native capability proof: %s", path)
		})
	}
}
