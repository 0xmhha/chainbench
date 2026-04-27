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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/signer"
	"github.com/0xmhha/chainbench/network/internal/state"
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

// feeDelegateTxType is the go-stablenet FeeDelegateDynamicFeeTx type byte.
// Distinct from any go-ethereum standard typed-tx prefix (0x01 / 0x02 / 0x03 /
// 0x04) so the broadcast envelope cannot be confused with a vanilla EIP-1559
// or SetCode transaction.
const feeDelegateTxType byte = 0x16

// feeDelegationAllowedChains is the chain-type allowlist for
// node.tx_fee_delegation_send. Hardcoded for Sprint 4c per spec §4.3 — promoting
// to an Adapter.SupportedTxTypes() method is a Sprint 5 concern. Anything not
// in this set returns NOT_SUPPORTED before any signer load or RPC round-trip.
var feeDelegationAllowedChains = map[string]bool{
	"stablenet": true,
	"wbft":      true,
}

// newHandleNodeTxFeeDelegationSend implements the go-stablenet
// FeeDelegateDynamicFeeTx (type 0x16). Two signers are required: the sender
// signs a standard DynamicFeeTx-shaped payload via SignTx; the fee_payer signs
// keccak256(0x16 || rlp([sender_inner_with_sig, fp_addr])) via SignHash. The
// final 0x16 envelope is broadcast through remote.Client.SendRawTransaction
// since ethclient.SendTransaction does not understand chain-specific typed
// envelopes.
//
// The handler does NOT auto-fill nonce / gas / fees: chain-specific testing
// intent demands explicit values per spec §4.3. Validation is strictly at the
// boundary — every structural error surfaces as INVALID_ARGS before any signer
// is loaded or any RPC round-trip occurs.
//
// Args:
//
//	{network, node_id, signer, fee_payer, to, value?, data?,
//	 max_fee_per_gas, max_priority_fee_per_gas, gas, nonce}
//
// Result: {tx_hash: "0x..." (computed locally as keccak256(rawTxBytes))}
//
// Error mapping (spec §11):
//
//	INVALID_ARGS   — missing args, malformed hex, bad to / value / data /
//	                 fee fields, signer.ErrInvalidAlias / ErrUnknownAlias for
//	                 either signer or fee_payer
//	NOT_SUPPORTED  — chain_type not in {stablenet, wbft}
//	UPSTREAM_ERROR — signer.ErrInvalidKey (config), state load failure,
//	                 dial / chainID / broadcast failure
//	INTERNAL       — RLP encode failure, SignTx / SignHash signature failure,
//	                 fp signature wrong length (invariant breach — caller
//	                 already validated structure)
func newHandleNodeTxFeeDelegationSend(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network              string  `json:"network"`
			NodeID               string  `json:"node_id"`
			Signer               string  `json:"signer"`
			FeePayer             string  `json:"fee_payer"`
			To                   string  `json:"to"`
			Value                string  `json:"value"`
			Data                 string  `json:"data"`
			Gas                  *uint64 `json:"gas"`
			MaxFeePerGas         string  `json:"max_fee_per_gas"`
			MaxPriorityFeePerGas string  `json:"max_priority_fee_per_gas"`
			Nonce                *uint64 `json:"nonce"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}

		// Required-field checks first — structural input issues never reach the
		// signer or the network. Order mirrors the spec §11 error matrix.
		if req.Signer == "" {
			return nil, NewInvalidArgs("args.signer is required")
		}
		if req.FeePayer == "" {
			return nil, NewInvalidArgs("args.fee_payer is required")
		}
		if req.To == "" {
			return nil, NewInvalidArgs("args.to is required")
		}
		if !common.IsHexAddress(req.To) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.to is not a valid hex address: %q", req.To))
		}
		to := common.HexToAddress(req.To)

		if req.MaxFeePerGas == "" || req.MaxPriorityFeePerGas == "" {
			return nil, NewInvalidArgs("args.max_fee_per_gas and args.max_priority_fee_per_gas are both required")
		}
		if req.Gas == nil {
			return nil, NewInvalidArgs("args.gas is required")
		}
		if req.Nonce == nil {
			return nil, NewInvalidArgs("args.nonce is required")
		}

		// Hex parsing — bad hex on any field surfaces as INVALID_ARGS before
		// any RPC round-trip.
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
		maxFee, ok := new(big.Int).SetString(strings.TrimPrefix(req.MaxFeePerGas, "0x"), 16)
		if !ok {
			return nil, NewInvalidArgs(fmt.Sprintf("args.max_fee_per_gas is not valid hex: %q", req.MaxFeePerGas))
		}
		maxPriorityFee, ok := new(big.Int).SetString(strings.TrimPrefix(req.MaxPriorityFeePerGas, "0x"), 16)
		if !ok {
			return nil, NewInvalidArgs(fmt.Sprintf("args.max_priority_fee_per_gas is not valid hex: %q", req.MaxPriorityFeePerGas))
		}

		// Resolve node + look up the network's chain_type. resolveNode hides the
		// state-layer details but doesn't surface chain_type, so we re-load via
		// state.LoadActive — same call resolveNode makes internally. Any state
		// failure surfaces as UPSTREAM_ERROR per spec §11.
		_, node, err := resolveNode(stateDir, req.Network, req.NodeID)
		if err != nil {
			return nil, err
		}
		networkName := req.Network
		if networkName == "" {
			networkName = "local"
		}
		netState, lerr := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: networkName})
		if lerr != nil {
			return nil, NewUpstream("failed to load network state", lerr)
		}
		chainType := string(netState.ChainType)
		if !feeDelegationAllowedChains[chainType] {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.tx_fee_delegation_send is only supported on stablenet/wbft chains; got chain_type=%q", chainType,
			))
		}

		// Two-signer load. Both keys live in their own *sealed struct; neither
		// touches the other's state, and the redaction boundary is unchanged.
		s, serr := signer.Load(signer.Alias(req.Signer))
		if serr != nil {
			if errors.Is(serr, signer.ErrInvalidAlias) || errors.Is(serr, signer.ErrUnknownAlias) {
				return nil, NewInvalidArgs(serr.Error())
			}
			return nil, NewUpstream("signer load", serr)
		}
		fp, ferr := signer.Load(signer.Alias(req.FeePayer))
		if ferr != nil {
			if errors.Is(ferr, signer.ErrInvalidAlias) || errors.Is(ferr, signer.ErrUnknownAlias) {
				return nil, NewInvalidArgs(ferr.Error())
			}
			return nil, NewUpstream("fee_payer signer load", ferr)
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

		// Sender signs a standard DynamicFeeTx — go-stablenet's
		// FeeDelegateDynamicFeeTx.SenderTx uses an identical sigHash, so SignTx
		// against a DynamicFeeTx produces the exact V/R/S the inner_with_sig RLP
		// list expects.
		innerTx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			ChainID:    chainID,
			Nonce:      *req.Nonce,
			GasTipCap:  maxPriorityFee,
			GasFeeCap:  maxFee,
			Gas:        *req.Gas,
			To:         &to,
			Value:      value,
			Data:       data,
			AccessList: nil,
		})
		signedInner, err := s.SignTx(ctx, innerTx, chainID)
		if err != nil {
			return nil, NewInternal("sign sender tx", err)
		}
		senderV, senderR, senderS := signedInner.RawSignatureValues()

		// Build inner_with_sig per Python reference fee_delegate.py:
		//   [chainID, nonce, tipCap, feeCap, gas, to_bytes, value, data,
		//    [], senderV, senderR, senderS]
		// RLP rules: *big.Int → canonical big-endian byte string (zero-stripped);
		// []byte → byte string of its raw length; [] → empty list. Mixed-type
		// lists are encoded as []interface{}.
		fpAddrBytes := fp.Address().Bytes()
		innerWithSig := []interface{}{
			chainID,
			new(big.Int).SetUint64(*req.Nonce),
			maxPriorityFee,
			maxFee,
			new(big.Int).SetUint64(*req.Gas),
			to.Bytes(),
			value,
			data,
			[]interface{}{}, // access list
			senderV,
			senderR,
			senderS,
		}
		fpPayload := []interface{}{innerWithSig, fpAddrBytes}
		fpRLP, rerr := rlp.EncodeToBytes(fpPayload)
		if rerr != nil {
			return nil, NewInternal("rlp encode fee-payer payload", rerr)
		}
		fpSigHash := crypto.Keccak256(append([]byte{feeDelegateTxType}, fpRLP...))

		// Fee payer signs the outer hash. SignHash returns 65 bytes laid out as
		// R(0..32) || S(32..64) || V(64). V here is the raw recovery id (0 or 1)
		// — the Python reference uses keys.PrivateKey.sign_msg_hash which produces
		// the same shape, so the wire tx accepts these values directly.
		fpSig, sigErr := fp.SignHash(ctx, common.BytesToHash(fpSigHash))
		if sigErr != nil {
			return nil, NewInternal("sign fee payer", sigErr)
		}
		if len(fpSig) != 65 {
			return nil, NewInternal(fmt.Sprintf("fee_payer signature wrong length: %d", len(fpSig)), nil)
		}
		fpR := new(big.Int).SetBytes(fpSig[0:32])
		fpS := new(big.Int).SetBytes(fpSig[32:64])
		fpV := big.NewInt(int64(fpSig[64]))

		// Final RLP envelope:
		//   0x16 || rlp([inner_with_sig, fp_addr, fpV, fpR, fpS])
		finalPayload := []interface{}{innerWithSig, fpAddrBytes, fpV, fpR, fpS}
		finalRLP, rerr := rlp.EncodeToBytes(finalPayload)
		if rerr != nil {
			return nil, NewInternal("rlp encode final payload", rerr)
		}
		rawTx := append([]byte{feeDelegateTxType}, finalRLP...)

		// tx_hash is computed locally as keccak256(rawTx). The endpoint's echo of
		// eth_sendRawTransaction is intentionally discarded — see
		// remote.Client.SendRawTransaction docs for rationale.
		txHash := crypto.Keccak256(rawTx)

		if err := client.SendRawTransaction(ctx, rawTx); err != nil {
			return nil, NewUpstream("eth_sendRawTransaction", err)
		}
		return map[string]any{"tx_hash": "0x" + hex.EncodeToString(txHash)}, nil
	}
}
