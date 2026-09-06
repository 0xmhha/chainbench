package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// The attached-network registry: networks a caller has attached to by name, so
// a later call can say "that one" instead of repeating its endpoints.

type (
	// Node is one node of a network as a surface receives it.
	Node = node.Node
	// DetectOptions selects what to detect about a running network.
	DetectOptions = collector.Options
	// DetectResult is what the detection found.
	DetectResult = collector.Result
)

// RoleEndpoint is the role of a node that serves RPC without producing blocks.
const RoleEndpoint = node.RoleEndpoint

// IsValidNetworkName reports whether a name may be registered. The rule is the
// registry's, so every surface applies the same one.
func IsValidNetworkName(name string) bool { return session.IsValidNetworkName(name) }

// AttachNetwork records a running network under its name so later calls can
// name it instead of repeating its endpoints.
func AttachNetwork(_ Deps, stateDir string, ns NodeSet) error {
	return session.SaveNetwork(stateDir, ns)
}

// Network reads a network the caller attached earlier.
func Network(_ Deps, stateDir, name string) (NodeSet, error) {
	return session.LoadNetwork(stateDir, name)
}

// Networks lists every attached network.
func Networks(_ Deps, stateDir string) ([]NodeSet, error) {
	return session.ListNetworks(stateDir)
}

// DetachNetwork forgets an attached network. It stops nothing: the nodes are
// not this bench's to stop, which is the point of having attached rather than
// composed.
func DetachNetwork(_ Deps, stateDir, name string) error {
	return session.RemoveNetwork(stateDir, name)
}

// DetectNetwork asks a running network what it is, so an attach can record the
// chain and roles rather than making the caller declare them.
func DetectNetwork(ctx context.Context, _ Deps, in DetectOptions) (*DetectResult, error) {
	return collector.Detect(ctx, in)
}

// PeersOf reports how many peers a node has, which is the cheapest question
// that distinguishes a wired network from a set of isolated nodes.
func PeersOf(ctx context.Context, _ Deps, rpcURL string) (uint64, error) {
	if rpcURL == "" {
		return 0, fmt.Errorf("a node endpoint is required")
	}
	return rpc.Dial(rpcURL).PeerCount(ctx)
}

// Reaching a node of a saved network is not the same as dialling a URL: an
// attached node may sit behind an SSH tunnel or a docker mapping, and the way
// through is the auth descriptor recorded with it. These take the node rather
// than an endpoint so that the way through comes with it.

// CallOnNode makes a JSON-RPC call to one node of a saved network, through
// whatever the node's record says is needed to reach it.
func CallOnNode(ctx context.Context, _ Deps, n Node, method string, params ...any) ([]byte, error) {
	if method == "" {
		return nil, fmt.Errorf("a method is required")
	}
	c, err := clientFor(n)
	if err != nil {
		return nil, err
	}
	var out any
	if err := c.Call(ctx, method, &out, params...); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

// PeersOfNode reports one node's peer count, reached the same way.
func PeersOfNode(ctx context.Context, _ Deps, n Node) (uint64, error) {
	c, err := clientFor(n)
	if err != nil {
		return 0, err
	}
	return c.PeerCount(ctx)
}

// NodeAt returns the node with this 1-based index, or the network's primary
// when no index is named.
func NodeAt(ns NodeSet, index int) (Node, bool) {
	if index > 0 {
		for _, n := range ns.Nodes {
			if n.Index == index {
				return n, true
			}
		}
		return Node{}, false
	}
	return ns.Primary()
}

// clientFor dials a node through its stored auth descriptor.
func clientFor(n Node) (*rpc.Client, error) {
	hc, err := remote.HTTPClientFromAuth(remote.Auth(n.Auth), os.Getenv)
	if err != nil {
		return nil, err
	}
	return rpc.DialWithClient(n.RPCURL, hc), nil
}
