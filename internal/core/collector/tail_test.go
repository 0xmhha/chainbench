package collector_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestCollector_LiveTail follows a node log from the start and through live
// appends, emitting each complete line once and never a partial line.
func TestCollector_LiveTail(t *testing.T) {
	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})
	path := env.LogPath("node1")
	if err := os.WriteFile(path, []byte("l1\nl2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var got []string
	c := collector.New(collector.Deps{
		Interval: time.Hour, // keep the sampler idle; this test is about the tail
		Probe:    func(context.Context, string) (collector.Sample, error) { return collector.Sample{}, nil },
		OnLine:   func(_, line string) { mu.Lock(); got = append(got, line); mu.Unlock() },
	})
	if err := c.Start(context.Background(), env); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	waitLines := func(want int) []string {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			n := len(got)
			snap := append([]string(nil), got...)
			mu.Unlock()
			if n >= want {
				return snap
			}
			time.Sleep(10 * time.Millisecond)
		}
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}

	// Pre-existing lines are tailed from the start.
	if lines := waitLines(2); len(lines) < 2 || lines[0] != "l1" || lines[1] != "l2" {
		t.Fatalf("initial tail = %v, want [l1 l2]", lines)
	}

	// Appended complete lines arrive in order.
	appendFile(t, path, "l3\nl4\n")
	if lines := waitLines(4); len(lines) < 4 || lines[2] != "l3" || lines[3] != "l4" {
		t.Fatalf("after append = %v, want ...l3 l4", lines)
	}

	// A partial line is not emitted until its newline arrives.
	appendFile(t, path, "par")
	time.Sleep(120 * time.Millisecond)
	if lines := waitLines(0); len(lines) != 4 {
		t.Fatalf("partial line must not emit yet, got %v", lines)
	}
	appendFile(t, path, "tial\n")
	if lines := waitLines(5); len(lines) < 5 || lines[4] != "partial" {
		t.Fatalf("completed line = %v, want ...partial", lines)
	}
}

func appendFile(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}

// chunkReader serves canned chunks for successive offsets, standing in for a
// remote reader.
type chunkReader struct {
	mu     sync.Mutex
	chunks map[int64]string
}

func (r *chunkReader) ReadFrom(_ context.Context, _ string, offset int64) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return []byte(r.chunks[offset]), nil
}

// TestCollector_TailUsesTheInjectedLogReader proves the tail loop reads through
// a boundary rather than the filesystem, which is what lets a remote reader (SSH)
// substitute for the local one. The first chunk ends mid-line; the partial must
// be neither emitted nor lost.
func TestCollector_TailUsesTheInjectedLogReader(t *testing.T) {
	reader := &chunkReader{chunks: map[int64]string{
		0:  "first line\npart",
		11: "partial line\n",
	}}

	env := envWithNodes(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})
	var mu sync.Mutex
	var got []string
	c := collector.New(collector.Deps{
		Logs:   reader,
		OnLine: func(_, line string) { mu.Lock(); got = append(got, line); mu.Unlock() },
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Start(ctx, env); err != nil {
		t.Fatalf("Start: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = c.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(got) < 2 {
		t.Fatalf("lines = %v, want at least 2", got)
	}
	if got[0] != "first line" || got[1] != "partial line" {
		t.Fatalf("lines = %v", got)
	}
}
