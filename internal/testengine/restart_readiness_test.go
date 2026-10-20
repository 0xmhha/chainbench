package testengine_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestRestartIdentityWaitsForRPCReadiness(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		var q struct{ ID any }
		_ = json.NewDecoder(r.Body).Decode(&q)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": "original"})
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	got, err := waitRestartIdentity(ctx, rpc.Dial(srv.URL), "original")
	if err != nil || got != "original" || calls.Load() != 3 {
		t.Fatalf("startup readiness lost: %q %v calls=%d", got, err, calls.Load())
	}
}

func TestViewChangeUsesGenerationDelayRatherThanHeaderTimestamp(t *testing.T) {
	p := viewProof{Validators: []string{"v0", "v1", "v2", "v3"}, Before: viewHeader{Number: 10, Timestamp: 100, Miner: "v0", Hash: "10"}, Changed: viewHeader{Number: 11, Timestamp: 102, Miner: "v2", Hash: "11", ParentHash: "10", Round: 1}, Successor: viewHeader{Number: 12, Timestamp: 104, PrevRound: 1, Hash: "12", ParentHash: "11"}, StoppedAddress: "v1", RestartedAddress: "v1", RequestTimeout: 3, BlockPeriod: 2, OldPID: 123, NewPID: 456, TargetIndex: 2, RoundChangeNodes: 2, CommitLogDelayMillis: 3003}
	for n := uint64(15); n <= 20; n++ {
		p.AfterRestart = append(p.AfterRestart, viewHeader{Number: n, Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1)})
	}
	if err := verifyViewChange(p); err != nil {
		t.Fatal("actual generation delay confused with declared header time", err)
	}
	p.CommitLogDelayMillis = 2000
	if verifyViewChange(p) == nil {
		t.Fatal("missing request timeout delay accepted")
	}
}

func TestRestartReadinessRejectsWrongIdentityAndDeadline(t *testing.T) {
	for _, name := range []string{"wrong identity", "deadline"} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if name == "deadline" {
					http.Error(w, "starting", http.StatusServiceUnavailable)
					return
				}
				var q struct{ ID any }
				_ = json.NewDecoder(r.Body).Decode(&q)
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": "stranger"})
			}))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()
			_, err := waitRestartIdentity(ctx, rpc.Dial(srv.URL), "original")
			if err == nil {
				t.Fatal("invalid restarted endpoint accepted")
			}
			if name == "wrong identity" && calls.Load() != 1 {
				t.Fatal("wrong identity was retried")
			}
			if name == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("lost readiness deadline", err)
			}
		})
	}
}
