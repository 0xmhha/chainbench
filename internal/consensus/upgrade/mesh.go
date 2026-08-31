package upgrade

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// WaitEndpointsReady polls each RPC endpoint until it answers (eth_blockNumber)
// or the deadline passes. It is what a handoff waits on before
// mesh wiring so admin_addPeer does not race the nodes' HTTP servers.
func WaitEndpointsReady(ctx context.Context, endpoints []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for _, ep := range endpoints {
		if ep == "" {
			continue
		}
		cli := rpc.Dial(ep)
		for {
			if _, err := cli.BlockNumber(ctx); err == nil {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("endpoint %s not ready within %s", ep, timeout)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil
}

// PeerCaller adds a peer to one node via its RPC endpoint. Abstracted so the
// mesh wiring can be tested without live nodes.
type PeerCaller interface {
	AddPeer(ctx context.Context, endpoint, enode string) error
}

// rpcPeerCaller is the default PeerCaller: it issues admin_addPeer over JSON-RPC.
type rpcPeerCaller struct{}

func (rpcPeerCaller) AddPeer(ctx context.Context, endpoint, enode string) error {
	var ok bool
	return rpc.Dial(endpoint).Call(ctx, "admin_addPeer", &ok, enode)
}

// DefaultPeerCaller returns the JSON-RPC admin_addPeer caller.
func DefaultPeerCaller() PeerCaller { return rpcPeerCaller{} }

// Enodes returns each plan node's enode (enode://<pubkey>@host:<p2p>), in plan
// order. It needs a pubkey per node (set on the plan); a node with no pubkey
// yields an empty entry, which WireMesh skips.
func (p Plan) Enodes(host string) []string {
	if host == "" {
		host = "127.0.0.1"
	}
	out := make([]string, len(p.Nodes))
	for i, n := range p.Nodes {
		if n.Pubkey == "" {
			continue
		}
		out[i] = node.Enode(n.Pubkey, host, n.Ports.P2P)
	}
	return out
}

// WireMesh connects every node to every other node by calling admin_addPeer on
// each node's RPC endpoint for all the other nodes' enodes. The go-wbft
// validators need a full mesh -- not just links to the producer -- to exchange
// WBFT consensus messages with each other and reach a 2f+1 quorum after the
// fork; without it they seed the validator set at the fork block but stall.
//
// endpoints[i] and enodes[i] describe the same node i. Empty enodes are skipped
// (a node advertises itself to others; it does not add itself). It attempts
// every pair and returns a combined error naming the pairs that failed, so one
// unreachable node does not abort the rest of the mesh.
func WireMesh(ctx context.Context, caller PeerCaller, endpoints, enodes []string) error {
	if len(endpoints) != len(enodes) {
		return fmt.Errorf("upgrade: %d endpoints but %d enodes", len(endpoints), len(enodes))
	}
	var failures []string
	for i, ep := range endpoints {
		if ep == "" {
			continue
		}
		for j, en := range enodes {
			if i == j || en == "" {
				continue
			}
			if err := caller.AddPeer(ctx, ep, en); err != nil {
				failures = append(failures, fmt.Sprintf("node%d->node%d: %v", i+1, j+1, err))
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("upgrade: mesh wiring had %d failure(s): %v", len(failures), failures)
	}
	return nil
}
