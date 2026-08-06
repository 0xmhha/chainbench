package collector_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// envWithNodes builds a real session Environment holding the given nodes.
func envWithNodes(t *testing.T, nodes ...node.Node) session.Environment {
	t.Helper()
	s, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC(), nil)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := s.NewEnvironment("ffffffffffff0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
	return env
}

func TestCollector_SnapshotFromProbe(t *testing.T) {
	env := envWithNodes(t,
		node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"},
		node.Node{Index: 2, Role: node.RoleValidator, RPCURL: "http://n2"},
	)
	states := map[string]collector.NodeState{
		"http://n1": {Height: 100, Peers: 3},
		"http://n2": {Height: 99, Peers: 3},
	}
	c := collector.New(collector.Deps{
		Interval: 10 * time.Millisecond,
		Probe: func(_ context.Context, rpcURL string) (collector.NodeState, error) {
			return states[rpcURL], nil
		},
	})
	if err := c.Start(context.Background(), env); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	// Wait for at least one sample.
	deadline := time.Now().Add(2 * time.Second)
	var snap collector.Chainstate
	for time.Now().Before(deadline) {
		snap = c.Snapshot()
		if len(snap.Heights) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if snap.Heights["node1"] != 100 || snap.Heights["node2"] != 99 {
		t.Fatalf("heights = %v", snap.Heights)
	}
	if snap.Peers["node1"] != 3 {
		t.Fatalf("peers = %v", snap.Peers)
	}
}

func TestCollector_ProbeErrorIsSkipped(t *testing.T) {
	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://down"})
	c := collector.New(collector.Deps{
		Interval: 10 * time.Millisecond,
		Probe: func(_ context.Context, _ string) (collector.NodeState, error) {
			return collector.NodeState{}, context.DeadlineExceeded
		},
	})
	_ = c.Start(context.Background(), env)
	defer c.Stop()
	time.Sleep(50 * time.Millisecond)
	// A failing probe must not appear and must not block.
	if _, ok := c.Snapshot().Heights["node1"]; ok {
		t.Fatal("failed probe should not populate height")
	}
}

func TestCollector_WaitLog(t *testing.T) {
	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})
	// Write the node's log with a matching line.
	if err := os.WriteFile(env.LogPath("node1"), []byte("boot\nblock reward 100 paid\ndone\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := collector.New(collector.Deps{Interval: time.Second, Probe: func(context.Context, string) (collector.NodeState, error) { return collector.NodeState{}, nil }})
	_ = c.Start(context.Background(), env)
	defer c.Stop()

	m, err := c.WaitLog(context.Background(), "node1", `block reward \d+`, time.Second)
	if err != nil {
		t.Fatalf("WaitLog: %v", err)
	}
	if m.Lines[0] != 2 {
		t.Fatalf("match line = %d, want 2", m.Lines[0])
	}
	// Absent pattern times out.
	if _, err := c.WaitLog(context.Background(), "node1", "nonexistent", 100*time.Millisecond); err == nil {
		t.Fatal("absent pattern must time out")
	}
}
