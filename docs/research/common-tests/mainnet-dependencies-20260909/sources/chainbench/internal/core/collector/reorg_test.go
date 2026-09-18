package collector_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func waitForked(t *testing.T, c collector.Collector, want bool) collector.Chainstate {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var snap collector.Chainstate
	for time.Now().Before(deadline) {
		snap = c.Snapshot()
		if len(snap.Heights) > 0 && snap.Forked == want {
			return snap
		}
		time.Sleep(5 * time.Millisecond)
	}
	return snap
}

// TestCollector_ForkAcrossNodes flags a fork when two nodes report divergent
// hashes at the same height.
func TestCollector_ForkAcrossNodes(t *testing.T) {
	env := envWithNodes(t,
		node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"},
		node.Node{Index: 2, Role: node.RoleValidator, RPCURL: "http://n2"},
	)
	states := map[string]collector.Sample{
		"http://n1": {Height: 5, HeadHash: "0xaaaa", HeadMiner: "0xA"},
		"http://n2": {Height: 5, HeadHash: "0xbbbb", HeadMiner: "0xA"},
	}
	c := collector.New(collector.Deps{
		Interval: 5 * time.Millisecond,
		Probe: func(_ context.Context, url string) (collector.Sample, error) {
			return states[url], nil
		},
	})
	_ = c.Start(context.Background(), env)
	defer c.Stop()

	if snap := waitForked(t, c, true); !snap.Forked {
		t.Fatalf("expected Forked=true for divergent hashes, got %+v", snap)
	}
}

// TestCollector_NoForkWhenConsistent keeps Forked false while nodes agree.
func TestCollector_NoForkWhenConsistent(t *testing.T) {
	env := envWithNodes(t,
		node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"},
		node.Node{Index: 2, Role: node.RoleValidator, RPCURL: "http://n2"},
	)
	states := map[string]collector.Sample{
		"http://n1": {Height: 5, HeadHash: "0xaaaa", HeadMiner: "0xA"},
		"http://n2": {Height: 5, HeadHash: "0xaaaa", HeadMiner: "0xA"},
	}
	c := collector.New(collector.Deps{
		Interval: 5 * time.Millisecond,
		Probe: func(_ context.Context, url string) (collector.Sample, error) {
			return states[url], nil
		},
	})
	_ = c.Start(context.Background(), env)
	defer c.Stop()

	// Let several samples run, then confirm no fork was flagged.
	time.Sleep(80 * time.Millisecond)
	if snap := c.Snapshot(); snap.Forked {
		t.Fatalf("expected Forked=false for consistent hashes, got %+v", snap)
	}
}

// TestCollector_ReorgOnSingleNode flags a fork when a node's hash at a known
// height changes across samples (a reorg).
func TestCollector_ReorgOnSingleNode(t *testing.T) {
	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})

	var mu sync.Mutex
	var calls int
	c := collector.New(collector.Deps{
		Interval: 3 * time.Millisecond,
		Probe: func(context.Context, string) (collector.Sample, error) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			hash := "0xaaaa"
			if calls > 3 { // reorg: same height, different hash
				hash = "0xbbbb"
			}
			return collector.Sample{Height: 5, HeadHash: hash, HeadMiner: "0xA"}, nil
		},
	})
	_ = c.Start(context.Background(), env)
	defer c.Stop()

	if snap := waitForked(t, c, true); !snap.Forked {
		t.Fatalf("expected Forked=true after a reorg, got %+v", snap)
	}
}
