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
	defer func() { _ = resp.Body.Close() }()

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

// EstimateGas returns eth_estimateGas for a call to `to` with 0x-hex calldata.
// from is optional (some contract calls estimate differently per caller).
func (c *Client) EstimateGas(ctx context.Context, from, to, data string) (uint64, error) {
	arg := map[string]string{"to": to, "data": data}
	if from != "" {
		arg["from"] = from
	}
	var s string
	if err := c.Call(ctx, "eth_estimateGas", &s, arg); err != nil {
		return 0, err
	}
	return parseHexUint(s)
}

// Coinbase returns the node's etherbase (eth_coinbase) — for a validator node
// this is its unlocked block-signing account, usable as the `from` of a
// node-signed transaction.
func (c *Client) Coinbase(ctx context.Context) (string, error) {
	var s string
	if err := c.Call(ctx, "eth_coinbase", &s); err != nil {
		return "", err
	}
	return s, nil
}

// LogFilter is an eth_getLogs filter. Empty fields are omitted, so the zero
// value asks for every log the node will return.
type LogFilter struct {
	Address   string   `json:"address,omitempty"`
	Topics    []string `json:"-"`
	FromBlock string   `json:"fromBlock,omitempty"`
	ToBlock   string   `json:"toBlock,omitempty"`
}

// Logs returns the raw log objects matching filter (eth_getLogs). Decoding is
// left to the caller: what a log's data means is contract-specific, and the
// generic client should not pretend to know an ABI.
func (c *Client) Logs(ctx context.Context, filter LogFilter) ([]map[string]any, error) {
	arg := map[string]any{}
	if filter.Address != "" {
		arg["address"] = filter.Address
	}
	if len(filter.Topics) > 0 {
		topics := make([]any, len(filter.Topics))
		for i, t := range filter.Topics {
			// An empty topic position is a wildcard, which JSON-RPC spells null.
			if t == "" {
				topics[i] = nil
				continue
			}
			topics[i] = t
		}
		arg["topics"] = topics
	}
	if filter.FromBlock != "" {
		arg["fromBlock"] = filter.FromBlock
	}
	if filter.ToBlock != "" {
		arg["toBlock"] = filter.ToBlock
	}
	var out []map[string]any
	if err := c.Call(ctx, "eth_getLogs", &out, arg); err != nil {
		return nil, err
	}
	return out, nil
}

// Enode returns the node's own devp2p enode URL (admin_nodeInfo.enode), the
// address other nodes use to add or drop it as a peer.
func (c *Client) Enode(ctx context.Context) (string, error) {
	var info struct {
		Enode string `json:"enode"`
	}
	if err := c.Call(ctx, "admin_nodeInfo", &info); err != nil {
		return "", err
	}
	if info.Enode == "" {
		return "", fmt.Errorf("rpc: admin_nodeInfo returned no enode")
	}
	return info.Enode, nil
}

// AddPeer asks the node to connect to enode (admin_addPeer).
func (c *Client) AddPeer(ctx context.Context, enode string) error {
	var ok bool
	return c.Call(ctx, "admin_addPeer", &ok, enode)
}

// RemovePeer asks the node to drop enode (admin_removePeer). It is how a spec
// severs a link to partition a network.
func (c *Client) RemovePeer(ctx context.Context, enode string) error {
	var ok bool
	return c.Call(ctx, "admin_removePeer", &ok, enode)
}

// Block is the subset of an Ethereum block chainbench reads. BaseFeePerGas is
// nil on chains/blocks without EIP-1559. GasLimit and GasUsed drive the gas-load
// primitive: a burn tx sized to a fraction of GasLimit fills the block it lands
// in, and GasUsed/GasLimit is that block's utilization.
type Block struct {
	Number        uint64
	Hash          string
	Miner         string
	Timestamp     uint64
	GasLimit      uint64
	GasUsed       uint64
	BaseFeePerGas *big.Int
}

// BlockByNumber fetches a block by tag ("latest", "earliest", or a 0x-hex number
// like "0x0") via eth_getBlockByNumber(tag, false). An empty result yields a
// zero Block.
func (c *Client) BlockByNumber(ctx context.Context, tag string) (Block, error) {
	var b struct {
		Number        string `json:"number"`
		Hash          string `json:"hash"`
		Miner         string `json:"miner"`
		Timestamp     string `json:"timestamp"`
		GasLimit      string `json:"gasLimit"`
		GasUsed       string `json:"gasUsed"`
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	if err := c.Call(ctx, "eth_getBlockByNumber", &b, tag, false); err != nil {
		return Block{}, err
	}
	blk := Block{Hash: b.Hash, Miner: b.Miner}
	var err error
	if b.Number != "" {
		if blk.Number, err = parseHexUint(b.Number); err != nil {
			return Block{}, fmt.Errorf("rpc: block number: %w", err)
		}
	}
	if b.Timestamp != "" {
		if blk.Timestamp, err = parseHexUint(b.Timestamp); err != nil {
			return Block{}, fmt.Errorf("rpc: block timestamp: %w", err)
		}
	}
	if b.GasLimit != "" {
		if blk.GasLimit, err = parseHexUint(b.GasLimit); err != nil {
			return Block{}, fmt.Errorf("rpc: block gas limit: %w", err)
		}
	}
	if b.GasUsed != "" {
		if blk.GasUsed, err = parseHexUint(b.GasUsed); err != nil {
			return Block{}, fmt.Errorf("rpc: block gas used: %w", err)
		}
	}
	if b.BaseFeePerGas != "" {
		v, ok := new(big.Int).SetString(strings.TrimPrefix(b.BaseFeePerGas, "0x"), 16)
		if !ok {
			return Block{}, fmt.Errorf("rpc: bad baseFeePerGas %q", b.BaseFeePerGas)
		}
		blk.BaseFeePerGas = v
	}
	return blk, nil
}

// HeadBlock returns the latest block's number, hash, and miner (producer). It is
// a convenience wrapper over BlockByNumber("latest").
func (c *Client) HeadBlock(ctx context.Context) (number uint64, hash, miner string, err error) {
	b, err := c.BlockByNumber(ctx, "latest")
	return b.Number, b.Hash, b.Miner, err
}

// SendTxArgs are the fields of an eth_sendTransaction call (node-side signing).
// From must be an account the node has unlocked (e.g. a validator coinbase);
// omitted fields (gas, gas price) are filled by the node.
type SendTxArgs struct {
	From  string `json:"from"`
	To    string `json:"to,omitempty"`
	Data  string `json:"data,omitempty"`
	Value string `json:"value,omitempty"` // 0x-hex wei
	Gas   string `json:"gas,omitempty"`   // 0x-hex
	// GasPrice is the legacy (pre-1559) price. Setting it alongside the fee-cap
	// fields is invalid; the caller picks one form.
	GasPrice string `json:"gasPrice,omitempty"` // 0x-hex wei
	// MaxFeePerGas and MaxPriorityFeePerGas are the EIP-1559 fee caps, which
	// fee-policy tests set deliberately low to be rejected.
	MaxFeePerGas         string `json:"maxFeePerGas,omitempty"`         // 0x-hex wei
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas,omitempty"` // 0x-hex wei
	// Nonce pins the transaction's position. Omitted, the node assigns the next
	// one; set explicitly, a spec can submit out of order or replace a pending
	// transaction at the same nonce.
	Nonce string `json:"nonce,omitempty"` // 0x-hex
	// AccessList, when present, selects a typed transaction (EIP-2930 type 0x01
	// with GasPrice, or type 0x02 with the fee caps). It is [{address,
	// storageKeys:[...]}] — the empty list [] is valid and still selects a typed
	// envelope, so it is passed through as-is: a []AccessTuple with omitempty
	// would silently drop the empty case and downgrade the tx to legacy.
	AccessList any `json:"accessList,omitempty"`
}

// SendTransaction submits a node-signed transaction (eth_sendTransaction) and
// returns its hash. The node signs with From's unlocked key — unlike the
// accounts.Wallet path, no private key is held client-side. Used for flows that
// must originate from a node-held account, e.g. a validator casting a
// governance vote.
func (c *Client) SendTransaction(ctx context.Context, args SendTxArgs) (string, error) {
	var hash string
	if err := c.Call(ctx, "eth_sendTransaction", &hash, args); err != nil {
		return "", err
	}
	return hash, nil
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
