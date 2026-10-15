package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

const (
	legacyAuthAccount  = "0x90f79bf6eb2c4f870365e785982e1f101e93b906"
	legacyBlackAccount = "0x15d34aaf54267db7d7c367839aaf71a00a2c6a65"
	legacyDualAccount  = "0x9965507d1a55bcc2695c58ba16fb37d819b0a4dc"
	legacyParamAuth    = "0x976ea74026e726554db657fa54763abd0c3a0aa9"
	legacyParamBlack   = "0x14dc79964da2c08b23698b3d3cc7ca32193d9955"
)

func legacyObject(value map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		next, _ := value[key].(map[string]any)
		if next == nil {
			return nil
		}
		value = next
	}
	return value
}

func TestLegacyExtraCasesRejectMissingDirectionsAndBits(t *testing.T) {
	for _, name := range []string{"02-stablenet-account-extra.json", "07b-extra-state-across-delayed-boho.json"} {
		mutations := []string{"valid", "auth opposite flag", "black opposite flag", "dual auth missing", "dual blacklist missing", "auth proof wrong", "black proof wrong", "dual proof wrong"}
		if strings.HasPrefix(name, "02-") {
			mutations = append(mutations, "params auth lost", "params blacklist lost")
		} else {
			mutations = append(mutations, "auth before-fork balance changed", "black before-fork balance changed", "dual boundary balance changed", "dual before-fork balance changed")
		}
		for _, mutation := range mutations {
			t.Run(name+"/"+mutation, func(t *testing.T) {
				raws, err := dsl.ReadFiles([]string{filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/extra-state", name)})
				if err != nil {
					t.Fatal(err)
				}
				var spec map[string]any
				if err = json.Unmarshal(raws[0], &spec); err != nil {
					t.Fatal(err)
				}
				overlay := legacyObject(spec, "chainPreset", "genesis", "overlay")
				bits := map[string]uint64{}
				for address, raw := range legacyObject(overlay, "alloc") {
					row, _ := raw.(map[string]any)
					extra, _ := row["extra"].(string)
					if extra == "" {
						continue
					}
					v, err := strconv.ParseUint(strings.TrimPrefix(extra, "0x"), 16, 64)
					if err != nil {
						t.Fatal(err)
					}
					bits[strings.ToLower(address)] = v
				}
				params := legacyObject(overlay, "config", "anzeon", "systemContracts", "govCouncil", "params")
				for key, mask := range map[string]uint64{"authorizedAccounts": uint64(1) << 62, "blacklist": uint64(1) << 63} {
					value, _ := params[key].(string)
					for _, address := range strings.Split(value, ",") {
						if address = strings.ToLower(strings.TrimSpace(address)); address != "" {
							bits[address] |= mask
						}
					}
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var q struct {
						ID     any    `json:"id"`
						Method string `json:"method"`
						Params []any  `json:"params"`
					}
					if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					var result any
					rpcError := false
					switch q.Method {
					case "eth_blockNumber":
						result = "0x9"
					case "eth_chainId":
						result = "0x205b"
					case "net_version":
						result = "8283"
					case "web3_clientVersion":
						result = "gstable/v1.1.0"
					case "eth_getCode":
						result = "0x6001600055"
					case "eth_call":
						input, _ := q.Params[0].(map[string]any)
						data, _ := input["data"].(string)
						if len(data) < 74 {
							rpcError = true
							break
						}
						address := "0x" + strings.ToLower(data[len(data)-40:])
						selector := data[:10]
						mask := uint64(1) << 62
						if selector == "0xfe575a87" {
							mask = uint64(1) << 63
						}
						flag := bits[address]&mask != 0
						switch mutation {
						case "auth opposite flag":
							if address == legacyAuthAccount && mask == uint64(1)<<63 {
								flag = true
							}
						case "black opposite flag":
							if address == legacyBlackAccount && mask == uint64(1)<<62 {
								flag = true
							}
						case "dual auth missing":
							if address == legacyDualAccount && mask == uint64(1)<<62 {
								flag = false
							}
						case "dual blacklist missing":
							if address == legacyDualAccount && mask == uint64(1)<<63 {
								flag = false
							}
						case "params auth lost":
							if address == legacyParamAuth {
								flag = false
							}
						case "params blacklist lost":
							if address == legacyParamBlack {
								flag = false
							}
						}
						n := 0
						if flag {
							n = 1
						}
						result = fmt.Sprintf("0x%064x", n)
					case "eth_getBalance":
						address, _ := q.Params[0].(string)
						address = strings.ToLower(address)
						result = "0xde0b6b3a7640000"
						block, _ := q.Params[1].(string)
						if (mutation == "auth before-fork balance changed" && address == legacyAuthAccount && block == "0x5") || (mutation == "black before-fork balance changed" && address == legacyBlackAccount && block == "0x5") || (mutation == "dual boundary balance changed" && address == legacyDualAccount && block == "0x6") || (mutation == "dual before-fork balance changed" && address == legacyDualAccount && block == "0x5") {
							result = "0x1"
						}
					case "eth_getProof":
						address, _ := q.Params[0].(string)
						address = strings.ToLower(address)
						extra := bits[address]
						if (mutation == "auth proof wrong" && address == legacyAuthAccount) || (mutation == "black proof wrong" && address == legacyBlackAccount) || (mutation == "dual proof wrong" && address == legacyDualAccount) {
							extra ^= uint64(1) << 62
						}
						result = map[string]any{"extra": fmt.Sprintf("0x%x", extra), "balance": "0xde0b6b3a7640000", "nonce": "0x0"}
					default:
						rpcError = true
					}
					response := map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result}
					if rpcError {
						delete(response, "result")
						response["error"] = map[string]any{"code": -32601, "message": "method outside fixture"}
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: []string{server.URL}, Caps: []string{"rpc", "fork:boho", "contract:accountManager", "contract:govCouncil", "contract:govMinter"}, ArtifactRoot: t.TempDir()})
				if err != nil {
					t.Fatal(err)
				}
				root, err := engine.Run(context.Background(), raws)
				if err != nil {
					t.Fatal(err)
				}
				b, err := os.ReadFile(filepath.Join(root, "session.json"))
				if err != nil {
					t.Fatal(err)
				}
				var session struct {
					Summary struct{ Pass, Fail, Blocked, Skip int }
				}
				if err = json.Unmarshal(b, &session); err != nil {
					t.Fatal(err)
				}
				pass, fail := 0, 1
				if mutation == "valid" {
					pass, fail = 1, 0
				}
				if session.Summary.Pass != pass || session.Summary.Fail != fail || session.Summary.Blocked != 0 || session.Summary.Skip != 0 {
					t.Fatalf("fault %q hidden: %+v", mutation, session.Summary)
				}
			})
		}
	}
}
