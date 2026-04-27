package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/signer"
)

// newHandleNodeTxSend signs + broadcasts a transaction against the resolved
// node. Fee mode is selected from args: gas_price → legacy (type 0);
// max_fee_per_gas + max_priority_fee_per_gas → EIP-1559 (type 2). Mixed or
// partial 1559 args are rejected as INVALID_ARGS. Missing nonce / gas /
// gas_price (legacy only) are auto-filled via the remote.Client. Signer alias
// is resolved per-request via signer.Load.
//
// Args: {network?, node_id, signer, to, value?, data?, gas?,
//
//	gas_price?, max_fee_per_gas?, max_priority_fee_per_gas?, nonce?}
//
// Result: {tx_hash: "0x..."}
//
// Error mapping (spec §11 + §3.1):
//
//	INVALID_ARGS   — malformed args, missing signer / to, bad to / value / data /
//	                 gas_price / max_fee / max_priority_fee hex, mixed legacy + 1559
//	                 fee fields, partial 1559 fee fields, signer.ErrInvalidAlias,
//	                 signer.ErrUnknownAlias
//	UPSTREAM_ERROR — signer.ErrInvalidKey (config failure), dial failure, chainID /
//	                 nonce / gas / gas_price fetch failure, SendTransaction failure
//	INTERNAL       — SignTx failure (invariant breach — inputs validated earlier)
func newHandleNodeTxSend(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network              string  `json:"network"`
			NodeID               string  `json:"node_id"`
			Signer               string  `json:"signer"`
			To                   string  `json:"to"`
			Value                string  `json:"value"`
			Data                 string  `json:"data"`
			Gas                  *uint64 `json:"gas"`
			GasPrice             string  `json:"gas_price"`
			MaxFeePerGas         string  `json:"max_fee_per_gas"`
			MaxPriorityFeePerGas string  `json:"max_priority_fee_per_gas"`
			Nonce                *uint64 `json:"nonce"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Signer == "" {
			return nil, NewInvalidArgs("args.signer is required")
		}
		if req.To == "" {
			return nil, NewInvalidArgs("args.to is required")
		}
		if !common.IsHexAddress(req.To) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.to is not a valid hex address: %q", req.To))
		}
		to := common.HexToAddress(req.To)

		// Parse optional hex fields up front so structural issues surface as
		// INVALID_ARGS before any network round-trip.
		value := big.NewInt(0)
		if req.Value != "" {
			parsed, ok := new(big.Int).SetString(strings.TrimPrefix(req.Value, "0x"), 16)
			if !ok {
				return nil, NewInvalidArgs(fmt.Sprintf("args.value is not valid hex: %q", req.Value))
			}
			value = parsed
		}
		var data []byte
		if req.Data != "" && req.Data != "0x" {
			trimmed := strings.TrimPrefix(req.Data, "0x")
			decoded, err := hex.DecodeString(trimmed)
			if err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.data is not valid hex: %q", req.Data))
			}
			data = decoded
		}

		// Fee-mode selection: legacy (gas_price) vs EIP-1559 (max_fee + tip).
		// Mixed and partial 1559 are rejected at the boundary so structural
		// mistakes never reach the signer.
		hasLegacy := req.GasPrice != ""
		hasMaxFee := req.MaxFeePerGas != ""
		hasTip := req.MaxPriorityFeePerGas != ""

		if hasLegacy && (hasMaxFee || hasTip) {
			return nil, NewInvalidArgs("args.gas_price cannot be combined with args.max_fee_per_gas or args.max_priority_fee_per_gas")
		}
		if hasMaxFee != hasTip {
			return nil, NewInvalidArgs("args.max_fee_per_gas and args.max_priority_fee_per_gas must both be provided when using EIP-1559")
		}
		useDynamicFee := hasMaxFee && hasTip

		var maxFee, maxPriorityFee *big.Int
		if useDynamicFee {
			var ok bool
			maxFee, ok = new(big.Int).SetString(strings.TrimPrefix(req.MaxFeePerGas, "0x"), 16)
			if !ok {
				return nil, NewInvalidArgs(fmt.Sprintf("args.max_fee_per_gas is not valid hex: %q", req.MaxFeePerGas))
			}
			maxPriorityFee, ok = new(big.Int).SetString(strings.TrimPrefix(req.MaxPriorityFeePerGas, "0x"), 16)
			if !ok {
				return nil, NewInvalidArgs(fmt.Sprintf("args.max_priority_fee_per_gas is not valid hex: %q", req.MaxPriorityFeePerGas))
			}
		}

		// Signer resolution per-request. InvalidAlias / UnknownAlias are
		// structural input issues (INVALID_ARGS); InvalidKey is a config failure
		// of the operator-supplied env var (UPSTREAM_ERROR per spec §11).
		s, serr := signer.Load(signer.Alias(req.Signer))
		if serr != nil {
			if errors.Is(serr, signer.ErrInvalidAlias) ||
				errors.Is(serr, signer.ErrUnknownAlias) {
				return nil, NewInvalidArgs(serr.Error())
			}
			return nil, NewUpstream("signer load", serr)
		}

		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		chainID, err := client.ChainID(ctx)
		if err != nil {
			return nil, NewUpstream("eth_chainId", err)
		}

		nonce, err := resolveNonce(ctx, client, s.Address(), req.Nonce)
		if err != nil {
			return nil, err
		}
		gas, err := resolveGas(ctx, client, req.Gas, s.Address(), to, value, data)
		if err != nil {
			return nil, err
		}

		var unsigned *ethtypes.Transaction
		if useDynamicFee {
			unsigned = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				ChainID:   chainID,
				Nonce:     nonce,
				GasTipCap: maxPriorityFee,
				GasFeeCap: maxFee,
				Gas:       gas,
				To:        &to,
				Value:     value,
				Data:      data,
			})
		} else {
			gasPrice, err := resolveGasPrice(ctx, client, req.GasPrice)
			if err != nil {
				return nil, err
			}
			unsigned = ethtypes.NewTx(&ethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: gasPrice,
				Gas:      gas,
				To:       &to,
				Value:    value,
				Data:     data,
			})
		}
		signed, err := s.SignTx(ctx, unsigned, chainID)
		if err != nil {
			return nil, NewInternal("sign tx", err)
		}
		if err := client.SendTransaction(ctx, signed); err != nil {
			return nil, NewUpstream("eth_sendRawTransaction", err)
		}
		return map[string]any{"tx_hash": signed.Hash().Hex()}, nil
	}
}

// resolveNonce returns explicit when provided; otherwise fetches the next
// pending-state nonce for `from` from the endpoint. Classifies RPC failure
// as UPSTREAM_ERROR.
func resolveNonce(ctx context.Context, client *remote.Client, from common.Address, explicit *uint64) (uint64, error) {
	if explicit != nil {
		return *explicit, nil
	}
	n, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return 0, NewUpstream("eth_getTransactionCount", err)
	}
	return n, nil
}

// resolveGasPrice parses explicit hex when provided; otherwise fetches the
// endpoint's suggested gas price. Bad hex → INVALID_ARGS; RPC failure →
// UPSTREAM_ERROR.
func resolveGasPrice(ctx context.Context, client *remote.Client, explicit string) (*big.Int, error) {
	if explicit != "" {
		v, ok := new(big.Int).SetString(strings.TrimPrefix(explicit, "0x"), 16)
		if !ok {
			return nil, NewInvalidArgs(fmt.Sprintf("args.gas_price is not valid hex: %q", explicit))
		}
		return v, nil
	}
	gp, err := client.GasPrice(ctx)
	if err != nil {
		return nil, NewUpstream("eth_gasPrice", err)
	}
	return gp, nil
}

// resolveGas returns explicit when provided; otherwise calls eth_estimateGas
// with the synthesized CallMsg. RPC failure → UPSTREAM_ERROR.
func resolveGas(ctx context.Context, client *remote.Client, explicit *uint64, from, to common.Address, value *big.Int, data []byte) (uint64, error) {
	if explicit != nil {
		return *explicit, nil
	}
	msg := ethereum.CallMsg{From: from, To: &to, Value: value, Data: data}
	g, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, NewUpstream("eth_estimateGas", err)
	}
	return g, nil
}
