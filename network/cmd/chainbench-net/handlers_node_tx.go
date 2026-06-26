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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/0xmhha/chainbench/network/internal/abiutil"
	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/signer"
)

// authEntry mirrors a single element of node.tx_send args.authorization_list.
// Each tuple references a separate signer alias so that EIP-7702
// authorizations can compose across distinct keys without sharing state.
type authEntry struct {
	ChainID string `json:"chain_id"`
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
	Signer  string `json:"signer"`
}

// newHandleNodeTxSend signs + broadcasts a transaction against the resolved
// node. Fee mode is selected from args: gas_price → legacy (type 0);
// max_fee_per_gas + max_priority_fee_per_gas → EIP-1559 (type 2);
// authorization_list (non-empty) plus 1559 fields → EIP-7702 SetCode (type 4).
// Mixed or partial 1559 args, and authorization_list combined with legacy or
// partial 1559, are rejected as INVALID_ARGS. Missing nonce / gas /
// gas_price (legacy only) are auto-filled via the remote.Client. Signer alias
// is resolved per-request via signer.Load. SetCode authorizations resolve a
// distinct signer alias per entry and sign the EIP-7702 SigHash via SignHash.
//
// Args: {network?, node_id, signer, to, value?, data?, gas?,
//
//	gas_price?, max_fee_per_gas?, max_priority_fee_per_gas?, nonce?,
//	authorization_list?: [{chain_id, address, nonce, signer}]}
//
// Result: {tx_hash: "0x..."}
//
// Error mapping (spec §11 + §3.1 + §9.2):
//
//	INVALID_ARGS   — malformed args, missing signer / to, bad to / value / data /
//	                 gas_price / max_fee / max_priority_fee hex, mixed legacy + 1559
//	                 fee fields, partial 1559 fee fields, signer.ErrInvalidAlias,
//	                 signer.ErrUnknownAlias, authorization_list combined with legacy
//	                 or partial 1559 fields, missing / malformed authorization entry
//	                 fields, unknown authorization signer alias
//	UPSTREAM_ERROR — signer.ErrInvalidKey (config failure), dial failure, chainID /
//	                 nonce / gas / gas_price fetch failure, SendTransaction failure
//	INTERNAL       — SignTx failure (invariant breach — inputs validated earlier),
//	                 authorization SignHash failure
func newHandleNodeTxSend(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network              string      `json:"network"`
			NodeID               string      `json:"node_id"`
			Signer               string      `json:"signer"`
			To                   string      `json:"to"`
			Value                string      `json:"value"`
			Data                 string      `json:"data"`
			Gas                  *uint64     `json:"gas"`
			GasPrice             string      `json:"gas_price"`
			MaxFeePerGas         string      `json:"max_fee_per_gas"`
			MaxPriorityFeePerGas string      `json:"max_priority_fee_per_gas"`
			Nonce                *uint64     `json:"nonce"`
			AuthorizationList    []authEntry `json:"authorization_list"`
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
		useDynamicFee, maxFee, maxPriorityFee, err := selectFeeMode(req.GasPrice, req.MaxFeePerGas, req.MaxPriorityFeePerGas)
		if err != nil {
			return nil, err
		}

		// SetCode (EIP-7702) selector: presence of a non-empty
		// authorization_list upgrades the fee mode from 2-way to 3-way. An
		// empty slice ([]) intentionally falls through to the DynamicFee
		// path so callers cannot accidentally upgrade by sending an empty
		// list. SetCode requires both 1559 fields — mixing with gas_price
		// or omitting either tip field is rejected at the boundary.
		useSetCode := false
		if len(req.AuthorizationList) > 0 {
			if !useDynamicFee {
				return nil, NewInvalidArgs("authorization_list requires both max_fee_per_gas and max_priority_fee_per_gas")
			}
			useSetCode = true
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

		ctx, cancel := context.WithTimeout(context.Background(), nodeWriteTimeout)
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
		switch {
		case useSetCode:
			auths, aerr := buildAuthorizations(ctx, req.AuthorizationList)
			if aerr != nil {
				return nil, aerr
			}
			unsigned = ethtypes.NewTx(&ethtypes.SetCodeTx{
				ChainID:   uint256.MustFromBig(chainID),
				Nonce:     nonce,
				GasTipCap: uint256.MustFromBig(maxPriorityFee),
				GasFeeCap: uint256.MustFromBig(maxFee),
				Gas:       gas,
				To:        to,
				Value:     uint256.MustFromBig(value),
				Data:      data,
				AuthList:  auths,
			})
		case useDynamicFee:
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
		default:
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

// newHandleNodeContractDeploy signs + broadcasts a contract-creation tx
// (to: nil) using the same fee-mode selector as node.tx_send minus the
// SetCode (EIP-7702) path. SetCode is unavailable here because the
// authorization_list designates an authority address that is paired with the
// tx's `to` field — contract creation has no `to`, so the combination is
// meaningless. Callers who pass authorization_list are rejected at the
// boundary before any signer load or RPC.
//
// Args: {network?, node_id, signer, bytecode, abi?, constructor_args?,
//
//	value?, gas?, gas_price?, max_fee_per_gas?, max_priority_fee_per_gas?,
//	nonce?}
//
// Result: {tx_hash: "0x...", contract_address: "0x..."}
//
// `contract_address` is computed locally as crypto.CreateAddress(senderAddr,
// nonce). Callers should follow up with node.tx_wait to confirm the receipt
// and verify the runtime code at the address.
//
// Error mapping (spec §4.2 + §11):
//
//	INVALID_ARGS   — missing signer / bytecode, malformed bytecode hex,
//	                 malformed value / gas_price / max_fee / max_priority_fee
//	                 hex, mixed legacy + 1559 fee fields, partial 1559 fee
//	                 fields, ABI parse failure, ABI arg-count mismatch,
//	                 constructor_args without abi, authorization_list provided,
//	                 signer.ErrInvalidAlias, signer.ErrUnknownAlias
//	UPSTREAM_ERROR — signer.ErrInvalidKey (config), dial / chainID / nonce /
//	                 gas / gas_price / broadcast failure
//	INTERNAL       — SignTx failure (invariant breach — inputs validated earlier)
func newHandleNodeContractDeploy(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network              string          `json:"network"`
			NodeID               string          `json:"node_id"`
			Signer               string          `json:"signer"`
			Bytecode             string          `json:"bytecode"`
			ABI                  string          `json:"abi"`
			ConstructorArgs      []any           `json:"constructor_args"`
			Value                string          `json:"value"`
			Gas                  *uint64         `json:"gas"`
			GasPrice             string          `json:"gas_price"`
			MaxFeePerGas         string          `json:"max_fee_per_gas"`
			MaxPriorityFeePerGas string          `json:"max_priority_fee_per_gas"`
			Nonce                *uint64         `json:"nonce"`
			AuthorizationList    json.RawMessage `json:"authorization_list"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Signer == "" {
			return nil, NewInvalidArgs("args.signer is required")
		}
		if req.Bytecode == "" {
			return nil, NewInvalidArgs("args.bytecode is required")
		}
		// authorization_list paired with deploy is meaningless — SetCode
		// authorizes an address that pairs with the tx's `to`, but a
		// contract-creation tx has no `to`. Empty list / null are accepted as
		// equivalent to absence.
		if len(req.AuthorizationList) > 0 && string(req.AuthorizationList) != "null" {
			// Probe the array length so an empty `[]` is not rejected. Anything
			// else (including non-array shapes) surfaces as the same boundary
			// error — callers must not pass this field for deploy.
			var probe []json.RawMessage
			if err := json.Unmarshal(req.AuthorizationList, &probe); err == nil && len(probe) == 0 {
				// empty array — fall through
			} else {
				return nil, NewInvalidArgs("authorization_list is not supported for node.contract_deploy (SetCode does not pair with contract creation)")
			}
		}

		// Hex-decode bytecode at the boundary. A leading "0x" is canonical but
		// optional — callers occasionally strip it when piping output.
		bytecodeBytes, err := hex.DecodeString(strings.TrimPrefix(req.Bytecode, "0x"))
		if err != nil {
			return nil, NewInvalidArgs(fmt.Sprintf("args.bytecode is not valid hex: %v", err))
		}

		value := big.NewInt(0)
		if req.Value != "" {
			parsed, ok := new(big.Int).SetString(strings.TrimPrefix(req.Value, "0x"), 16)
			if !ok {
				return nil, NewInvalidArgs(fmt.Sprintf("args.value is not valid hex: %q", req.Value))
			}
			value = parsed
		}

		// Fee-mode selector: legacy (gas_price) vs EIP-1559 (max_fee + tip).
		// SetCode (4c) has no meaning for contract creation and is excluded.
		useDynamicFee, maxFee, maxPriorityFee, err := selectFeeMode(req.GasPrice, req.MaxFeePerGas, req.MaxPriorityFeePerGas)
		if err != nil {
			return nil, err
		}

		// Build the calldata: bytecode || encoded(constructor_args). When the
		// caller omits abi we trust they have already encoded args into
		// bytecode, so we broadcast as-is. constructor_args without abi has no
		// well-defined encoding so we reject explicitly rather than silently
		// drop them.
		data := bytecodeBytes
		if req.ABI != "" {
			parsed, perr := abiutil.ParseABI(req.ABI)
			if perr != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.abi parse failed: %v", perr))
			}
			encoded, eerr := abiutil.PackConstructor(parsed, req.ConstructorArgs)
			if eerr != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.constructor_args: %v", eerr))
			}
			data = append(append([]byte{}, bytecodeBytes...), encoded...)
		} else if len(req.ConstructorArgs) > 0 {
			return nil, NewInvalidArgs("args.constructor_args provided without args.abi has no defined encoding")
		}

		// Signer resolution per-request. Same classification as tx_send.
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

		ctx, cancel := context.WithTimeout(context.Background(), nodeWriteTimeout)
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

		// Gas estimation for contract creation needs To: nil — the existing
		// resolveGas helper passes a non-nil *common.Address, which would make
		// the upstream treat this as a regular call. Inline the small dance
		// here rather than threading a pointer through the helper.
		var gas uint64
		if req.Gas != nil {
			gas = *req.Gas
		} else {
			msg := ethereum.CallMsg{From: s.Address(), Value: value, Data: data}
			estGas, eerr := client.EstimateGas(ctx, msg)
			if eerr != nil {
				return nil, NewUpstream("eth_estimateGas", eerr)
			}
			gas = estGas
		}

		var unsigned *ethtypes.Transaction
		if useDynamicFee {
			unsigned = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				ChainID:   chainID,
				Nonce:     nonce,
				GasTipCap: maxPriorityFee,
				GasFeeCap: maxFee,
				Gas:       gas,
				To:        nil, // contract creation
				Value:     value,
				Data:      data,
			})
		} else {
			gasPrice, gerr := resolveGasPrice(ctx, client, req.GasPrice)
			if gerr != nil {
				return nil, gerr
			}
			unsigned = ethtypes.NewTx(&ethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: gasPrice,
				Gas:      gas,
				To:       nil, // contract creation
				Value:    value,
				Data:     data,
			})
		}

		signed, serr := s.SignTx(ctx, unsigned, chainID)
		if serr != nil {
			return nil, NewInternal("sign tx", serr)
		}
		if err := client.SendTransaction(ctx, signed); err != nil {
			return nil, NewUpstream("eth_sendRawTransaction", err)
		}

		// Deterministic deployment address — recovered locally so the caller
		// can poll node.tx_wait without re-deriving it.
		contractAddr := crypto.CreateAddress(s.Address(), nonce)
		return map[string]any{
			"tx_hash":          signed.Hash().Hex(),
			"contract_address": contractAddr.Hex(),
		}, nil
	}
}

// buildAuthorizations resolves and signs each EIP-7702 authorization tuple.
// Validation order is required-fields → address shape → chain_id hex → nonce
// hex → signer.Load → SignHash → V/R/S decompose so that structural input
// errors surface as INVALID_ARGS before any signing call. Each entry's signer
// alias is independent — distinct aliases produce distinct signatures, and
// signer state never crosses entries.
func buildAuthorizations(ctx context.Context, entries []authEntry) ([]ethtypes.SetCodeAuthorization, error) {
	out := make([]ethtypes.SetCodeAuthorization, 0, len(entries))
	for i, e := range entries {
		if e.Signer == "" || e.Address == "" || e.ChainID == "" || e.Nonce == "" {
			return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d]: chain_id/address/nonce/signer are all required", i))
		}
		if !common.IsHexAddress(e.Address) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].address invalid: %q", i, e.Address))
		}
		cid, ok := new(big.Int).SetString(strings.TrimPrefix(e.ChainID, "0x"), 16)
		if !ok {
			return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].chain_id invalid: %q", i, e.ChainID))
		}
		nonce, ok := new(big.Int).SetString(strings.TrimPrefix(e.Nonce, "0x"), 16)
		if !ok {
			return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].nonce invalid: %q", i, e.Nonce))
		}
		s, serr := signer.Load(signer.Alias(e.Signer))
		if serr != nil {
			if errors.Is(serr, signer.ErrInvalidAlias) || errors.Is(serr, signer.ErrUnknownAlias) {
				return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].signer: %v", i, serr))
			}
			return nil, NewUpstream(fmt.Sprintf("authorization_list[%d] signer load", i), serr)
		}
		auth := ethtypes.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(cid),
			Address: common.HexToAddress(e.Address),
			Nonce:   nonce.Uint64(),
		}
		sig, sigErr := s.SignHash(ctx, auth.SigHash())
		if sigErr != nil {
			return nil, NewInternal(fmt.Sprintf("sign authorization[%d]", i), sigErr)
		}
		if len(sig) != 65 {
			return nil, NewInternal(fmt.Sprintf("authorization[%d] signature wrong length", i), nil)
		}
		auth.R = *uint256.NewInt(0).SetBytes(sig[0:32])
		auth.S = *uint256.NewInt(0).SetBytes(sig[32:64])
		auth.V = sig[64]
		out = append(out, auth)
	}
	return out, nil
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

// selectFeeMode validates the legacy (gas_price) vs EIP-1559 (max_fee + tip)
// fee fields and, on the dynamic-fee path, parses the hex max-fee / priority-fee
// values. It returns useDynamicFee plus the parsed fees (both nil in legacy
// mode). Mixed (legacy + 1559) or partial 1559 (only one of the two fields)
// combinations are rejected as INVALID_ARGS so structural mistakes never reach
// the signer. Shared by node.tx_send and node.contract_deploy.
func selectFeeMode(gasPrice, maxFeePerGas, maxPriorityFeePerGas string) (useDynamicFee bool, maxFee, maxPriorityFee *big.Int, err error) {
	hasLegacy := gasPrice != ""
	hasMaxFee := maxFeePerGas != ""
	hasTip := maxPriorityFeePerGas != ""
	if hasLegacy && (hasMaxFee || hasTip) {
		return false, nil, nil, NewInvalidArgs("args.gas_price cannot be combined with args.max_fee_per_gas or args.max_priority_fee_per_gas")
	}
	if hasMaxFee != hasTip {
		return false, nil, nil, NewInvalidArgs("args.max_fee_per_gas and args.max_priority_fee_per_gas must both be provided when using EIP-1559")
	}
	useDynamicFee = hasMaxFee && hasTip
	if useDynamicFee {
		var ok bool
		maxFee, ok = new(big.Int).SetString(strings.TrimPrefix(maxFeePerGas, "0x"), 16)
		if !ok {
			return false, nil, nil, NewInvalidArgs(fmt.Sprintf("args.max_fee_per_gas is not valid hex: %q", maxFeePerGas))
		}
		maxPriorityFee, ok = new(big.Int).SetString(strings.TrimPrefix(maxPriorityFeePerGas, "0x"), 16)
		if !ok {
			return false, nil, nil, NewInvalidArgs(fmt.Sprintf("args.max_priority_fee_per_gas is not valid hex: %q", maxPriorityFeePerGas))
		}
	}
	return useDynamicFee, maxFee, maxPriorityFee, nil
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

const (
	minTxWaitMs     = 1000
	defaultTxWaitMs = 60000
	maxTxWaitMs     = 600000
	txWaitInitial   = 200 * time.Millisecond
	txWaitCap       = 2 * time.Second
)

// newHandleNodeTxWait polls eth_getTransactionReceipt with bounded backoff
// until the receipt is available, the context deadline elapses, or an
// upstream error surfaces. ethereum.NotFound is the polling-tick signal
// (still pending); other RPC errors are upstream failures.
//
// Args:    {network?, node_id, tx_hash, timeout_ms? (1000..600000, default 60000)}
// Result:  {status: "success"|"failed"|"pending", tx_hash, block_number?,
//
//	block_hash?, gas_used?, effective_gas_price?, contract_address?,
//	logs_count?}
//
// On terminal "pending" (timeout), only {status, tx_hash} are returned —
// the caller decides whether to retry.
func newHandleNodeTxWait(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network   string `json:"network"`
			NodeID    string `json:"node_id"`
			TxHash    string `json:"tx_hash"`
			TimeoutMs *int   `json:"timeout_ms"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.TxHash == "" {
			return nil, NewInvalidArgs("args.tx_hash is required")
		}
		// 0x + 64 lowercase/uppercase hex chars
		if len(req.TxHash) != 66 || !strings.HasPrefix(req.TxHash, "0x") {
			return nil, NewInvalidArgs(fmt.Sprintf("args.tx_hash must be 0x + 32-byte hex: %q", req.TxHash))
		}
		if _, err := hex.DecodeString(strings.TrimPrefix(req.TxHash, "0x")); err != nil {
			return nil, NewInvalidArgs(fmt.Sprintf("args.tx_hash is not valid hex: %q", req.TxHash))
		}
		timeoutMs := defaultTxWaitMs
		if req.TimeoutMs != nil {
			timeoutMs = *req.TimeoutMs
		}
		if timeoutMs < minTxWaitMs || timeoutMs > maxTxWaitMs {
			return nil, NewInvalidArgs(fmt.Sprintf(
				"args.timeout_ms must be %d..%d, got %d",
				minTxWaitMs, maxTxWaitMs, timeoutMs,
			))
		}

		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
		client, err := dialNode(ctx, &node)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		hash := common.HexToHash(req.TxHash)
		delay := txWaitInitial
		for {
			rcpt, rerr := client.TransactionReceipt(ctx, hash)
			if rerr == nil {
				return receiptToResult(req.TxHash, rcpt), nil
			}
			if !errors.Is(rerr, ethereum.NotFound) {
				return nil, NewUpstream("eth_getTransactionReceipt", rerr)
			}
			// NotFound: backoff or fall through to "pending" on deadline.
			select {
			case <-ctx.Done():
				return map[string]any{"status": "pending", "tx_hash": req.TxHash}, nil
			case <-time.After(delay):
			}
			delay *= 2
			if delay > txWaitCap {
				delay = txWaitCap
			}
		}
	}
}

func receiptToResult(txHash string, rcpt *ethtypes.Receipt) map[string]any {
	status := "failed"
	if rcpt.Status == 1 {
		status = "success"
	}
	out := map[string]any{
		"status":              status,
		"tx_hash":             txHash,
		"block_number":        rcpt.BlockNumber.Uint64(),
		"block_hash":          rcpt.BlockHash.Hex(),
		"gas_used":            rcpt.GasUsed,
		"logs_count":          len(rcpt.Logs),
		"contract_address":    "",
		"effective_gas_price": "0x0",
	}
	if rcpt.ContractAddress != (common.Address{}) {
		out["contract_address"] = rcpt.ContractAddress.Hex()
	}
	if rcpt.EffectiveGasPrice != nil {
		out["effective_gas_price"] = "0x" + rcpt.EffectiveGasPrice.Text(16)
	}
	return out
}
