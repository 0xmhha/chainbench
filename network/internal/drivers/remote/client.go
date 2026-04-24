// Package remote is a thin wrapper around ethclient. Exists as a seam for future auth transport
// injection (Sprint 3b.2b) and method grouping (tx.send etc. in Sprint 4).
// Handlers talk to remote.Client, not ethclient directly.
package remote

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	neturl "net/url"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	// the underlying JSON-RPC client. The typical use case is auth injection
	// via APIKeyTransport / BearerTokenTransport returned by AuthFromNode.
	//
	// Only compatible with http(s) URLs. DialWithOptions rejects ws/wss/ipc
	// URLs when a Transport is provided, because go-ethereum's WithHTTPClient
	// is silently ignored by non-HTTP dials and the auth header would never
	// reach the server.
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
	// Custom transport only makes sense for http(s). go-ethereum's
	// WithHTTPClient is silently ignored by ws / wss / ipc dials, which would
	// drop auth headers on the floor without surfacing an error. Fail fast
	// instead so the operator learns the configuration is impossible rather
	// than debugging a 401 from a server that never saw the key.
	if parsed, perr := neturl.Parse(url); perr == nil {
		switch parsed.Scheme {
		case "http", "https":
			// OK — Transport is honored.
		case "":
			// Likely an IPC path; Transport would be ignored.
			return nil, fmt.Errorf("remote.DialWithOptions: auth transport requires http(s) URL, got %q", url)
		default:
			return nil, fmt.Errorf("remote.DialWithOptions: auth transport requires http(s) scheme, got %q", parsed.Scheme)
		}
	}
	// Custom transport: build an *http.Client and wire it via the rpc
	// package's ClientOption hook.
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

// ChainID returns the chain id reported by the endpoint.
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	id, err := c.rpc.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("remote.ChainID: %w", err)
	}
	return id, nil
}

// BalanceAt returns the balance of address at the given block. Pass nil for latest.
func (c *Client) BalanceAt(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error) {
	bal, err := c.rpc.BalanceAt(ctx, address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("remote.BalanceAt: %w", err)
	}
	return bal, nil
}

// GasPrice returns the current suggested gas price (eth_gasPrice).
func (c *Client) GasPrice(ctx context.Context) (*big.Int, error) {
	gp, err := c.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("remote.GasPrice: %w", err)
	}
	return gp, nil
}

// PendingNonceAt returns the next available nonce for account (pending + mined).
// Thin wrapper over ethclient.PendingNonceAt which issues eth_getTransactionCount
// with the "pending" block tag.
func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	n, err := c.rpc.PendingNonceAt(ctx, account)
	if err != nil {
		return 0, fmt.Errorf("remote.PendingNonceAt: %w", err)
	}
	return n, nil
}

// EstimateGas asks the endpoint for the gas required to execute msg.
func (c *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	g, err := c.rpc.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("remote.EstimateGas: %w", err)
	}
	return g, nil
}

// SendTransaction broadcasts a signed transaction. Caller constructs + signs
// tx outside this package; remote.Client only forwards the RLP via
// eth_sendRawTransaction.
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if err := c.rpc.SendTransaction(ctx, tx); err != nil {
		return fmt.Errorf("remote.SendTransaction: %w", err)
	}
	return nil
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
