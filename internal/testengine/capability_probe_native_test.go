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
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
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
			output, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{DataDir: workspace, SpecContent: [][]byte{raw}, Binary: os.Getenv(sample.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, ArtifactRoot: filepath.Join(fixture, "reports"), WaitBlocks: 2, KeepUp: true})
			if err != nil || output.Summary.Summary.Pass != 1 || output.Summary.Failed() || output.Summary.Summary.Skip != 0 {
				t.Fatalf("native consensus probe: %v %+v", err, output)
			}
			if output.Stopped || len(output.Endpoints) != 4 {
				t.Fatal("native identity probe requires four still-running nodes")
			}
			endpoint := node.Node{Index: 1, RPCURL: output.Endpoints[0]}
			var client string
			if err := rpc.Dial(endpoint.RPCURL).Call(ctx, "web3_clientVersion", &client); err != nil || client == "" {
				t.Fatalf("native client name: %q %v", client, err)
			}
			if err := testengine.VerifyAttachedNode(ctx, sample.chain, endpoint, false); err != nil {
				t.Fatalf("actual native build alias refused: %v", err)
			}
			if sample.chain != "stablenet" {
				if err := testengine.VerifyAttachedNode(ctx, "stablenet", endpoint, false); err == nil {
					t.Fatal("actual non-stablenet client accepted as stablenet")
				}
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
			for _, pattern := range []string{"capability_probe*.go", "attach.go", "attach_verify*.go", "engine_impl.go", "../core/registry/consensus.go", "../consensus/poa/validators*.go", "../chains/*/manifest.json"} {
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
			proof := map[string]any{"chain": sample.chain, "binary": os.Getenv(sample.variable), "binarySHA256": hex.EncodeToString(binarySHA[:]), "sources": sources, "observation": observation, "facts": facts, "session": output.SessionRoot, "clientVersion": client, "clientBuildAliasAccepted": true}
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
