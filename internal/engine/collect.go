package engine

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// chainstateInterval is how often live collection samples nodes and publishes a
// chainstate snapshot to the dashboard.
const chainstateInterval = 2 * time.Second

// rpcProbe adapts an rpc.Client dialer into a collector probe: it reads a node's
// height, peers, and head block (hash + producer) over RPC. A failing height
// read means "not yet reachable" and is surfaced as an error the collector
// skips; peers and head are best-effort.
func rpcProbe(dial func(string) *rpc.Client) func(context.Context, string) (collector.NodeState, error) {
	if dial == nil {
		dial = rpc.Dial
	}
	return func(ctx context.Context, url string) (collector.NodeState, error) {
		c := dial(url)
		height, err := c.BlockNumber(ctx)
		if err != nil {
			return collector.NodeState{}, err
		}
		peers, _ := c.PeerCount(ctx)
		_, hash, miner, _ := c.HeadBlock(ctx)
		return collector.NodeState{Height: height, Peers: int(peers), HeadHash: hash, HeadMiner: miner}, nil
	}
}

// startCollection runs a live collector for env, mirroring each node's log lines
// and a periodic chainstate snapshot to bus so the dashboard shows the network
// live. It returns a stop function that ends collection after a final snapshot.
// bus must be non-nil.
func startCollection(ctx context.Context, env session.Environment, bus *obs.Bus, probe func(context.Context, string) (collector.NodeState, error), interval time.Duration) func() error {
	col := collector.New(collector.Deps{
		Probe:    probe,
		Interval: interval,
		OnLine: func(nodeName, line string) {
			bus.Publish(obs.Event{
				Phase:   obs.PhaseTest,
				Kind:    obs.KindInfo,
				Message: line,
				Fields:  map[string]any{"node": nodeName, "log": true},
			})
		},
	})
	_ = col.Start(ctx, env)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-t.C:
				publishChainstate(bus, col.Snapshot())
			}
		}
	}()

	return func() error {
		close(stop)
		<-done
		// Stop the sampler first so at least one sample has completed, then emit
		// a final snapshot that reflects it.
		err := col.Stop()
		publishChainstate(bus, col.Snapshot())
		return err
	}
}

// publishChainstate mirrors one cross-node chainstate snapshot to the bus.
func publishChainstate(bus *obs.Bus, cs collector.Chainstate) {
	bus.Publish(obs.Event{
		Phase:   obs.PhaseTest,
		Kind:    obs.KindProgress,
		Message: "chainstate",
		Fields: map[string]any{
			"heights":         cs.Heights,
			"peers":           cs.Peers,
			"bpParticipation": cs.BPParticipation,
			"forked":          cs.Forked,
		},
	})
}
