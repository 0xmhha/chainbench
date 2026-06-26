package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xmhha/chainbench/network/internal/abiutil"
	"github.com/0xmhha/chainbench/network/internal/events"
)

// rpcMethodRe constrains node.rpc method names to a conservative
// alphanumeric+underscore pattern that mirrors the TS-side RPC_METHOD regex.
// JSON-RPC itself allows richer characters but every Ethereum-flavoured
// method (eth_*, debug_*, txpool_*, ...) fits this shape, and rejecting
// anything else at the boundary keeps the upstream surface predictable.
var rpcMethodRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// Per-call RPC deadlines, shared across node read/tx handlers in this package.
const (
	// nodeReadTimeout bounds read-only RPC calls (block_number, chain_id,
	// balance, gas_price, contract_call, events_get, account_state).
	nodeReadTimeout = 10 * time.Second
	// nodeWriteTimeout bounds write/broadcast calls (tx_send, contract_deploy,
	// fee_delegation) and the generic node.rpc passthrough, which may carry a write.
	nodeWriteTimeout = 30 * time.Second
)

// newHandleNodeBlockNumber returns a handler that opens an ethclient
// connection to the resolved node's HTTP endpoint and returns the current
// head block number. Works uniformly across local and remote networks
// because both populate types.Node.Http.
//
// Args: {network?: "local"|"<remote-name>", node_id: "nodeN"}
// Returns: {network, node_id, block_number}
//
// Error mapping:
//
//	INVALID_ARGS   — missing node_id, bad network name, unknown node
//	UPSTREAM_ERROR — state load failure, Dial failure, or eth_blockNumber failure
func newHandleNodeBlockNumber(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string `json:"network"`
			NodeID  string `json:"node_id"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		bn, err := client.BlockNumber(ctx)
		if err != nil {
			return nil, NewUpstream("eth_blockNumber", err)
		}
		return map[string]any{
			"network":      networkName,
			"node_id":      node.Id,
			"block_number": bn,
		}, nil
	}
}

// newHandleNodeChainID returns a handler for node.chain_id.
//
// Args: {network?: "local"|"<remote-name>", node_id: "nodeN"}
// Result: {network, node_id, chain_id: <uint64>}
//
// Error mapping:
//
//	INVALID_ARGS   — missing node_id, bad network name, unknown node
//	UPSTREAM_ERROR — state load failure, dial failure, or eth_chainId failure
func newHandleNodeChainID(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string `json:"network"`
			NodeID  string `json:"node_id"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		cid, err := client.ChainID(ctx)
		if err != nil {
			return nil, NewUpstream("eth_chainId", err)
		}
		return map[string]any{
			"network":  networkName,
			"node_id":  node.Id,
			"chain_id": cid.Uint64(),
		}, nil
	}
}

// newHandleNodeBalance returns a handler for node.balance.
//
// Args: {network?, node_id, address: "0x...", block_number?: <int>|"latest"|"earliest"|"pending"}
// Result: {network, node_id, address, block: <string>, balance: "0x..."}
//
// block_number is optional; when omitted, defaults to "latest". Integer values
// are passed directly to eth_getBalance; the string labels are translated as:
//
//	"latest"   — nil block (ethclient interprets nil as latest head)
//	"earliest" — block 0
//	"pending"  — nil (approximated as "latest" for 3b.2c; promote to a follow-up
//	             if real pending-state semantics are required)
//
// Error mapping:
//
//	INVALID_ARGS   — missing/invalid address, invalid block_number (negative
//	                 int, unknown label, or wrong JSON shape), missing node_id,
//	                 bad network name, unknown node
//	UPSTREAM_ERROR — state load failure, dial failure, or eth_getBalance failure
func newHandleNodeBalance(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network     string          `json:"network"`
			NodeID      string          `json:"node_id"`
			Address     string          `json:"address"`
			BlockNumber json.RawMessage `json:"block_number"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Address == "" {
			return nil, NewInvalidArgs("args.address is required")
		}
		if !common.IsHexAddress(req.Address) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.address is not a valid hex address: %q", req.Address))
		}
		addr := common.HexToAddress(req.Address)

		blockNum, blockLabel, err := parseBlockNumberArg(req.BlockNumber)
		if err != nil {
			return nil, err
		}

		networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		bal, err := client.BalanceAt(ctx, addr, blockNum)
		if err != nil {
			return nil, NewUpstream("eth_getBalance", err)
		}
		// big.Int.Text(16) returns lowercase hex without 0x prefix — prepend
		// the canonical Ethereum hex prefix for the wire response.
		return map[string]any{
			"network": networkName,
			"node_id": node.Id,
			"address": req.Address,
			"block":   blockLabel,
			"balance": "0x" + bal.Text(16),
		}, nil
	}
}

// newHandleNodeGasPrice returns a handler for node.gas_price.
//
// Args: {network?, node_id}
// Result: {network, node_id, gas_price: "0x..."}
//
// Wraps eth_gasPrice (ethclient.SuggestGasPrice). No fee-history / 1559
// semantics here — see node.fee_history follow-up for the richer signal.
//
// Error mapping:
//
//	INVALID_ARGS   — missing node_id, bad network name, unknown node
//	UPSTREAM_ERROR — state load failure, dial failure, or eth_gasPrice failure
func newHandleNodeGasPrice(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string `json:"network"`
			NodeID  string `json:"node_id"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		gp, err := client.GasPrice(ctx)
		if err != nil {
			return nil, NewUpstream("eth_gasPrice", err)
		}
		return map[string]any{
			"network":   networkName,
			"node_id":   node.Id,
			"gas_price": "0x" + gp.Text(16),
		}, nil
	}
}

// newHandleNodeContractCall returns a handler for node.contract_call. It is a
// thin read-only wrapper around eth_call with two mutually-exclusive modes:
//
//   - Calldata mode: caller provides raw `calldata` hex. Server forwards the
//     bytes verbatim and returns `result_raw` only.
//   - ABI mode: caller provides `abi` (JSON ABI string) + `method` + `args`.
//     Server parses the ABI, coerces JSON args via abiutil.CoerceArgs, packs
//     the method call (4-byte selector + encoded args) via
//     abiutil.PackMethodCall, forwards to eth_call, then decodes the response
//     via abiutil.UnpackMethodResult. Returns both `result_raw` and
//     `result_decoded`.
//
// `block_number` accepts an integer or one of "latest" / "earliest" / "pending"
// (matching node.balance's existing pattern). `from` is optional; when
// omitted, eth_call sees the zero address (sufficient for view function calls
// that do not depend on msg.sender).
//
// Args: {network?, node_id, contract_address, calldata? OR abi?+method?+args?,
//
//	block_number?: <int>|"latest"|"earliest"|"pending", from?: "0x..."}
//
// Result: {result_raw: "0x...", block: <label>, result_decoded?: [<value>, ...]}
//
// Error mapping:
//
//	INVALID_ARGS   — missing/invalid contract_address; missing both calldata
//	                 and abi; both calldata and abi provided; bad calldata hex;
//	                 ABI parse fail; method not in ABI; arg-count or arg-type
//	                 mismatch; invalid block_number (negative int, unknown
//	                 label, or wrong JSON shape); bad from address
//	UPSTREAM_ERROR — state load failure, dial failure, or eth_call failure
//	INTERNAL       — abiutil.UnpackMethodResult failure (invariant — inputs
//	                 already validated, so a decode failure indicates a remote
//	                 contract returning data that does not match the declared
//	                 ABI)
func newHandleNodeContractCall(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network         string          `json:"network"`
			NodeID          string          `json:"node_id"`
			ContractAddress string          `json:"contract_address"`
			Calldata        string          `json:"calldata"`
			ABI             string          `json:"abi"`
			Method          string          `json:"method"`
			Args            []any           `json:"args"`
			BlockNumber     json.RawMessage `json:"block_number"`
			From            string          `json:"from"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.ContractAddress == "" {
			return nil, NewInvalidArgs("args.contract_address is required")
		}
		if !common.IsHexAddress(req.ContractAddress) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.contract_address is not a valid hex address: %q", req.ContractAddress))
		}
		contractAddr := common.HexToAddress(req.ContractAddress)

		hasCalldata := req.Calldata != ""
		hasABI := req.ABI != ""
		if hasCalldata && hasABI {
			return nil, NewInvalidArgs("args.calldata and args.abi are mutually exclusive")
		}
		if !hasCalldata && !hasABI {
			return nil, NewInvalidArgs("either args.calldata or args.abi+method+args is required")
		}

		var calldata []byte
		var parsedABI abi.ABI
		var methodName string
		if hasCalldata {
			trimmed := strings.TrimPrefix(req.Calldata, "0x")
			decoded, err := hex.DecodeString(trimmed)
			if err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.calldata is not valid hex: %q", req.Calldata))
			}
			calldata = decoded
		} else {
			if req.Method == "" {
				return nil, NewInvalidArgs("args.method is required when args.abi is provided")
			}
			p, err := abiutil.ParseABI(req.ABI)
			if err != nil {
				return nil, NewInvalidArgs(err.Error())
			}
			parsedABI = p
			methodName = req.Method
			packed, err := abiutil.PackMethodCall(parsedABI, methodName, req.Args)
			if err != nil {
				return nil, NewInvalidArgs(err.Error())
			}
			calldata = packed
		}

		blockNum, blockLabel, err := parseBlockNumberArg(req.BlockNumber)
		if err != nil {
			return nil, err
		}

		var from common.Address
		if req.From != "" {
			if !common.IsHexAddress(req.From) {
				return nil, NewInvalidArgs(fmt.Sprintf("args.from is not a valid hex address: %q", req.From))
			}
			from = common.HexToAddress(req.From)
		}

		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		msg := ethereum.CallMsg{To: &contractAddr, Data: calldata}
		if from != (common.Address{}) {
			msg.From = from
		}
		result, err := client.CallContract(ctx, msg, blockNum)
		if err != nil {
			return nil, NewUpstream("eth_call", err)
		}

		out := map[string]any{
			"result_raw": "0x" + hex.EncodeToString(result),
			"block":      blockLabel,
		}
		if hasABI {
			decoded, derr := abiutil.UnpackMethodResult(parsedABI, methodName, result)
			if derr != nil {
				return nil, NewInternal("unpack result", derr)
			}
			out["result_decoded"] = decoded
		}
		return out, nil
	}
}

// parseBlockNumberArg parses an optional block_number that is an integer (≥0)
// or one of "latest"/"earliest"/"pending". It returns the *big.Int to hand to
// ethclient (nil = latest head, big.NewInt(0) = earliest) plus the label to
// echo back in the response ("latest" by default; "pending" is degraded to
// latest upstream but still surfaced here). Unlike parseBlockArg (events_get)
// this variant does NOT accept "0x" hex, mirroring node.balance /
// contract_call / account_state, whose JSON callers always send int or label.
func parseBlockNumberArg(raw json.RawMessage) (*big.Int, string, error) {
	blockLabel := "latest"
	if len(raw) == 0 {
		return nil, blockLabel, nil
	}
	var asInt int64
	var asStr string
	if err := json.Unmarshal(raw, &asInt); err == nil {
		if asInt < 0 {
			return nil, "", NewInvalidArgs(fmt.Sprintf("args.block_number must be non-negative, got %d", asInt))
		}
		return big.NewInt(asInt), fmt.Sprintf("%d", asInt), nil
	}
	if err := json.Unmarshal(raw, &asStr); err == nil {
		switch asStr {
		case "latest", "pending":
			return nil, asStr, nil
		case "earliest":
			return big.NewInt(0), asStr, nil
		default:
			return nil, "", NewInvalidArgs(fmt.Sprintf("args.block_number label invalid: %q", asStr))
		}
	}
	return nil, "", NewInvalidArgs("args.block_number must be an integer or a block label string")
}

// newHandleNodeAccountState returns a handler for node.account_state — a
// read-only command that composites balance / nonce / code / storage subset
// reads against a single account at a single block height. `fields` selects
// which subset to fetch; default is balance + nonce + code (storage is
// opt-in only because it requires a separate `storage_key` and would issue
// a wasteful RPC otherwise).
//
// Args: {network?, node_id, address, fields?: [...], storage_key?,
//
//	block_number?: <int>|"latest"|"earliest"|"pending"}
//
// Result: {address, block, balance?, nonce?, code?, storage?}
//
// The nonce read uses NonceAt (eth_getTransactionCount with the explicit
// block tag) rather than PendingNonceAt — pending-pool semantics belong to
// tx_send, not state inspection.
//
// Validation order: address shape → field-name validity → storage_key shape
// (when wantStorage) → block_number parse → resolveNode → dial → per-field
// RPC. Errors fire before any network round-trip when caused by malformed
// input.
//
// Error mapping:
//
//	INVALID_ARGS   — missing/invalid address; unknown field name; missing or
//	                 malformed storage_key when 'storage' requested; invalid
//	                 block_number; unknown node / bad network name
//	UPSTREAM_ERROR — state load failure, dial failure, or any of the four
//	                 underlying eth_get* failures
func newHandleNodeAccountState(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network     string          `json:"network"`
			NodeID      string          `json:"node_id"`
			Address     string          `json:"address"`
			Fields      []string        `json:"fields"`
			StorageKey  string          `json:"storage_key"`
			BlockNumber json.RawMessage `json:"block_number"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}

		if req.Address == "" {
			return nil, NewInvalidArgs("args.address is required")
		}
		if !common.IsHexAddress(req.Address) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.address is not a valid hex address: %q", req.Address))
		}
		addr := common.HexToAddress(req.Address)

		// Default fields = balance + nonce + code. Storage requires explicit
		// request because it needs a storage_key and would otherwise issue
		// a wasteful RPC against a slot the caller did not ask for.
		fields := req.Fields
		if len(fields) == 0 {
			fields = []string{"balance", "nonce", "code"}
		}

		var wantBalance, wantNonce, wantCode, wantStorage bool
		for _, f := range fields {
			switch f {
			case "balance":
				wantBalance = true
			case "nonce":
				wantNonce = true
			case "code":
				wantCode = true
			case "storage":
				wantStorage = true
			default:
				return nil, NewInvalidArgs(fmt.Sprintf(
					"args.fields contains unknown field: %q (must be balance/nonce/code/storage)", f,
				))
			}
		}

		// Storage requires storage_key — validate before any RPC round-trip
		// so callers get a fast, deterministic INVALID_ARGS for malformed
		// input rather than a delayed UPSTREAM error from a confused upstream.
		var storageKey common.Hash
		if wantStorage {
			if req.StorageKey == "" {
				return nil, NewInvalidArgs("args.storage_key is required when 'storage' is in fields")
			}
			if !strings.HasPrefix(req.StorageKey, "0x") || len(req.StorageKey) != 66 {
				return nil, NewInvalidArgs(fmt.Sprintf(
					"args.storage_key must be 0x + 64 hex chars: %q", req.StorageKey,
				))
			}
			if _, err := hex.DecodeString(strings.TrimPrefix(req.StorageKey, "0x")); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.storage_key not valid hex: %q", req.StorageKey))
			}
			storageKey = common.HexToHash(req.StorageKey)
		}

		blockNum, blockLabel, err := parseBlockNumberArg(req.BlockNumber)
		if err != nil {
			return nil, err
		}

		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		out := map[string]any{
			"address": req.Address,
			"block":   blockLabel,
		}

		if wantBalance {
			bal, err := client.BalanceAt(ctx, addr, blockNum)
			if err != nil {
				return nil, NewUpstream("eth_getBalance", err)
			}
			out["balance"] = "0x" + bal.Text(16)
		}
		if wantNonce {
			// NonceAt — eth_getTransactionCount at the given block tag.
			// Distinct from PendingNonceAt: state inspection wants the
			// historical/latest mined nonce, not pending-pool inclusion.
			nonce, err := client.NonceAt(ctx, addr, blockNum)
			if err != nil {
				return nil, NewUpstream("eth_getTransactionCount", err)
			}
			out["nonce"] = nonce
		}
		if wantCode {
			code, err := client.CodeAt(ctx, addr, blockNum)
			if err != nil {
				return nil, NewUpstream("eth_getCode", err)
			}
			out["code"] = "0x" + hex.EncodeToString(code)
		}
		if wantStorage {
			slot, err := client.StorageAt(ctx, addr, storageKey, blockNum)
			if err != nil {
				return nil, NewUpstream("eth_getStorageAt", err)
			}
			out["storage"] = "0x" + hex.EncodeToString(slot)
		}

		return out, nil
	}
}

// newHandleNodeRpc returns a handler for node.rpc — a generic JSON-RPC
// passthrough. The handler validates the method name against rpcMethodRe,
// decodes params as a JSON array (or null / missing for the no-arg case),
// resolves the node, and forwards the call to the upstream via the underlying
// rpc.Client. The raw RPC result is returned inside a {"result": <raw>}
// envelope so the MCP layer can pretty-print or pass through unchanged
// without ethclient interpreting the value.
//
// Args: {network?: "local"|"<remote-name>", node_id: "nodeN", method: "eth_*",
//
//	params?: [<arg>, ...] | null}
//
// Result: {result: <raw json.RawMessage>}
//
// The 30-second timeout matches the existing read handlers; node.rpc covers
// both read and write methods (eth_sendRawTransaction is a valid passthrough
// target) but the wider deadline gives slow upstreams headroom without
// blocking indefinitely.
//
// Error mapping:
//
//	INVALID_ARGS   — missing node_id, bad network name, unknown node, empty
//	                 method, method failing rpcMethodRe, params not a JSON
//	                 array (objects / scalars rejected)
//	UPSTREAM_ERROR — state load failure, dial failure, or any RPC-side
//	                 error (transport, JSON-RPC error envelope, decode)
func newHandleNodeRpc(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string          `json:"network"`
			NodeID  string          `json:"node_id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Method == "" {
			return nil, NewInvalidArgs("args.method is required")
		}
		if !rpcMethodRe.MatchString(req.Method) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.method invalid: %q", req.Method))
		}

		// Decode params: must be a JSON array OR null/missing. A nil
		// paramsList serialises as the empty params slot rpc.Client expects
		// for no-arg methods, so explicit null and missing both collapse to
		// the same wire shape.
		var paramsList []any
		if len(req.Params) > 0 && string(req.Params) != "null" {
			if err := json.Unmarshal(req.Params, &paramsList); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.params must be a JSON array or null: %v", err))
			}
		}

		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), nodeWriteTimeout)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		var raw json.RawMessage
		if err := client.CallContext(ctx, &raw, req.Method, paramsList...); err != nil {
			return nil, NewUpstream(req.Method, err)
		}
		return map[string]any{"result": raw}, nil
	}
}
