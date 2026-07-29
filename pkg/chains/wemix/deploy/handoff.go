package deploy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/remote"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

// handoffConfirmed reports whether a post-fork block's miner proves the handoff:
// it must be non-empty and NOT one of the wemix producers (the pre-fork
// producers cannot seal post-Croissant blocks, so a different sealer means a
// go-wbft validator took over).
func handoffConfirmed(miner string, producers map[string]bool) bool {
	m := strings.ToLower(strings.TrimSpace(miner))
	if m == "" || m == "0x0000000000000000000000000000000000000000" {
		return false
	}
	return !producers[m]
}

// lowerSet returns the lowercased set of the addresses.
func lowerSet(addrs []string) map[string]bool {
	s := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		if a != "" {
			s[strings.ToLower(a)] = true
		}
	}
	return s
}

// WaitHandoff polls a go-wbft validator's RPC (reached over an SSH tunnel, since
// the cluster is a closed network) until the chain crosses the Croissant block
// AND the block after it was sealed by a validator rather than a wemix producer.
// It returns the confirming block's miner. producerAddrs are the pre-fork wemix
// producer coinbases to exclude.
func WaitHandoff(ctx context.Context, c *Cluster, cr *Credentials, hostKey remote.HostKeyCallback, producerAddrs []string, timeout time.Duration, env func(string) string) (string, error) {
	vals := c.Validators()
	if len(vals) == 0 {
		return "", fmt.Errorf("deploy: cluster has no wbft_bp validator to poll")
	}
	rc, err := cr.For(c, vals[0], env)
	if err != nil {
		return "", err
	}
	tc, closer, err := remote.DialTunnelClient(rc, hostKey)
	if err != nil {
		return "", fmt.Errorf("deploy: ssh tunnel to validator: %w", err)
	}
	defer func() { _ = closer.Close() }()

	// Over the tunnel, reach the validator's own local RPC.
	cli := rpc.DialWithClient(fmt.Sprintf("http://127.0.0.1:%d", c.RPCPort), tc)
	producers := lowerSet(producerAddrs)
	croissant := c.CroissantBlock
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		head, err := cli.BlockNumber(ctx)
		if err == nil && head > uint64(croissant) {
			var blk struct {
				Miner string `json:"miner"`
			}
			_ = cli.Call(ctx, "eth_getBlockByNumber", &blk, fmt.Sprintf("0x%x", croissant+1), false)
			if handoffConfirmed(blk.Miner, producers) {
				return strings.ToLower(blk.Miner), nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("deploy: handoff not observed within %s (croissant block %d)", timeout, croissant)
}
