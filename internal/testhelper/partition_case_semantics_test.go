package testhelper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// partitionCase loads the public procedure with the selected native profile.
func partitionCase(t *testing.T, chain string) dsl.Spec {
	t.Helper()
	raw, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/fault/CT-FAULT-004-network-partition.json"}, "../../presets/chain/"+chain+"-bp9.json")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := dsl.ParseV2(raw[0])
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

// partitionFixture provides per-node RPC observations without programming a firewall.
type partitionFixture struct {
	mu            sync.Mutex
	heads         map[int]int
	syncReads     map[int]int
	majorityDelta int
	minorityNode  int
	badSyncNode   int
	catchupNode   int
}

type partitionRecorder struct {
	recordStub
	steps []session.StepResult
}

func (r *partitionRecorder) Step(_ int, step session.StepResult) {
	r.steps = append(r.steps, step)
}

func (f *partitionFixture) environment(t *testing.T) session.Environment {
	t.Helper()
	f.heads, f.syncReads = map[int]int{}, map[int]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			t.Error(err)
			return
		}
		index, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/node"))
		if err != nil {
			t.Error(err)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		var value any
		switch q.Method {
		case "eth_blockNumber":
			height := 100
			if index == f.catchupNode && f.heads[index] == 0 {
				height = 99
			} else if f.heads[index] > 0 {
				switch index {
				case 9:
					height += f.majorityDelta
				case f.minorityNode:
					height += 3
				}
			}
			f.heads[index]++
			value = fmt.Sprintf("0x%x", height)
		case "eth_syncing":
			f.syncReads[index]++
			value = index == f.badSyncNode
		case "eth_getBlockByNumber":
			if len(q.Params) == 0 {
				t.Error("missing block number")
				return
			}
			block, err := strconv.ParseUint(fmt.Sprint(q.Params[0]), 0, 64)
			if err != nil {
				t.Error(err)
				return
			}
			value = map[string]any{"hash": fmt.Sprintf("0x%064x", block)}
			if index == f.catchupNode && f.heads[index] < 2 && block > 99 {
				value = nil
			}
		default:
			t.Errorf("unexpected RPC %s", q.Method)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": value})
	}))
	t.Cleanup(srv.Close)
	env := envWithNodes(t, 9, srv.URL)
	nodes := make([]node.Node, 9)
	for i := range nodes {
		nodes[i] = node.Node{Index: i + 1, Role: node.RoleBP, RPCURL: fmt.Sprintf("%s/node%d", srv.URL, i+1)}
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
	return env
}

func TestPartitionCaseUsesChainSpecificQuorum(t *testing.T) {
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		for _, delta := range []int{-1, 0, 2, 3, 26} {
			t.Run(fmt.Sprintf("%s/delta%d", chain, delta), func(t *testing.T) {
				s := partitionCase(t, chain)
				start, end := -1, -1
				for i, st := range s.Sequence {
					if st.Do == actionPartition {
						start = i + 1
					}
					if st.Do == actionHealPartition {
						end = i
					}
				}
				if start < 0 || end <= start {
					t.Fatal("public case has no partition observation window")
				}
				s.Sequence = s.Sequence[start:end]
				for _, st := range s.Sequence {
					if st.Expect == assertBlockHalt {
						if st.Args["within"] != "30s" {
							t.Fatalf("public observation window changed: %v", st.Args)
						}
						st.Args["within"] = "1ms"
					}
				}
				fixture := &partitionFixture{majorityDelta: delta}
				rec := &recordStub{}
				got, err := interp.NewInterpreter(deps()).Run(t.Context(), s, fixture.environment(t), rec)
				want := session.StatusFail
				if (chain == "wemix" && delta > 2) || (chain != "wemix" && delta >= 0 && delta <= 2) {
					want = session.StatusPass
				}
				if err != nil || got != want {
					t.Fatalf("public partition status=%s err=%v want=%s assertions=%+v", got, err, want, rec.asserts)
				}
			})
		}
	}
}

func TestPartitionCaseObservesEveryMinorityNode(t *testing.T) {
	for _, moving := range []int{1, 2, 3, 4} {
		t.Run(strconv.Itoa(moving), func(t *testing.T) {
			s := partitionCase(t, "stablenet")
			start, end := -1, -1
			for i, st := range s.Sequence {
				if st.Do == actionPartition {
					start = i + 1
				}
				if st.Do == actionHealPartition {
					end = i
				}
			}
			if start < 0 || end <= start {
				t.Fatal("public case has no partition observation window")
			}
			s.Sequence = s.Sequence[start:end]
			for _, st := range s.Sequence {
				if st.Expect == assertBlockHalt {
					st.Args["within"] = "1ms"
				}
			}
			fixture := &partitionFixture{minorityNode: moving}
			got, err := interp.NewInterpreter(deps()).Run(t.Context(), s, fixture.environment(t), &recordStub{})
			if err != nil || got != session.StatusFail {
				t.Fatalf("moving minority node%d was ignored: %s, %v", moving, got, err)
			}
		})
	}
}

func TestPartitionCaseWaitsForEveryNodeToSynchronize(t *testing.T) {
	for bad := 0; bad <= 9; bad++ {
		t.Run(strconv.Itoa(bad), func(t *testing.T) {
			s := partitionCase(t, "stablenet")
			start := -1
			for i, st := range s.Sequence {
				if st.Do == actionHealPartition {
					start = i + 1
				}
			}
			if start < 0 {
				t.Fatal("public case has no recovery sequence")
			}
			s.Sequence = s.Sequence[start:]
			// Exercise the public synchronization and final hash checks; block
			// advance is covered by the actual three-chain native procedure.
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Expect == assertBlockAdvance {
					continue
				}
				if st.Do == actionWaitFor {
					st.Args["timeout"], st.Args["pollInterval"] = "2s", "1ms"
					if bad != 0 && st.Args["method"] == "eth_syncing" {
						st.Args["timeout"] = "20ms"
					}
				}
				selected = append(selected, st)
			}
			s.Sequence = selected
			fixture := &partitionFixture{badSyncNode: bad}
			rec := &partitionRecorder{}
			got, err := interp.NewInterpreter(deps()).Run(t.Context(), s, fixture.environment(t), rec)
			want := session.StatusPass
			if bad != 0 {
				want = session.StatusFail
			}
			if err != nil || got != want {
				t.Fatalf("synchronizing node%d: status=%s err=%v want=%s", bad, got, err, want)
			}
			fixture.mu.Lock()
			checked := len(fixture.syncReads)
			fixture.mu.Unlock()
			if bad == 0 && checked != 9 {
				t.Fatalf("only %d of nine nodes checked for synchronization", checked)
			}
			if bad != 0 {
				if len(rec.steps) == 0 {
					t.Fatal("missing failed synchronization step")
				}
				last := rec.steps[len(rec.steps)-1]
				if last.On != "node"+strconv.Itoa(bad) || last.Error == "" {
					t.Fatalf("failure must come from the unsynchronized node%d, got %+v", bad, last)
				}
			}
		})
	}
}

func TestPartitionCaseWaitsForRecoveredHeightBeforeComparingHashes(t *testing.T) {
	s := partitionCase(t, "stablenet")
	selected := []dsl.Statement{}
	recovering := false
	for _, st := range s.Sequence {
		if st.Do == actionHealPartition {
			recovering = true
			continue
		}
		if !recovering || st.Expect == assertBlockAdvance {
			continue
		}
		if st.Do == actionWaitFor {
			st.Args["timeout"], st.Args["pollInterval"] = "2s", "1ms"
		}
		selected = append(selected, st)
	}
	if len(selected) == 0 {
		t.Fatal("public case has no recovery checks")
	}
	s.Sequence = selected
	fixture := &partitionFixture{catchupNode: 2}
	rec := &partitionRecorder{}
	got, err := interp.NewInterpreter(deps()).Run(t.Context(), s, fixture.environment(t), rec)
	if err != nil || got != session.StatusPass {
		t.Fatalf("healthy delayed node must catch up before hash comparison: %s, %v, assertions=%+v", got, err, rec.asserts)
	}
	if len(rec.asserts) != 1 || !strings.Contains(fmt.Sprint(rec.asserts[0].Expected), "block 0x64 hash") {
		t.Fatalf("recovery must compare reference height100 rather than fall back to the delayed height99: %+v", rec.asserts)
	}
	fixture.mu.Lock()
	observations := fixture.heads[2]
	fixture.mu.Unlock()
	if observations < 2 {
		t.Fatalf("delayed node was not observed catching up: %d reads", observations)
	}
}
