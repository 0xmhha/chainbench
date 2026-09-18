package collector_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestCollector_BPParticipation drives the sampler through a sequence of head
// blocks with known producers and asserts the tally counts blocks per producer.
func TestCollector_BPParticipation(t *testing.T) {
	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})

	// Heights 1..4 produced by A, B, A, A.
	miners := []string{"0xA", "0xB", "0xA", "0xA"}
	var mu sync.Mutex
	var i int
	c := collector.New(collector.Deps{
		Interval: 5 * time.Millisecond,
		Probe: func(context.Context, string) (collector.Sample, error) {
			mu.Lock()
			defer mu.Unlock()
			idx := i
			if idx >= len(miners) {
				idx = len(miners) - 1 // steady state: keep reporting the last head
			} else {
				i++
			}
			return collector.Sample{Height: uint64(idx + 1), HeadMiner: miners[idx], HeadHash: "h"}, nil
		},
	})
	if err := c.Start(context.Background(), env); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	// Wait until all four heights have been sampled.
	deadline := time.Now().Add(2 * time.Second)
	var snap collector.Chainstate
	for time.Now().Before(deadline) {
		snap = c.Snapshot()
		if snap.Heights["node1"] == 4 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if snap.Heights["node1"] != 4 {
		t.Fatalf("did not reach height 4: %v", snap.Heights)
	}
	// Give one more sample so the tally over heights 1..4 is complete.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap = c.Snapshot()
		if snap.BPParticipation["0xA"] == 3 && snap.BPParticipation["0xB"] == 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("bp participation = %v, want {0xA:3, 0xB:1}", snap.BPParticipation)
}

// TestCollector_BPWindowPrunesOldHeights checks the tally only spans the recent
// window, so a long-running producer set does not grow unbounded.
func TestCollector_BPWindowPrunesOldHeights(t *testing.T) {
	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})

	var mu sync.Mutex
	var h uint64
	c := collector.New(collector.Deps{
		Interval: 2 * time.Millisecond,
		BPWindow: 3,
		Probe: func(context.Context, string) (collector.Sample, error) {
			mu.Lock()
			defer mu.Unlock()
			h++
			// Every block by the same producer.
			return collector.Sample{Height: h, HeadMiner: "0xA", HeadHash: "h"}, nil
		},
	})
	_ = c.Start(context.Background(), env)
	defer c.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap := c.Snapshot()
		// With a window of 3, the tally never exceeds 3 despite many blocks.
		if snap.Heights["node1"] >= 10 {
			if got := snap.BPParticipation["0xA"]; got > 3 {
				t.Fatalf("bp tally = %d, want <= window 3", got)
			}
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("did not reach height 10")
}
