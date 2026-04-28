// Package remote is a thin wrapper around ethclient. Exists as a seam for future auth transport
// injection (Sprint 3b.2b) and method grouping (tx.send etc. in Sprint 4).
// Handlers talk to remote.Client, not ethclient directly.
package remote

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	neturl "net/url"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

// NonceAt returns the account nonce at the given block (eth_getTransactionCount).
// Pass nil for the latest block. Distinct from PendingNonceAt: this reads the
// historical / latest mined nonce rather than including pending-pool entries,
// which is the correct semantics for state inspection (node.account_state).
func (c *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	n, err := c.rpc.NonceAt(ctx, account, blockNumber)
	if err != nil {
		return 0, fmt.Errorf("remote.NonceAt: %w", err)
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

// SendRawTransaction broadcasts pre-encoded RLP bytes via eth_sendRawTransaction.
// Used by chain-specific tx types whose envelopes ethclient.SendTransaction does
// not understand (e.g. go-stablenet FeeDelegateDynamicFeeTx 0x16). The caller
// is responsible for constructing + signing the bytes; this wrapper only
// formats the hex payload and forwards.
//
// The returned tx hash from the endpoint is intentionally discarded — the
// caller already has the canonical hash via keccak256(raw) and we don't want
// to rely on the endpoint's echo for correctness.
func (c *Client) SendRawTransaction(ctx context.Context, raw []byte) error {
	var result string
	if err := c.rpc.Client().CallContext(ctx, &result, "eth_sendRawTransaction", hexutil.Encode(raw)); err != nil {
		return fmt.Errorf("remote.SendRawTransaction: %w", err)
	}
	_ = result
	return nil
}

// TransactionReceipt fetches the receipt for a tx hash. Returns
// ethereum.NotFound (verbatim) when the endpoint reports a null result,
// so callers can distinguish "still pending" from a real RPC failure
// without string-matching error messages.
func (c *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	rcpt, err := c.rpc.TransactionReceipt(ctx, hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, ethereum.NotFound
		}
		return nil, fmt.Errorf("remote.TransactionReceipt: %w", err)
	}
	return rcpt, nil
}

// CallContract executes a read-only contract call (eth_call) at the given
// block and returns the raw return bytes. Pass nil for the latest block.
// Used by Sprint 4d's contract_call / contract read paths.
func (c *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	out, err := c.rpc.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("remote.CallContract: %w", err)
	}
	return out, nil
}

// FilterLogs queries event logs (eth_getLogs) matching the given filter.
// Used by Sprint 4d's events_get path.
func (c *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	logs, err := c.rpc.FilterLogs(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("remote.FilterLogs: %w", err)
	}
	return logs, nil
}

// CodeAt fetches the deployed bytecode of an account (eth_getCode) at the
// given block. Returns an empty slice for EOAs / undeployed addresses.
func (c *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	code, err := c.rpc.CodeAt(ctx, account, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("remote.CodeAt: %w", err)
	}
	return code, nil
}

// StorageAt fetches a single 32-byte storage slot for account / key
// (eth_getStorageAt). Pass nil block for latest.
func (c *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	val, err := c.rpc.StorageAt(ctx, account, key, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("remote.StorageAt: %w", err)
	}
	return val, nil
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
