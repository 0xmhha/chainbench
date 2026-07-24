// Package rpc is a minimal JSON-RPC-over-HTTP client for the verify and test
// phases. It is chain-agnostic (only standard eth_/net_ methods) and separate
// from the accounts SDK transport, which carries stablenet-specific account
// queries. Consensus-namespace calls (istanbul_/wemix_) go through the generic
// Call with a method name from the chain manifest.
package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

// Client is a JSON-RPC over HTTP client.
type Client struct {
	url  string
	http *http.Client
	id   atomic.Uint64
}

// Dial returns a client for the given JSON-RPC HTTP endpoint using the default
// HTTP client.
func Dial(url string) *Client {
	return &Client{url: url, http: http.DefaultClient}
}

// DialWithClient returns a client using a caller-supplied *http.Client (for
// custom transports/timeouts, e.g. an SSH-tunneled RoundTripper).
func DialWithClient(url string, hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{url: url, http: hc}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// Call invokes method with params and, if out is non-nil, decodes the result
// into it. A JSON-RPC error response is returned as a Go error.
func (c *Client) Call(ctx context.Context, method string, out any, params ...any) error {
	if params == nil {
		params = []any{}
	}
	reqBody, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: c.id.Add(1), Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("rpc: marshal %s: %w", method, err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("rpc: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("rpc: %s: %w", method, err)
	}
	defer resp.Body.Close()

	var rr rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return fmt.Errorf("rpc: decode %s: %w", method, err)
	}
	if rr.Error != nil {
		return fmt.Errorf("rpc: %s: server error %d: %s", method, rr.Error.Code, rr.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(rr.Result, out); err != nil {
			return fmt.Errorf("rpc: unmarshal %s result: %w", method, err)
		}
	}
	return nil
}

// BlockNumber returns eth_blockNumber as a decimal uint64.
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	var s string
	if err := c.Call(ctx, "eth_blockNumber", &s); err != nil {
		return 0, err
	}
	return parseHexUint(s)
}

// ChainID returns eth_chainId as a decimal uint64.
func (c *Client) ChainID(ctx context.Context) (uint64, error) {
	var s string
	if err := c.Call(ctx, "eth_chainId", &s); err != nil {
		return 0, err
	}
	return parseHexUint(s)
}

// PeerCount returns net_peerCount as a decimal uint64.
func (c *Client) PeerCount(ctx context.Context) (uint64, error) {
	var s string
	if err := c.Call(ctx, "net_peerCount", &s); err != nil {
		return 0, err
	}
	return parseHexUint(s)
}

// Syncing reports whether the node is syncing (eth_syncing returns false when
// fully synced, or a progress object while syncing).
func (c *Client) Syncing(ctx context.Context) (bool, error) {
	var raw json.RawMessage
	if err := c.Call(ctx, "eth_syncing", &raw); err != nil {
		return false, err
	}
	return strings.TrimSpace(string(raw)) != "false", nil
}

// BalanceAt returns the latest balance (in wei) of a 0x-hex address.
func (c *Client) BalanceAt(ctx context.Context, addr string) (*big.Int, error) {
	var s string
	if err := c.Call(ctx, "eth_getBalance", &s, addr, "latest"); err != nil {
		return nil, err
	}
	v, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(s), "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("rpc: bad balance %q", s)
	}
	return v, nil
}

// NonceAt returns the latest transaction count (nonce) of a 0x-hex address.
func (c *Client) NonceAt(ctx context.Context, addr string) (uint64, error) {
	var s string
	if err := c.Call(ctx, "eth_getTransactionCount", &s, addr, "latest"); err != nil {
		return 0, err
	}
	return parseHexUint(s)
}

// CodeAt returns the deployed bytecode at a 0x-hex address ("0x" if none).
func (c *Client) CodeAt(ctx context.Context, addr string) (string, error) {
	var s string
	if err := c.Call(ctx, "eth_getCode", &s, addr, "latest"); err != nil {
		return "", err
	}
	return s, nil
}

// EthCall runs a read-only contract call (eth_call) and returns the 0x-hex
// result. data is the 0x-hex calldata.
func (c *Client) EthCall(ctx context.Context, to, data string) (string, error) {
	var s string
	if err := c.Call(ctx, "eth_call", &s, map[string]string{"to": to, "data": data}, "latest"); err != nil {
		return "", err
	}
	return s, nil
}

// TxReceipt returns the raw eth_getTransactionReceipt result for a tx hash, or
// nil (no error) when the transaction is still pending.
func (c *Client) TxReceipt(ctx context.Context, hash string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.Call(ctx, "eth_getTransactionReceipt", &raw, hash); err != nil {
		return nil, err
	}
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}
	return raw, nil
}

func parseHexUint(s string) (uint64, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return 0, fmt.Errorf("rpc: empty hex quantity")
	}
	n, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("rpc: parse hex %q: %w", s, err)
	}
	return n, nil
}
