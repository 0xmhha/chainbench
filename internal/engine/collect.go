package engine

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"
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

// withCollection wraps build so that, when bus is non-nil, a live collector runs
// for each built environment — mirroring its chainstate and logs to bus — and is
// torn down as part of the environment teardown. When bus is nil it returns
// build unchanged, so collection is opt-in and never affects a run without a
// dashboard. dial nil uses the default RPC dialer.
func withCollection(build BuildEnvFunc, bus *obs.Bus, dial func(string) *rpc.Client) BuildEnvFunc {
	if bus == nil {
		return build
	}
	probe := rpcProbe(dial)
	return func(ctx context.Context, env session.Environment, spec testspec.Spec) (node.NodeSet, TeardownFunc, error) {
		ns, td, err := build(ctx, env, spec)
		if err != nil {
			return ns, td, err
		}
		// Populate the node table before starting collection so the collector
		// samples the built nodes; engine.Run re-populates it (idempotently)
		// after BuildEnv returns. Without this the collector could sample an
		// empty table and report no chainstate.
		env.PopulateNodeTable(ns)
		stop := startCollection(ctx, env, bus, probe, chainstateInterval)
		return ns, func(c context.Context) error {
			cerr := stop()
			if td != nil {
				if terr := td(c); terr != nil {
					return terr
				}
			}
			return cerr
		}, nil
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
