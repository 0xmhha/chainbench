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

const (
	// defaultInterval is the chainstate sampling period when Deps.Interval is unset.
	defaultInterval = time.Second
	// defaultBPWindow is how many recent block heights the bp-participation tally
	// spans when Deps.BPWindow is unset.
	defaultBPWindow = 100
)

// NodeState is one node's RPC-sampled state. HeadHash and HeadMiner describe the
// node's latest block; they drive bp-participation and fork detection and are
// optional (an empty HeadHash means the probe supplied height/peers only).
type NodeState struct {
	Height    uint64
	Peers     int
	HeadHash  string
	HeadMiner string
}

// Deps injects the collector's collaborators so it is testable without live
// nodes. A production wiring supplies an RPC-backed probe.
type Deps struct {
	// Probe samples one node's state by RPC. Errors are treated as "not yet
	// reachable" and skipped, never blocking the node.
	Probe func(ctx context.Context, rpcURL string) (NodeState, error)
	// Interval is the sampling period (defaults to one second).
	Interval time.Duration
	// BPWindow is the number of recent block heights the bp-participation tally
	// spans (defaults to defaultBPWindow). Older heights are pruned.
	BPWindow int
}

// collector samples chainstate over RPC and locates log lines on demand. It
// tracks recent (height, producer) pairs for bp-participation and remembers each
// height's block hash to detect forks/reorgs across samples.
type collector struct {
	deps     Deps
	interval time.Duration
	bpWindow int

	env  session.Environment
	stop chan struct{}
	done chan struct{}

	mu     sync.Mutex
	state  Chainstate
	blocks map[uint64]string // height -> producer (miner) for the bp-participation window
}

// New returns an RPC-sampling collector.
func New(deps Deps) Collector {
	interval := deps.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	bpWindow := deps.BPWindow
	if bpWindow <= 0 {
		bpWindow = defaultBPWindow
	}
	return &collector{deps: deps, interval: interval, bpWindow: bpWindow, blocks: map[uint64]string{}}
}

// nodeName is the collector's canonical name for a node ("node"+index).
func nodeName(index int) string { return "node" + strconv.Itoa(index) }

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
// skipped so a slow or down node never blocks or corrupts the snapshot. Recent
// (height, producer) pairs accumulate for the bp-participation tally.
func (c *collector) sampleOnce(ctx context.Context) {
	heights := map[string]uint64{}
	peers := map[string]int{}
	for _, n := range c.env.Nodes() {
		st, err := c.deps.Probe(ctx, n.RPCURL)
		if err != nil {
			continue
		}
		name := nodeName(n.Index)
		heights[name] = st.Height
		peers[name] = st.Peers
		c.recordBlock(st)
	}
	c.mu.Lock()
	c.state = Chainstate{
		Heights:         heights,
		Peers:           peers,
		BPParticipation: c.tallyBP(),
	}
	c.mu.Unlock()
}

// recordBlock remembers the producer of a probed head block and prunes heights
// older than the bp-participation window. Only the sampler goroutine calls it.
func (c *collector) recordBlock(st NodeState) {
	if st.HeadMiner == "" {
		return
	}
	c.blocks[st.Height] = st.HeadMiner

	var max uint64
	for h := range c.blocks {
		if h > max {
			max = h
		}
	}
	if max <= uint64(c.bpWindow) {
		return
	}
	cutoff := max - uint64(c.bpWindow) + 1
	for h := range c.blocks {
		if h < cutoff {
			delete(c.blocks, h)
		}
	}
}

// tallyBP counts blocks produced per producer over the current window. It
// returns nil when no producer has been observed.
func (c *collector) tallyBP() map[string]int {
	if len(c.blocks) == 0 {
		return nil
	}
	bp := map[string]int{}
	for _, miner := range c.blocks {
		bp[miner]++
	}
	return bp
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
