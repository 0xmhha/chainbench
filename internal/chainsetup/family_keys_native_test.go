package chainsetup_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/preset"
)

// TestNativeBlueprintPinnedGeneration runs the public CLI against all three
// real chains, serially, and reclaims owned node data before each subtest ends.
func TestNativeBlueprintPinnedGeneration(t *testing.T) {
	samples := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, sample := range samples {
		if os.Getenv(sample.variable) == "" {
			t.Skip("native pinned generation requires " + sample.variable)
		}
	}
	executable := filepath.Join(t.TempDir(), "chainbench")
	build := exec.Command("go", "build", "-o", executable, "./cmd/chainbench")
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, executable, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return output, fmt.Errorf("CLI %v: %w\n%s", args, err, output)
		}
		return output, nil
	}
	for _, sample := range samples {
		t.Run(sample.chain, func(t *testing.T) {
			root := os.Getenv("WEBUI_RUNTIME_ROOT")
			if root == "" {
				root = t.TempDir()
			}
			fixture, err := os.MkdirTemp(root, "n3-native-")
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(fixture, "w")
			keys := filepath.Join(fixture, "keys")
			blueprint := filepath.Join(fixture, "network.yaml")
			inventory := filepath.Join(fixture, "server-set.yaml")
			if err := os.WriteFile(blueprint, []byte("chain: "+sample.chain+"\nnodes:\n - {role: bp, server: local}\n - {role: bp, server: local}\n - {role: bp, server: local}\n - {role: bp, server: local}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(inventory, []byte(localInventory), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				for _, operation := range []string{"stop", "rm"} {
					if _, err := run(ctx, "chain", operation, "--workspace-dir", workspace); err != nil {
						t.Errorf("cleanup: %v", err)
						return
					}
				}
			})
			t.Cleanup(func() {
				// Preserve diagnostics before the earlier cleanup reclaims node
				// data, so a failed native gate retains its actual cause.
				state, err := chainsetup.Open(workspace, fixedClock())
				if err != nil {
					return
				}
				for _, node := range state.State().Nodes {
					for kind, path := range map[string]string{"config": node.ConfigPath, "log": node.LogPath} {
						content, err := os.ReadFile(path)
						if err == nil {
							destination := filepath.Join(fixture, fmt.Sprintf("node%d.%s", node.Index, kind))
							if err := os.WriteFile(destination, content, 0o600); err != nil {
								t.Errorf("preserve native diagnostic: %v", err)
							}
						}
					}
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			output, err := run(ctx, "chain", "up", "--chain", sample.chain, "--binary", os.Getenv(sample.variable), "--workspace-dir", workspace, "--keys", keys, "--keys-source", "generate", "--blueprint", blueprint, "--server-set", inventory, "--server", "local", "--json")
			if err != nil {
				t.Fatal(err)
			}
			var up app.ChainUpOut
			if err := json.Unmarshal(output, &up); err != nil {
				t.Fatalf("CLI did not return JSON: %v\n%s", err, output)
			}
			if len(up.Steps) < 9 || len(up.Nodes.Nodes.Nodes) != 4 {
				t.Fatal("CLI did not report the complete four-node composition")
			}
			state, err := chainsetup.Open(workspace, fixedClock())
			if err != nil {
				t.Fatal(err)
			}
			if len(state.State().Nodes) != 4 {
				t.Fatal("wrong native topology")
			}
			for _, node := range state.State().Nodes {
				if node.Server != "" || node.Host != "127.0.0.1" || node.HTTP != 8545+(node.Index-1)*10 || node.PID <= 0 {
					t.Fatalf("pin did not reach native node: %+v", node)
				}
			}
			identities, err := preset.LoadKeyPreset(keys)
			if err != nil {
				t.Fatal(err)
			}
			var bls []bool
			for _, identity := range identities.Nodes {
				present := identity.BLS != nil
				bls = append(bls, present)
				if present != (sample.chain != "wemix") {
					t.Fatal("wrong native family derivation")
				}
			}
			deadline := time.Now().Add(time.Minute)
			var readings app.NodeReadings
			for {
				readings, err = app.ReadNode(ctx, app.Deps{}, up.Nodes.Nodes.Nodes[0].RPCURL)
				if err == nil && readings.BlockNumber >= 2 && readings.PeerCount >= 3 {
					break
				}
				if time.Now().After(deadline) {
					for _, node := range up.Nodes.Nodes.Nodes {
						reading, err := app.ReadNode(ctx, app.Deps{}, node.RPCURL)
						t.Logf("node%d native reading: %+v, error: %v", node.Index, reading, err)
					}
					t.Fatalf("native CLI composition did not seal: %v %+v", err, readings)
				}
				time.Sleep(time.Second)
			}
			sources := map[string]string{}
			for _, pattern := range []string{"*.go", "../resource/*.go", "../core/keyring/store/source.go", "../core/node/launchreq.go"} {
				paths, err := filepath.Glob(pattern)
				if err != nil {
					t.Fatal(err)
				}
				for _, path := range paths {
					bytes, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					sha := sha256.Sum256(bytes)
					sources[path] = hex.EncodeToString(sha[:])
				}
			}
			binary, err := os.ReadFile(os.Getenv(sample.variable))
			if err != nil {
				t.Fatal(err)
			}
			binarySHA := sha256.Sum256(binary)
			proof := map[string]any{"chain": sample.chain, "binary": os.Getenv(sample.variable), "binarySHA256": hex.EncodeToString(binarySHA[:]), "sources": sources, "nodes": state.State().Nodes, "BLS": bls, "readings": readings, "steps": up.Steps}
			bytes, err := json.MarshalIndent(proof, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(fixture, "proof.json")
			if err := os.WriteFile(path, bytes, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("native pinned-generation proof: %s", path)
		})
	}
}
