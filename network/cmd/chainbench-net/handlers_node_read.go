package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

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
