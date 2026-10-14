package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testengine"
	"github.com/gorilla/websocket"
)

func TestNetworkCapabilitiesProbeEveryNodeBeforeHooks(t *testing.T) {
	for _, c := range []struct {
		name, capability, want string
		replies                []string
	}{
		{"RPC reachable", "rpc", "pass", []string{"supported", "supported"}},
		{"RPC method absent", "rpc", "skip", []string{"supported", "unsupported"}},
		{"RPC malformed", "rpc", "blocked", []string{"supported", "malformed"}},
		{"contract code present", "contract:govCouncil", "pass", []string{"supported", "supported"}},
		{"contract absent later", "contract:govCouncil", "skip", []string{"supported", "unsupported"}},
		{"contract unreadable later", "contract:govCouncil", "blocked", []string{"supported", "unknown"}},
		{"unknown outranks absent", "contract:govCouncil", "blocked", []string{"unsupported", "unknown"}},
		{"contract malformed", "contract:govCouncil", "blocked", []string{"supported", "malformed"}},
		{"builtin account manager", "contract:accountManager", "pass", []string{"supported", "supported"}},
		{"builtin account manager absent", "contract:accountManager", "skip", []string{"supported", "unsupported"}},
		{"WS accepts subscriptions", "ws", "pass", []string{"supported", "supported"}},
		{"WS method missing", "ws", "skip", []string{"supported", "unsupported"}},
		{"WS malformed acknowledgement", "ws", "blocked", []string{"supported", "malformed"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var body, pre, rpcProbes, capProbes, closed atomic.Int32
			nodes := node.NodeSet{}
			for index, reply := range c.replies {
				variable := fmt.Sprintf("NETWORK_CAP_AUTH_%d", index)
				t.Setenv(variable, "private-fixture-token")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer private-fixture-token" {
						http.Error(w, "private token detail", http.StatusUnauthorized)
						return
					}
					if websocket.IsWebSocketUpgrade(r) {
						capProbes.Add(1)
						up := websocket.Upgrader{}
						conn, err := up.Upgrade(w, r, nil)
						if err != nil {
							return
						}
						defer conn.Close()
						defer closed.Add(1)
						var req map[string]any
						if err := conn.ReadJSON(&req); err != nil {
							return
						}
						switch reply {
						case "unsupported":
							_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "error": map[string]any{"code": -32601, "message": "private unsupported detail"}})
						case "malformed":
							_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "result": ""})
						default:
							_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "0xsub"})
						}
						for {
							if _, _, err := conn.ReadMessage(); err != nil {
								return
							}
						}
					}
					var req struct {
						ID     any
						Method string
					}
					_ = json.NewDecoder(r.Body).Decode(&req)
					var result any = "0x10"
					switch req.Method {
					case "eth_blockNumber":
						rpcProbes.Add(1)
						if c.capability == "rpc" {
							if reply == "unsupported" {
								_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32601, "message": "missing"}})
								return
							}
							if reply == "malformed" {
								result = "0x00"
							}
						}
					case "eth_getCode", "eth_call":
						capProbes.Add(1)
						switch reply {
						case "unsupported":
							result = "0x"
						case "unknown":
							http.Error(w, "private upstream detail", http.StatusServiceUnavailable)
							return
						case "malformed":
							result = "0x0"
						default:
							result = "0x6000"
							if c.capability == "contract:accountManager" {
								result = "0x" + strings.Repeat("0", 64)
							}
						}
					default:
						body.Add(1)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
				}))
				t.Cleanup(server.Close)
				nodes.Nodes = append(nodes.Nodes, node.Node{Index: index + 1, Role: node.RoleEN, RPCURL: server.URL, WSURL: "ws" + strings.TrimPrefix(server.URL, "http"), Auth: node.Auth{"type": "jwt", "env": variable}})
			}
			engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", Nodes: nodes, Caps: []string{c.capability}, ArtifactRoot: t.TempDir(), PreSpec: func(context.Context, session.Environment) error { pre.Add(1); return nil }})
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "network-caps", "chainPreset": map[string]any{"chain": "stablenet"}, "requires": []string{"rpc", c.capability}, "hooks": map[string]any{"pre": []any{map[string]any{"do": "read", "source": "rpcCall", "method": "test_hook", "save": "value"}}}, "steps": []any{map[string]any{"expect": "rpcCall", "method": "test_body", "is": "0x10"}}})
			root, err := engine.Run(t.Context(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "tests", "001_network-caps")
			data, err := os.ReadFile(filepath.Join(path, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var status struct{ Result string }
			_ = json.Unmarshal(data, &status)
			if status.Result != c.want {
				t.Fatalf("declared claims hid runtime capability: want=%s %s", c.want, data)
			}
			if c.want != "pass" && (body.Load() != 0 || pre.Load() != 0) {
				t.Fatalf("inadequate endpoints executed hooks/body: %d/%d", body.Load(), pre.Load())
			}
			if rpcProbes.Load() != 2 || c.capability != "rpc" && capProbes.Load() != 2 {
				t.Fatalf("not every node was checked: rpc=%d capability=%d", rpcProbes.Load(), capProbes.Load())
			}
			data, err = os.ReadFile(filepath.Join(path, "observations", "network-capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "private") {
				t.Fatal("private errors or credentials reached evidence", string(data))
			}
			var observed struct {
				Probes []struct {
					Capability string
					Nodes      []struct {
						Node           int
						Status, Source string
					}
				}
			}
			if err := json.Unmarshal(data, &observed); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, probe := range observed.Probes {
				if probe.Capability == c.capability {
					found = len(probe.Nodes) == 2
					for index, n := range probe.Nodes {
						if n.Node != index+1 || n.Source == "" {
							t.Fatal("node provenance missing", string(data))
						}
					}
				}
			}
			if !found {
				t.Fatal("capability observations missing", string(data))
			}
		})
	}
}

func TestUnknownNetworkProbeOutranksAnotherUnsupportedCapability(t *testing.T) {
	var body, consensus, contracts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any
			Method string
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch req.Method {
		case "istanbul_getValidators":
			consensus.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32601, "message": "unsupported"}})
		case "eth_getCode":
			contracts.Add(1)
			http.Error(w, "private unavailable detail", http.StatusServiceUnavailable)
		case "eth_blockNumber":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x1"})
		default:
			body.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x1"})
		}
	}))
	defer server.Close()
	engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: []string{server.URL}, Caps: []string{"consensus", "contract:govCouncil"}, ArtifactRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"mixed-availability","requires":["rpc","consensus","contract:govCouncil"],"chainPreset":{"chain":"stablenet"},"hooks":{"pre":[{"do":"read","source":"rpcCall","method":"test_hook","save":"value"}]},"steps":[{"expect":"rpcCall","method":"test_body","is":"0x1"}]}`)
	root, err := engine.Run(t.Context(), [][]byte{raw})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "tests", "001_mixed-availability", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var status struct{ Result string }
	_ = json.Unmarshal(data, &status)
	if status.Result != "blocked" || body.Load() != 0 || consensus.Load() != 1 || contracts.Load() != 1 {
		t.Fatalf("unsupported hid unknown or ran hooks: %s body=%d consensus=%d contracts=%d", data, body.Load(), consensus.Load(), contracts.Load())
	}
}
