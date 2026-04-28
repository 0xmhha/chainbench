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

// newHandleNodeEventsGet returns a handler for node.events_get — a read-only
// wrapper around eth_getLogs with optional ABI-based log decoding. All filter
// fields (address, from_block, to_block, topics) are optional. When `abi` and
// `event` are both provided each returned log gets a `decoded` map alongside
// the raw fields, populated by abiutil.DecodeLog (topics for indexed args,
// log.Data for non-indexed). Without ABI, only raw log fields are returned.
//
// Block references accept the same forms as node.balance / node.contract_call
// (integer ≥0, "latest" / "pending" / "earliest"), plus a "0x..."-prefixed
// hex literal — eth_getLogs commonly takes hex blocks in the wild and we want
// callers to pass them through unchanged.
//
// Topics support per-position wildcard semantics: each entry can be a single
// topic hash (string), an OR-set of hashes ([string, string, ...]), or null
// (match anything in this position). Empty `topics` array → no topic filter.
//
// Args: {network?, node_id, address?, from_block?, to_block?, topics?, abi?, event?}
// Result: {logs: [{block_number, block_hash, tx_hash, tx_index, log_index,
//
//	address, topics, data, removed, decoded?}]}
//
// Error mapping:
//
//	INVALID_ARGS   — bad address, malformed topic hash, bad block ref,
//	                 unparseable ABI, missing event when abi present, event
//	                 not found in ABI
//	UPSTREAM_ERROR — state load / dial / eth_getLogs failure
//	INTERNAL       — abiutil.DecodeLog failure (caller's input was valid;
//	                 upstream returned data that does not match the ABI)
func newHandleNodeEventsGet(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network   string            `json:"network"`
			NodeID    string            `json:"node_id"`
			Address   string            `json:"address"`
			FromBlock json.RawMessage   `json:"from_block"`
			ToBlock   json.RawMessage   `json:"to_block"`
			Topics    []json.RawMessage `json:"topics"`
			ABI       string            `json:"abi"`
			Event     string            `json:"event"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}

		var query ethereum.FilterQuery

		if req.Address != "" {
			if !common.IsHexAddress(req.Address) {
				return nil, NewInvalidArgs(fmt.Sprintf("args.address is not a valid hex address: %q", req.Address))
			}
			query.Addresses = []common.Address{common.HexToAddress(req.Address)}
		}

		fromBlock, err := parseBlockArg(req.FromBlock, "from_block")
		if err != nil {
			return nil, err
		}
		toBlock, err := parseBlockArg(req.ToBlock, "to_block")
		if err != nil {
			return nil, err
		}
		query.FromBlock = fromBlock
		query.ToBlock = toBlock

		if len(req.Topics) > 0 {
			query.Topics = make([][]common.Hash, len(req.Topics))
			for i, t := range req.Topics {
				if len(t) == 0 || string(t) == "null" {
					query.Topics[i] = nil // wildcard at position i
					continue
				}
				// Try string first (single topic), then []string (OR-set). The
				// dual-form lets callers express "topic A" without nesting it
				// in a 1-element array, matching the wire convention used by
				// most JSON-RPC clients.
				var asStr string
				var asArr []string
				if err := json.Unmarshal(t, &asStr); err == nil {
					h, err := parseTopicHex(asStr)
					if err != nil {
						return nil, NewInvalidArgs(fmt.Sprintf("args.topics[%d]: %v", i, err))
					}
					query.Topics[i] = []common.Hash{h}
				} else if err := json.Unmarshal(t, &asArr); err == nil {
					hashes := make([]common.Hash, 0, len(asArr))
					for j, s := range asArr {
						h, err := parseTopicHex(s)
						if err != nil {
							return nil, NewInvalidArgs(fmt.Sprintf("args.topics[%d][%d]: %v", i, j, err))
						}
						hashes = append(hashes, h)
					}
					query.Topics[i] = hashes
				} else {
					return nil, NewInvalidArgs(fmt.Sprintf("args.topics[%d] must be string, [string], or null", i))
				}
			}
		}

		var parsedABI abi.ABI
		if req.ABI != "" {
			p, err := abiutil.ParseABI(req.ABI)
			if err != nil {
				return nil, NewInvalidArgs(err.Error())
			}
			parsedABI = p
			if req.Event == "" {
				return nil, NewInvalidArgs("args.event is required when args.abi is provided")
			}
			// Verify event exists in the ABI before any RPC round-trip — catches
			// typos cheaply and keeps the INVALID_ARGS / INTERNAL split clean
			// (decode failures after this point can only be remote-data issues).
			if _, ok := parsedABI.Events[req.Event]; !ok {
				return nil, NewInvalidArgs(fmt.Sprintf("event %q not in ABI", req.Event))
			}
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

		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			return nil, NewUpstream("eth_getLogs", err)
		}

		// Initialise the slice non-nil so an empty result marshals as `[]` not
		// `null` — clients shouldn't have to distinguish "no logs" from "field
		// missing". make(_, 0, len) keeps the allocation tight.
		outLogs := make([]map[string]any, 0, len(logs))
		for _, log := range logs {
			entry := map[string]any{
				"block_number": log.BlockNumber,
				"block_hash":   log.BlockHash.Hex(),
				"tx_hash":      log.TxHash.Hex(),
				"tx_index":     log.TxIndex,
				"log_index":    log.Index,
				"address":      log.Address.Hex(),
				"topics":       hashesToHex(log.Topics),
				"data":         "0x" + hex.EncodeToString(log.Data),
				"removed":      log.Removed,
			}
			if req.ABI != "" {
				decoded, derr := abiutil.DecodeLog(parsedABI, req.Event, log)
				if derr != nil {
					return nil, NewInternal(fmt.Sprintf("decode log[%d]", len(outLogs)), derr)
				}
				entry["decoded"] = map[string]any{
					"event": req.Event,
					"args":  decoded,
				}
			}
			outLogs = append(outLogs, entry)
		}
		return map[string]any{"logs": outLogs}, nil
	}
}

// parseBlockArg parses an optional block reference from a json.RawMessage.
// Accepts: integer N (≥0) → big.NewInt(N); "latest"/"pending" → nil (the
// canonical "current head" representation); "earliest" → big.NewInt(0); a
// "0x"-prefixed hex string → big.Int from hex; missing/empty → nil. The hex
// path is unique to events_get because eth_getLogs blocks are commonly hex
// literals in the JSON-RPC wire form, and round-tripping callers should be
// able to pass them through unchanged.
func parseBlockArg(raw json.RawMessage, fieldName string) (*big.Int, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var asInt int64
	var asStr string
	if err := json.Unmarshal(raw, &asInt); err == nil {
		if asInt < 0 {
			return nil, NewInvalidArgs(fmt.Sprintf("args.%s must be non-negative, got %d", fieldName, asInt))
		}
		return big.NewInt(asInt), nil
	}
	if err := json.Unmarshal(raw, &asStr); err == nil {
		switch asStr {
		case "latest", "pending":
			return nil, nil
		case "earliest":
			return big.NewInt(0), nil
		default:
			if strings.HasPrefix(asStr, "0x") {
				v, ok := new(big.Int).SetString(strings.TrimPrefix(asStr, "0x"), 16)
				if !ok {
					return nil, NewInvalidArgs(fmt.Sprintf("args.%s invalid hex: %q", fieldName, asStr))
				}
				return v, nil
			}
			return nil, NewInvalidArgs(fmt.Sprintf("args.%s label invalid: %q", fieldName, asStr))
		}
	}
	return nil, NewInvalidArgs(fmt.Sprintf("args.%s must be integer or block label string", fieldName))
}

// parseTopicHex parses a "0x" + 64-hex-char string into a common.Hash. Topic
// hashes are always 32 bytes; rejecting any other length here lets us
// surface bad input as INVALID_ARGS rather than letting it reach the upstream
// where the failure mode is less clear.
func parseTopicHex(s string) (common.Hash, error) {
	if !strings.HasPrefix(s, "0x") || len(s) != 66 {
		return common.Hash{}, fmt.Errorf("must be 0x + 64 hex chars: %q", s)
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(s, "0x")); err != nil {
		return common.Hash{}, fmt.Errorf("not valid hex: %q", s)
	}
	return common.HexToHash(s), nil
}

// hashesToHex converts a slice of common.Hash to canonical 0x-prefixed hex
// strings. Nil slice yields a nil slice; the JSON marshaller emits `[]` for
// empty topics regardless because the parent map field is never absent.
func hashesToHex(hashes []common.Hash) []string {
	if hashes == nil {
		return nil
	}
	out := make([]string, len(hashes))
	for i, h := range hashes {
		out[i] = h.Hex()
	}
	return out
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

		// block_number — int OR "latest" / "earliest" / "pending". Mirrors the
		// node.balance / node.contract_call parsing for shape parity. nil
		// signals "latest" to ethclient (interpreted as the head block);
		// big.NewInt(0) for earliest; integers passed as-is. blockLabel echoes
		// the caller-provided form back in the result so "pending" (silently
		// degraded to "latest" upstream) is observable.
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
