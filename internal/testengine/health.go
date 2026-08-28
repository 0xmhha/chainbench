package testengine

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// healthPollInterval is how often the block-advance gate polls the head.
const healthPollInterval = 2 * time.Second

// NewBlockAdvanceGate returns a launcher health gate that passes once the
// primary node's head reaches target, polling until timeout. It is the local
// (non-etcd) liveness check: a network that produces blocks is up. A gate
// failure is classified as RPCUnready with the reason in Detail.
func NewBlockAdvanceGate(target uint64, timeout time.Duration) func(context.Context, node.NodeSet) (launcher.Diagnosis, error) {
	return func(ctx context.Context, ns node.NodeSet) (launcher.Diagnosis, error) {
		if len(ns.Nodes) == 0 {
			return launcher.Diagnosis{Mode: launcher.RPCUnready, Detail: "no nodes"},
				fmt.Errorf("engine: health gate: no nodes")
		}
		c := rpc.Dial(ns.Nodes[0].RPCURL)
		deadline := time.NewTimer(timeout)
		defer deadline.Stop()
		tick := time.NewTicker(healthPollInterval)
		defer tick.Stop()
		for {
			if h, err := c.BlockNumber(ctx); err == nil && h >= target {
				return launcher.Diagnosis{OK: true}, nil
			}
			select {
			case <-ctx.Done():
				return launcher.Diagnosis{Mode: launcher.RPCUnready, Detail: ctx.Err().Error()}, ctx.Err()
			case <-deadline.C:
				detail := fmt.Sprintf("head did not reach %d within %s", target, timeout)
				return launcher.Diagnosis{Mode: launcher.RPCUnready, Detail: detail},
					fmt.Errorf("engine: health gate: %s", detail)
			case <-tick.C:
			}
		}
	}
}
