package testengine_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestNativeDeploymentPublicProvenance verifies real contract creation and
// recorded transaction facts on the three supported chains, in serial order.
func TestNativeDeploymentPublicProvenance(t *testing.T) {
	samples := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, sample := range samples {
		if os.Getenv(sample.variable) == "" {
			t.Skip("native deployment gate requires " + sample.variable)
		}
	}
	const code = "0x600a600c600039600a6000f3602a60005260206000f3"
	for _, sample := range samples {
		t.Run(sample.chain, func(t *testing.T) {
			root := os.Getenv("WEBUI_RUNTIME_ROOT")
			if root == "" {
				root = t.TempDir()
			}
			fixture, err := os.MkdirTemp(root, "deployment-native-")
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(fixture, "w")
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				if _, err := verb.ChainStop(ctx, chainsetup.Deps{}, verb.ChainStopIn{DataDir: workspace}); err != nil {
					t.Errorf("stop: %v", err)
					return
				}
				if _, err := verb.ChainRm(ctx, chainsetup.Deps{}, verb.ChainRmIn{DataDir: workspace}); err != nil {
					t.Errorf("remove node data: %v", err)
				}
			})
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "deployment-provenance", "requires": []string{"rpc"}, "chainPreset": map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "native", "chain": sample.chain, "binaries": map[string]any{"default": "native"}, "topology": map[string]any{"bp": 4}}, "steps": []map[string]any{
				{"do": "deployContract", "from": "node1", "bytecode": code, "gas": "200000", "save": "contract"},
				{"expect": "codeAt", "address": "$contract", "compare": "NotEqual", "is": "0x"},
			}})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			out, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: [][]byte{raw}, DataDir: workspace, ArtifactRoot: filepath.Join(fixture, "reports"), Binary: os.Getenv(sample.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, WaitBlocks: 2})
			if err != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("native deployment failed: %v %+v", err, out)
			}
			files, err := filepath.Glob(filepath.Join(out.SessionRoot, "tests", "*", "steps.json"))
			if err != nil || len(files) != 1 {
				t.Fatal(err, files)
			}
			content, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			var steps []session.StepResult
			if err := json.Unmarshal(content, &steps); err != nil {
				t.Fatal(err)
			}
			if len(steps) != 1 || steps[0].Deployment == nil {
				t.Fatal("no persisted deployment facts")
			}
			recorded := steps[0]
			p := recorded.Deployment
			initCode, _ := hex.DecodeString(code[2:])
			sha := sha256.Sum256(initCode)
			if p.Nonce == nil || p.Deployer == "" || p.ExpectedAddress == "" || !strings.EqualFold(p.ExpectedAddress, p.ActualAddress) || p.BytecodeSHA256 != hex.EncodeToString(sha[:]) || p.ObservationError != "" || recorded.Hash == "" || recorded.Receipt["blockNumber"] == nil || recorded.Signer != p.Deployer || recorded.Nonce != *p.Nonce {
				t.Fatalf("real deployment provenance incomplete: %+v %+v", p, recorded)
			}
			sources := map[string]string{}
			for _, pattern := range []string{"../testhelper/deployment_provenance*.go", "../testhelper/assets.go", "../core/session/record.go", "../dsl/interp/interpreter.go", "../dsl/interp/run.go", "deployment_provenance_native_test.go"} {
				paths, err := filepath.Glob(pattern)
				if err != nil {
					t.Fatal(err)
				}
				for _, path := range paths {
					bytes, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					sum := sha256.Sum256(bytes)
					sources[path] = hex.EncodeToString(sum[:])
				}
			}
			binary, err := os.ReadFile(os.Getenv(sample.variable))
			if err != nil {
				t.Fatal(err)
			}
			binarySHA := sha256.Sum256(binary)
			proof := map[string]any{"chain": sample.chain, "binary": os.Getenv(sample.variable), "binarySHA256": hex.EncodeToString(binarySHA[:]), "session": out.SessionRoot, "step": recorded, "sources": sources}
			bytes, err := json.MarshalIndent(proof, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(fixture, "proof.json")
			if err := os.WriteFile(path, bytes, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("native deployment provenance: %s", path)
		})
	}
}
