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
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// The node supplies matching historical blocks even when its current head lags.
// The test therefore distinguishes catch-up from historical hash agreement.
func faultCheckpointNode(t *testing.T, head uint64, sequence ...uint64) string {
	t.Helper()
	var mutex sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var result any
		switch req.Method {
		case "eth_blockNumber":
			if len(sequence) != 0 {
				head = sequence[min(calls, len(sequence)-1)]
				calls++
			}
			result = fmt.Sprintf("0x%x", head)
		case "eth_getBlockByNumber":
			var tag string
			if len(req.Params) == 0 || json.Unmarshal(req.Params[0], &tag) != nil {
				http.Error(w, "missing block tag", http.StatusBadRequest)
				return
			}
			var height uint64
			if tag == "latest" {
				height = head
			} else if _, err := fmt.Sscanf(tag, "0x%x", &height); err != nil {
				http.Error(w, "invalid block tag", http.StatusBadRequest)
				return
			}
			if height <= head {
				result = map[string]any{"number": fmt.Sprintf("0x%x", height), "hash": fmt.Sprintf("0x%064x", height+1), "stateRoot": fmt.Sprintf("0x%064x", height+1000), "parentHash": fmt.Sprintf("0x%064x", height), "timestamp": "0x1", "transactions": []any{}}
			}
		default:
			http.Error(w, "unexpected method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func publicFaultCase(t *testing.T, file string) dsl.Spec {
	t.Helper()
	raw, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/fault/" + file}, "../../presets/chain/stablenet-bp4.json")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := dsl.ParseV2(raw[0])
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

func publicFaultRecoveryCheckpoint(t *testing.T, file string) dsl.Spec {
	t.Helper()
	spec := publicFaultCase(t, file)
	start := -1
	for i, statement := range spec.Sequence {
		if statement.Do == "restartNode" {
			start = i + 1
		}
	}
	if start < 0 {
		t.Fatal("public case has no restart boundary")
	}
	checkpoint := []dsl.Statement{}
	for _, statement := range spec.Sequence[start:] {
		// This is a checkpoint unit test. Log transport is covered separately.
		if statement.Do != "readNodeLog" {
			checkpoint = append(checkpoint, statement)
		}
	}
	if len(checkpoint) == 0 {
		t.Fatal("public recovery checkpoint is empty")
	}
	spec.Sequence = checkpoint
	return spec
}

type faultProgressRecorder struct {
	recordStub
	steps []session.StepResult
}

func (r *faultProgressRecorder) Step(_ int, result session.StepResult) {
	r.steps = append(r.steps, result)
}

func TestPublicFaultRequiresRelativeProgressWhileNodeStopped(t *testing.T) {
	for _, file := range []string{"CT-FAULT-001-producer-crash-and-restart.json", "CT-FAULT-002-node-recover-and-sync.json"} {
		t.Run(file, func(t *testing.T) {
			spec := publicFaultCase(t, file)
			sequence := []dsl.Statement{}
			for _, statement := range spec.Sequence {
				if statement.Do == "restartNode" {
					break
				}
				if statement.Do == "stopNode" {
					continue
				}
				args := make(map[string]any, len(statement.Args))
				for key, value := range statement.Args {
					args[key] = value
				}
				statement.Args = args
				if statement.Do == "waitBlock" || statement.Expect == "blockAdvance" {
					// Only timing is accelerated: actual public targets and
					// comparison/progress semantics remain unchanged.
					statement.Args["timeout"] = "60ms"
					statement.Args["pollInterval"] = "1ms"
				}
				sequence = append(sequence, statement)
			}
			spec.Sequence = sequence
			primary := faultCheckpointNode(t, 100, 100, 100, 100, 101)
			urls := []string{primary, faultCheckpointNode(t, 100), faultCheckpointNode(t, 100), faultCheckpointNode(t, 100)}
			recorder := &faultProgressRecorder{}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			status, err := interp.NewInterpreter(deps()).Run(ctx, spec, envOf(t, urls...), recorder)
			if err != nil || status != session.StatusFail {
				t.Fatalf("only0 or1 new block from initial100 must not meet required10/20-block progress: status=%s err=%v steps=%+v", status, err, recorder.steps)
			}
			if len(recorder.steps) == 0 || !strings.Contains(recorder.steps[len(recorder.steps)-1].Error, "not reached") {
				t.Fatalf("failure must be the actual unmet block target, not an unrelated error: %+v", recorder.steps)
			}
		})
	}
}

func TestPublicFaultRecoveryRetainsLegacyCatchup(t *testing.T) {
	fixtures := []struct {
		name, file string
		heads      [4]uint64
		want       session.TestStatus
	}{
		{"crash-equal", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{20, 20, 20, 20}, session.StatusPass},
		{"crash-two-block-boundary", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{20, 18, 20, 19}, session.StatusPass},
		{"crash-recovered-two-block-boundary", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{20, 20, 18, 20}, session.StatusPass},
		{"crash-lagging-primary", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{17, 20, 20, 20}, session.StatusFail},
		{"crash-lagging-node2", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{20, 17, 20, 20}, session.StatusFail},
		{"crash-lagging-node4", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{20, 20, 20, 17}, session.StatusFail},
		{"crash-non-anchor-extremes", "CT-FAULT-001-producer-crash-and-restart.json", [4]uint64{20, 19, 22, 20}, session.StatusFail},
		{"recover-equal", "CT-FAULT-002-node-recover-and-sync.json", [4]uint64{20, 20, 20, 20}, session.StatusPass},
		{"recover-two-block-boundary", "CT-FAULT-002-node-recover-and-sync.json", [4]uint64{20, 20, 18, 20}, session.StatusPass},
		{"recover-lagging-node3", "CT-FAULT-002-node-recover-and-sync.json", [4]uint64{20, 20, 17, 20}, session.StatusFail},
		{"recover-node3-ahead", "CT-FAULT-002-node-recover-and-sync.json", [4]uint64{20, 20, 23, 20}, session.StatusPass},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			spec := publicFaultRecoveryCheckpoint(t, fixture.file)
			urls := []string{}
			for _, height := range fixture.heads {
				urls = append(urls, faultCheckpointNode(t, height))
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			recorder := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(ctx, spec, envOf(t, urls...), recorder)
			if err != nil || got != fixture.want {
				t.Fatalf("actual public recovery checkpoint status=%s err=%v want=%s heads=%v assertions=%+v", got, err, fixture.want, fixture.heads, recorder.asserts)
			}
		})
	}
}
