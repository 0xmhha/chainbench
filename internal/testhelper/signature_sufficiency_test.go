package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type signatureFixture struct {
	fakeNodeControl
	mu                sync.Mutex
	fault             string
	swapped           map[int]bool
	secondHistoryRead bool
}

func (c *signatureFixture) Swap(_ context.Context, n node.Node, _ interp.NodeChange) (node.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.swapped[n.Index] = true
	return n, nil
}

func (c *signatureFixture) Log(_ context.Context, n node.Node, limit int) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fault == "missing all EN logs" {
		return "", nil
	}
	if c.fault == "partial signature boundary" && n.Index == 6 {
		if c.swapped[n.Index] {
			return "boot\nSignature verification FAILED\n", nil
		}
		return "boot\nSignature", nil
	}
	text := "historical invalid signature\nboot\n"
	if c.swapped[n.Index] {
		text += "restart\nimported block\n"
		if c.fault == "new signature error" && n.Index == 6 {
			text += "Signature verification FAILED\n"
		}
		if c.fault == "lost log prefix" && n.Index == 5 {
			text = "rotated\nimported block\n"
		}
		if c.fault == "truncated log" && n.Index == 6 {
			text += strings.Repeat("x", 1<<20)
		}
	}
	if c.fault == "second EN log unavailable" && n.Index == 6 {
		return "", fmt.Errorf("log unavailable")
	}
	if len(text) > limit {
		text = text[len(text)-limit:]
	}
	return text, nil
}

func TestPublicSignatureSwapRequiresPreservedBlocksCurrentHeadsAndCleanLogs(t *testing.T) {
	for _, fault := range []string{"valid", "legacy transaction reverted", "missing legacy block", "changed legacy block", "changed receipt block hash", "changed receipt transaction hash", "source head stalled", "second EN stale", "second EN exactly two blocks behind", "second EN three blocks behind", "second EN still syncing", "second EN same version", "second EN wrong sender", "new signature error", "lost log prefix", "truncated log", "second EN log unavailable", "missing all EN logs", "partial signature boundary"} {
		t.Run(fault, func(t *testing.T) {
			const sender = "0x1212121212121212121212121212121212121212"
			const recipient = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
			txhash, blockhash := "0x"+strings.Repeat("ab", 32), "0x"+strings.Repeat("cd", 32)
			fixture := &signatureFixture{fault: fault, swapped: map[int]bool{}}
			env := envWithNodes(t, 1, "http://unused")
			nodes := []node.Node{}
			for _, index := range []int{1, 5, 6} {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fixture.mu.Lock()
					defer fixture.mu.Unlock()
					var q struct {
						ID     any
						Method string
						Params []any
					}
					if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
						t.Error(err)
						return
					}
					anySwapped := fixture.swapped[5] || fixture.swapped[6]
					var result any
					switch q.Method {
					case "eth_blockNumber":
						height := 10
						if anySwapped && fault != "source head stalled" {
							height = 20
						}
						if index == 6 && fixture.swapped[6] && fault == "second EN stale" {
							height = 3
						}
						if index == 1 && fixture.secondHistoryRead && fault == "second EN exactly two blocks behind" {
							height = 22
						}
						if index == 1 && fixture.secondHistoryRead && fault == "second EN three blocks behind" {
							height = 23
						}
						result = fmt.Sprintf("0x%x", height)
					case "web3_clientVersion":
						result = "gstable/v1.0.1"
						if fixture.swapped[index] && (fault != "second EN same version" || index != 6) {
							result = "gstable/v1.1.0"
						}
					case "eth_getTransactionReceipt":
						status := "0x1"
						if fault == "legacy transaction reverted" {
							status = "0x0"
						}
						hash, transactionHash := blockhash, txhash
						if fixture.swapped[index] && index == 6 && fault == "changed receipt block hash" {
							hash = "0x" + strings.Repeat("ef", 32)
						}
						if fixture.swapped[index] && index == 6 && fault == "changed receipt transaction hash" {
							transactionHash = "0x" + strings.Repeat("ef", 32)
						}
						result = map[string]any{"blockNumber": "0x3", "status": status, "blockHash": hash, "transactionHash": transactionHash, "from": sender, "to": recipient}
					case "eth_getTransactionByHash":
						from := sender
						if fixture.swapped[index] && index == 6 && fault == "second EN wrong sender" {
							from = "0x" + strings.Repeat("34", 20)
						}
						result = map[string]any{"hash": txhash, "blockNumber": "0x3", "blockHash": blockhash, "from": from, "to": recipient}
					case "eth_getBlockByNumber":
						if index == 6 {
							fixture.secondHistoryRead = true
						}
						hash := blockhash
						if index == 6 && fault == "changed legacy block" {
							hash = "0x" + strings.Repeat("ef", 32)
						}
						result = map[string]any{"hash": hash, "number": "0x3"}
						if fault == "missing legacy block" {
							result = nil
						}
					case "eth_syncing":
						result = false
						if index == 6 && fault == "second EN still syncing" {
							result = map[string]any{"currentBlock": "0x3", "highestBlock": "0x14"}
						}
					default:
						t.Error("unexpected public observation", q.Method)
					}
					if err := json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result}); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				role := node.RoleEN
				if index == 1 {
					role = node.RoleBP
				}
				nodes = append(nodes, node.Node{Index: index, Role: role, Host: "127.0.0.1", RPCURL: server.URL})
			}
			env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
			raws, err := dsl.ReadFiles([]string{"../../tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json"})
			if err != nil {
				t.Fatal(err)
			}
			spec, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			selected := []dsl.Statement{}
			for _, st := range spec.Sequence {
				// Only account creation/funding/signing are fixture setup. Native execution
				// exercises those; retain every public observation, control and assertion.
				if st.Do == "newAccount" || st.Do == "sendTx" {
					continue
				}
				if st.Do == "waitBlock" || st.Do == "waitFor" {
					st.Args["timeout"] = "200ms"
					st.Args["pollInterval"] = "1ms"
				}
				selected = append(selected, st)
			}
			spec.Sequence = selected
			spec.EnvValues = map[string]any{"sender": sender, "legacyTx": txhash}
			d := faultDeps(fixture)
			rec := &recordStub{}
			status, err := interp.NewInterpreter(d).Run(context.Background(), spec, env, rec)
			want := session.StatusFail
			if fault == "valid" || fault == "second EN exactly two blocks behind" {
				want = session.StatusPass
			}
			if err != nil || status != want {
				t.Fatalf("status=%s want %s err=%v assertions=%+v", status, want, err, rec.asserts)
			}
			if want == session.StatusPass && (!fixture.swapped[5] || !fixture.swapped[6]) {
				t.Fatalf("both ENs must be swapped: %v", fixture.swapped)
			}
		})
	}
}
