// Package remote is a thin wrapper around ethclient. Exists as a seam for future auth transport
// injection (Sprint 3b.2b) and method grouping (tx.send etc. in Sprint 4).
// Handlers talk to remote.Client, not ethclient directly.
package remote

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client is a read-only Ethereum RPC client backed by go-ethereum's ethclient.
// Construct via Dial and Close when done.
type Client struct {
	rpc *ethclient.Client
}

// DialOptions configures optional behavior for DialWithOptions. A zero value
// is equivalent to the bare Dial path (no custom transport).
type DialOptions struct {
	// Transport, when non-nil, replaces the default HTTP transport used by
	// the underlying JSON-RPC client. The typical use case is
	// auth-injection: APIKeyTransport / BearerTokenTransport returned by
	// AuthFromNode. It has no effect for non-HTTP schemes (ws, ipc) — those
	// fall through to ethclient.DialContext regardless.
	Transport http.RoundTripper
}

// Dial opens a JSON-RPC client against the given URL. Accepts any scheme
// ethclient supports (http, https, ws, wss, or an IPC path); Sprint 3b.2a
// only exercises http(s). Equivalent to DialWithOptions(ctx, url, DialOptions{}).
func Dial(ctx context.Context, url string) (*Client, error) {
	return DialWithOptions(ctx, url, DialOptions{})
}

// DialWithOptions opens a Client with optional transport injection. Passing a
// zero DialOptions is behaviorally identical to Dial. When opts.Transport is
// set, the request goes through rpc.DialOptions + rpc.WithHTTPClient so the
// injected RoundTripper sits in front of every outbound request (used for
// API-key / JWT auth).
//
// go-ethereum v1.17.2 rpc API reference:
//   - rpc.DialOptions(ctx, url, ...ClientOption) (*rpc.Client, error)
//   - rpc.WithHTTPClient(*http.Client) ClientOption
//   - ethclient.NewClient(*rpc.Client) *ethclient.Client
func DialWithOptions(ctx context.Context, url string, opts DialOptions) (*Client, error) {
	if opts.Transport == nil {
		// No transport override — take the simpler ethclient.DialContext path
		// so ws / wss / ipc schemes continue to work unchanged.
		rpcClient, err := ethclient.DialContext(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("remote.Dial %q: %w", url, err)
		}
		return &Client{rpc: rpcClient}, nil
	}
	// Custom transport: build an *http.Client and wire it via the rpc
	// package's ClientOption hook. Only meaningful for http(s) URLs; ws/ipc
	// dials ignore WithHTTPClient inside go-ethereum.
	httpClient := &http.Client{Transport: opts.Transport}
	rpcClient, err := rpc.DialOptions(ctx, url, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("remote.DialWithOptions %q: %w", url, err)
	}
	return &Client{rpc: ethclient.NewClient(rpcClient)}, nil
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
