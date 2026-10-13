package feature_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	_ "github.com/0xmhha/chainbench/internal/chains/stablenet"
	_ "github.com/0xmhha/chainbench/internal/chains/wbft"
	_ "github.com/0xmhha/chainbench/internal/chains/wemix"
)

// The same binary gates WEB-14 already opens. Each chain runs serially and is
// stopped and removed before testing.T can reclaim the fixture directory.
func TestRegistryNativeComposition(t *testing.T) {
	for _, sample := range []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}} {
		t.Run(sample.chain, func(t *testing.T) {
			binary := os.Getenv(sample.variable)
			if binary == "" {
				t.Skip("native registry gate requires " + sample.variable)
			}
			bytes, err := os.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			binarySHA := sha256.Sum256(bytes)
			keys, err := filepath.Abs(filepath.Join("..", "..", "presets", "keys"))
			if err != nil {
				t.Fatal(err)
			}
			root := os.Getenv("WEBUI_RUNTIME_ROOT")
			if root == "" {
				root = t.TempDir()
			}
			fixture, err := os.MkdirTemp(root, "feature-native-")
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(fixture, "workspace")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			t.Cleanup(func() {
				cleanCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				if _, err := registered(t, "chain.stop").Invoke(cleanCtx, app.Deps{}, &app.ChainStopIn{DataDir: workspace}); err != nil {
					t.Errorf("stop fixture: %v", err)
					return
				}
				if _, err := registered(t, "chain.rm").Invoke(cleanCtx, app.Deps{}, &app.ChainRmIn{DataDir: workspace}); err != nil {
					t.Errorf("remove fixture: %v", err)
				}
			})
			up, err := registered(t, "chain.up").Invoke(ctx, app.Deps{}, &app.ChainUpIn{DataDir: workspace, Chain: sample.chain, Binary: binary, KeysDir: keys, BPCount: 4, Stage: app.UpStart})
			if err != nil {
				t.Fatal(err)
			}
			composed := up.(app.ChainUpOut)
			if len(composed.Nodes.Nodes.Nodes) != 4 || len(composed.Steps) < 9 {
				t.Fatal("registered up did not compose the requested four nodes")
			}
			for _, node := range composed.Nodes.Nodes.Nodes {
				if node.PID <= 0 || !composed.Nodes.Alive[node.Index] {
					t.Fatalf("node %d did not become an owned live process", node.Index)
				}
			}
			deadline := time.Now().Add(time.Minute)
			var readings app.NodeReadings
			for {
				value := registered(t, "node.status")
				input := value.Input()
				raw, _ := json.Marshal(map[string]any{"RPC": composed.Nodes.Nodes.Nodes[0].RPCURL})
				if err := json.Unmarshal(raw, input); err != nil {
					t.Fatal(err)
				}
				out, err := value.Invoke(ctx, app.Deps{}, input)
				if err == nil {
					readings = out.(app.NodeReadings)
				}
				if err == nil && readings.BlockNumber >= 2 && readings.PeerCount >= 3 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("registered status did not observe native block production: %v %+v", err, readings)
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(time.Second):
				}
			}
			sources := map[string]string{}
			paths, err := filepath.Glob("*.go")
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range paths {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(content)
				sources[path] = hex.EncodeToString(sum[:])
			}
			record := map[string]any{"chain": sample.chain, "binary": binary, "binarySHA256": hex.EncodeToString(binarySHA[:]), "sourceFiles": sources, "steps": composed.Steps, "nodes": composed.Nodes.Nodes, "readings": readings}
			raw, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			proof := filepath.Join(fixture, "proof.json")
			if err := os.WriteFile(proof, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("native registration proof: %s", proof)
		})
	}
}
