package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xmhha/chainbench/network/internal/abiutil"
	"github.com/0xmhha/chainbench/network/internal/events"
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

		var blockNum *big.Int
		blockLabel := "latest"
		if len(req.BlockNumber) > 0 {
			// Try integer first, then string label — json.RawMessage lets the
			// caller submit either form under the same field name.
			var asInt int64
			var asStr string
			if err := json.Unmarshal(req.BlockNumber, &asInt); err == nil {
				if asInt < 0 {
					return nil, NewInvalidArgs(fmt.Sprintf("args.block_number must be non-negative, got %d", asInt))
				}
				blockNum = big.NewInt(asInt)
				blockLabel = fmt.Sprintf("%d", asInt)
			} else if err := json.Unmarshal(req.BlockNumber, &asStr); err == nil {
				switch asStr {
				case "latest", "pending":
					// ethclient interprets nil as "latest"; "pending" is
					// approximated as "latest" here — promote to a follow-up
					// if real pending-state semantics are ever required.
					blockLabel = asStr
				case "earliest":
					blockLabel = asStr
					blockNum = big.NewInt(0)
				default:
					return nil, NewInvalidArgs(fmt.Sprintf("args.block_number label invalid: %q", asStr))
				}
			} else {
				return nil, NewInvalidArgs("args.block_number must be an integer or a block label string")
			}
		}

		networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

		// block_number — int OR "latest" / "earliest" / "pending". Mirrors
		// node.balance's parsing: nil signals "latest" to ethclient,
		// big.NewInt(0) for earliest, integers passed as-is. blockLabel echoes
		// the caller-provided label (or numeric form) back in the result so
		// "pending" (silently degraded to "latest" upstream) is observable.
		var blockNum *big.Int
		blockLabel := "latest"
		if len(req.BlockNumber) > 0 {
			var asInt int64
			var asStr string
			if err := json.Unmarshal(req.BlockNumber, &asInt); err == nil {
				if asInt < 0 {
					return nil, NewInvalidArgs(fmt.Sprintf("args.block_number must be non-negative, got %d", asInt))
				}
				blockNum = big.NewInt(asInt)
				blockLabel = fmt.Sprintf("%d", asInt)
			} else if err := json.Unmarshal(req.BlockNumber, &asStr); err == nil {
				switch asStr {
				case "latest", "pending":
					// nil — ethclient interprets nil as "latest"; "pending" is
					// approximated as "latest" here (consistent with node.balance).
					blockLabel = asStr
				case "earliest":
					blockLabel = asStr
					blockNum = big.NewInt(0)
				default:
					return nil, NewInvalidArgs(fmt.Sprintf("args.block_number label invalid: %q", asStr))
				}
			} else {
				return nil, NewInvalidArgs("args.block_number must be an integer or a block label string")
			}
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
