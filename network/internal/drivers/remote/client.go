// Package remote is a thin wrapper around ethclient. Exists as a seam for future auth transport
// injection (Sprint 3b.2b) and method grouping (tx.send etc. in Sprint 4).
// Handlers talk to remote.Client, not ethclient directly.
package remote

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// Client is a read-only Ethereum RPC client backed by go-ethereum's ethclient.
// Construct via Dial and Close when done.
type Client struct {
	rpc *ethclient.Client
}

// Dial opens a JSON-RPC client against the given URL. Accepts any scheme
// ethclient supports (http, https, ws, wss, or an IPC path); Sprint 3b.2a
// only exercises http(s). Auth transport plumbing will arrive in 3b.2b.
func Dial(ctx context.Context, url string) (*Client, error) {
	rpc, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("remote.Dial %q: %w", url, err)
	}
	return &Client{rpc: rpc}, nil
}

// BlockNumber returns the current head block number as reported by the endpoint.
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	bn, err := c.rpc.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("remote.BlockNumber: %w", err)
	}
	return bn, nil
}

// Close releases the underlying HTTP/RPC connection. Nil-safe on the
// remote.Client receiver; ethclient's own Close is not nil-safe so the
// receiver check is load-bearing. BlockNumber and future methods assume
// construction via Dial and do not replicate the guard.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.rpc.Close()
}
