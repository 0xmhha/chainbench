package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xmhha/chainbench/network/internal/abiutil"
	"github.com/0xmhha/chainbench/network/internal/events"
)

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
		ctx, cancel := context.WithTimeout(context.Background(), nodeReadTimeout)
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
