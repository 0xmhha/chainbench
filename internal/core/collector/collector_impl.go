package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// defaultInterval is the chainstate sampling period when Deps.Interval is unset.
const defaultInterval = time.Second

// NodeState is one node's RPC-sampled state.
type NodeState struct {
	Height uint64
	Peers  int
}

// Deps injects the collector's collaborators so it is testable without live
// nodes. A production wiring supplies an RPC-backed probe.
type Deps struct {
	// Probe samples one node's state by RPC. Errors are treated as "not yet
	// reachable" and skipped, never blocking the node.
	Probe func(ctx context.Context, rpcURL string) (NodeState, error)
	// Interval is the sampling period (defaults to one second).
	Interval time.Duration
}

// collector samples chainstate over RPC and locates log lines on demand. This
// is the RPC-minimal build; live log tailing plus bp-participation and reorg
// detection land in the full collector (T3.3).
type collector struct {
	deps     Deps
	interval time.Duration

	env  session.Environment
	stop chan struct{}
	done chan struct{}

	mu    sync.Mutex
	state Chainstate
}

// New returns an RPC-sampling collector.
func New(deps Deps) Collector {
	interval := deps.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	return &collector{deps: deps, interval: interval}
}

// Start samples the environment's nodes immediately and then every interval,
// until Stop or ctx cancellation.
func (c *collector) Start(ctx context.Context, env session.Environment) error {
	c.env = env
	c.stop = make(chan struct{})
	c.done = make(chan struct{})
	go c.run(ctx)
	return nil
}

func (c *collector) run(ctx context.Context) {
	defer close(c.done)
	t := time.NewTicker(c.interval)
	defer t.Stop()

	c.sampleOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stop:
			return
		case <-t.C:
			c.sampleOnce(ctx)
		}
	}
}

// sampleOnce probes every node and replaces the snapshot. A failing probe is
// skipped so a slow or down node never blocks or corrupts the snapshot.
func (c *collector) sampleOnce(ctx context.Context) {
	heights := map[string]uint64{}
	peers := map[string]int{}
	for _, n := range c.env.Nodes() {
		st, err := c.deps.Probe(ctx, n.RPCURL)
		if err != nil {
			continue
		}
		name := "node" + strconv.Itoa(n.Index)
		heights[name] = st.Height
		peers[name] = st.Peers
	}
	c.mu.Lock()
	c.state = Chainstate{Heights: heights, Peers: peers}
	c.mu.Unlock()
}

// Snapshot returns the latest sampled chainstate.
func (c *collector) Snapshot() Chainstate {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Stop ends the sampler and waits for it to exit.
func (c *collector) Stop() error {
	if c.stop == nil {
		return nil
	}
	close(c.stop)
	<-c.done
	c.stop = nil
	return nil
}

// WaitLog polls the node's log file until pattern matches or timeout elapses.
// This is a poll-scan; the full collector upgrades it to an append-only tail.
func (c *collector) WaitLog(ctx context.Context, nodeName, pattern string, timeout time.Duration) (LogMatch, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return LogMatch{}, fmt.Errorf("collector: invalid pattern %q: %w", pattern, err)
	}
	path := c.env.LogPath(nodeName)
	deadline := time.Now().Add(timeout)
	for {
		if m, ok := scanLog(path, re); ok {
			return m, nil
		}
		if time.Now().After(deadline) {
			return LogMatch{}, fmt.Errorf("collector: pattern %q not found in %s within %s", pattern, path, timeout)
		}
		select {
		case <-ctx.Done():
			return LogMatch{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// scanLog returns the first line in path matching re, with its 1-based line
// number and byte offset.
func scanLog(path string, re *regexp.Regexp) (LogMatch, bool) {
	f, err := os.Open(path)
	if err != nil {
		return LogMatch{}, false
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	var offset int64
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if re.MatchString(text) {
			return LogMatch{File: path, Lines: [2]int{line, line}, ByteOffset: offset, Text: text}, true
		}
		offset += int64(len(text)) + 1 // + newline
	}
	return LogMatch{}, false
}
