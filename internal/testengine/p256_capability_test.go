package testengine_test

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func p256CorpusInputs(t *testing.T) (string, string) {
	t.Helper()
	data, err := os.ReadFile("../../tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct{ Steps []struct{ Data string } }
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	valid := doc.Steps[0].Data
	bytes, err := hex.DecodeString(strings.TrimPrefix(valid, "0x"))
	if err != nil || len(bytes) != 160 {
		t.Fatal("invalid corpus P256 vector", err)
	}
	key := ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(bytes[96:128]), Y: new(big.Int).SetBytes(bytes[128:160])}
	if _, err := ecdh.P256().NewPublicKey(append([]byte{4}, bytes[96:160]...)); err != nil {
		t.Fatal("corpus vector has an invalid P256 public key", err)
	}
	if !ecdsa.Verify(&key, bytes[:32], new(big.Int).SetBytes(bytes[32:64]), new(big.Int).SetBytes(bytes[64:96])) {
		t.Fatal("corpus vector is not a valid P256 signature")
	}
	invalid := valid[:66] + strings.Repeat("0", 64) + valid[130:]
	if ecdsa.Verify(&key, bytes[:32], big.NewInt(0), new(big.Int).SetBytes(bytes[64:96])) {
		t.Fatal("zero-r negative control verifies")
	}
	return valid, invalid
}

func TestP256DeclarationRequiresValidAndInvalidSignatureBehavior(t *testing.T) {
	valid, invalid := p256CorpusInputs(t)
	for _, c := range []struct {
		name, reply, status string
		calls               int
	}{
		{"supported", "supported", "pass", 4},
		{"absent", "absent", "skip", 4},
		{"constant return is not verification", "constant", "blocked", 4},
		{"unusable result", "unusable", "blocked", 4},
		{"missing call API", "missing", "blocked", 3},
		{"HTTP failure", "http", "blocked", 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			var probes, steps atomic.Int32
			nodes := node.NodeSet{}
			for i := 0; i < 2; i++ {
				token, variable := []string{"p256-first", "p256-second"}[i], []string{"P256_NODE_FIRST", "P256_NODE_SECOND"}[i]
				t.Setenv(variable, token)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer "+token {
						http.Error(w, "denied", http.StatusUnauthorized)
						return
					}
					var request struct {
						ID     any
						Method string
						Params []json.RawMessage
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					if request.Method != "eth_call" {
						if request.Method == "test_hook" || request.Method == "test_body" {
							steps.Add(1)
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x10"})
						return
					}
					probes.Add(1)
					if len(request.Params) != 2 {
						t.Error("missing bounded probe parameters")
						return
					}
					var call map[string]any
					_ = json.Unmarshal(request.Params[0], &call)
					if call["to"] != "0x0000000000000000000000000000000000000100" || call["gas"] != "0x186a0" || (call["data"] != valid && call["data"] != invalid) || string(request.Params[1]) != `"latest"` {
						t.Errorf("incorrect crypto probe: %+v", call)
					}
					if i == 1 && c.reply == "http" {
						http.Error(w, "private details", http.StatusServiceUnavailable)
						return
					}
					if i == 1 && c.reply == "missing" {
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32601, "message": "method not found"}})
						return
					}
					var result any = "0x"
					if call["data"] == valid {
						result = "0x" + strings.Repeat("0", 63) + "1"
					}
					if i == 1 {
						switch c.reply {
						case "absent":
							result = "0x"
						case "constant":
							result = "0x" + strings.Repeat("0", 63) + "1"
						case "unusable":
							result = nil
						}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
				}))
				defer server.Close()
				nodes.Nodes = append(nodes.Nodes, node.Node{Index: i + 1, Role: node.RoleEN, RPCURL: server.URL, Auth: node.Auth{"type": "jwt", "env": variable}})
			}
			eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "wbft", Nodes: nodes, Caps: []string{"precompile:secp256r1"}, ArtifactRoot: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte(`{"schemaVersion":"2","kind":"case","id":"p256-probe","requires":["rpc","precompile:secp256r1"],"chainPreset":{"chain":"wbft"},"hooks":{"pre":[{"do":"read","source":"rpcCall","method":"test_hook","save":"height"}]},"steps":[{"expect":"rpcCall","method":"test_body","on":"node2","is":"0x10"}]}`)
			root, err := eng.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "tests", "001_p256-probe")
			data, err := os.ReadFile(filepath.Join(dir, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var verdict struct{ Result string }
			_ = json.Unmarshal(data, &verdict)
			if verdict.Result != c.status || probes.Load() != int32(c.calls) {
				t.Errorf("declaration hid P256 behavior: %s probes=%d", data, probes.Load())
			}
			if c.status != "pass" && steps.Load() != 0 {
				t.Error("unqualified precompile executed the case")
			}
			data, err = os.ReadFile(filepath.Join(dir, "observations", "precompile-capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			var observation struct {
				Capability string
				Nodes      []struct {
					Node           int
					Status, Source string
				}
			}
			if err := json.Unmarshal(data, &observation); err != nil {
				t.Fatal(err)
			}
			if observation.Capability != "precompile:secp256r1" || len(observation.Nodes) != 2 || observation.Nodes[0].Status != "supported" {
				t.Fatalf("missing crypto provenance: %s", data)
			}
		})
	}
}
