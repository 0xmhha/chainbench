package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xmhha/chainbench/network/internal/adapters"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/signer"
	"github.com/0xmhha/chainbench/network/internal/state"
)

// feeDelegateTxType is the go-stablenet FeeDelegateDynamicFeeTx type byte.
// Distinct from any go-ethereum standard typed-tx prefix (0x01 / 0x02 / 0x03 /
// 0x04) so the broadcast envelope cannot be confused with a vanilla EIP-1559
// or SetCode transaction. Sourced from the adapter contract so the gate below
// and the adapters' SupportedTxTypes share one definition.
const feeDelegateTxType = adapters.FeeDelegateDynamicFeeTxType

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
		adapter, aerr := adapters.Load(chainType)
		if aerr != nil || !slices.Contains(adapter.SupportedTxTypes(), feeDelegateTxType) {
			return nil, NewNotSupported(fmt.Sprintf(
				"node.tx_fee_delegation_send is not supported on chain_type=%q", chainType,
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
