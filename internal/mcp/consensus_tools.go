package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// consensusStatusTool reports a one-shot consensus snapshot: head, chain id,
// peer count, sync state, and validator-set size. Backed by the core rpc client
// and the chain's manifest validators method.
func consensusStatusTool() Tool {
	return Tool{
		Name:        "chainbench_consensus_status",
		Description: "Consensus snapshot: head block, chain id, peers, syncing, validator count. Args: chain, rpc.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain": map[string]any{"type": "string"},
				"rpc":   map[string]any{"type": "string"},
			},
			"required": []string{"chain", "rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			chain := argString(args, "chain", "stablenet")
			rpcURL := argString(args, "rpc", "")
			if rpcURL == "" {
				return "", fmt.Errorf("rpc is required")
			}
			p, err := registry.Get(chain)
			if err != nil {
				return "", err
			}
			cli := rpc.Dial(rpcURL)
			head, err := cli.BlockNumber(ctx)
			if err != nil {
				return "", err
			}
			cid, _ := cli.ChainID(ctx)
			peers, _ := cli.PeerCount(ctx)
			syncing, _ := cli.Syncing(ctx)
			vals, _ := registry.Validators(ctx, cli, p.Manifest().Consensus.ValidatorsMethod)
			return fmt.Sprintf("chain=%s head=%d chain_id=%d peers=%d syncing=%t validators=%d",
				chain, head, cid, peers, syncing, len(vals)), nil
		},
	}
}

// consensusHealthTool reports a quick health verdict for an endpoint: healthy
// when it is not syncing and has advanced past genesis. Instant (no sampling
// window); use verify for produce-rate confirmation.
func consensusHealthTool() Tool {
	return Tool{
		Name:        "chainbench_consensus_health",
		Description: "Quick consensus health verdict (not syncing + past genesis). Args: rpc.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"rpc": map[string]any{"type": "string"}},
			"required":   []string{"rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rpcURL := argString(args, "rpc", "")
			if rpcURL == "" {
				return "", fmt.Errorf("rpc is required")
			}
			cli := rpc.Dial(rpcURL)
			head, err := cli.BlockNumber(ctx)
			if err != nil {
				return "", err
			}
			syncing, _ := cli.Syncing(ctx)
			peers, _ := cli.PeerCount(ctx)
			verdict := "healthy"
			if syncing || head == 0 {
				verdict = "unhealthy"
			}
			return fmt.Sprintf("%s: head=%d syncing=%t peers=%d", verdict, head, syncing, peers), nil
		},
	}
}

// consensusBlockInfoTool reports a block's header fields. Args: rpc, optional
// block (number hex/decimal or "latest"; default "latest").
func consensusBlockInfoTool() Tool {
	return Tool{
		Name:        "chainbench_consensus_block_info",
		Description: "Block header info (number, hash, miner, timestamp, tx count, gas). Args: rpc, optional block.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rpc":   map[string]any{"type": "string"},
				"block": map[string]any{"type": "string"},
			},
			"required": []string{"rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rpcURL := argString(args, "rpc", "")
			if rpcURL == "" {
				return "", fmt.Errorf("rpc is required")
			}
			block := normalizeBlockTag(argString(args, "block", "latest"))
			var blk struct {
				Number       string   `json:"number"`
				Hash         string   `json:"hash"`
				Miner        string   `json:"miner"`
				Timestamp    string   `json:"timestamp"`
				GasUsed      string   `json:"gasUsed"`
				Transactions []string `json:"transactions"`
			}
			if err := rpc.Dial(rpcURL).Call(ctx, "eth_getBlockByNumber", &blk, block, false); err != nil {
				return "", err
			}
			if blk.Number == "" {
				return "", fmt.Errorf("no block %q", block)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "number=%d hash=%s miner=%s txs=%d",
				hexToU64(blk.Number), blk.Hash, blk.Miner, len(blk.Transactions))
			if blk.Timestamp != "" {
				fmt.Fprintf(&b, " timestamp=%d", hexToU64(blk.Timestamp))
			}
			if blk.GasUsed != "" {
				fmt.Fprintf(&b, " gas_used=%d", hexToU64(blk.GasUsed))
			}
			return b.String(), nil
		},
	}
}

// normalizeBlockTag passes through "latest"/"pending"/"earliest" and 0x-prefixed
// hex, and converts a bare decimal to hex (what eth_getBlockByNumber expects).
func normalizeBlockTag(s string) string {
	switch s {
	case "", "latest":
		return "latest"
	case "pending", "earliest":
		return s
	}
	if strings.HasPrefix(s, "0x") {
		return s
	}
	if n, err := strconv.ParseUint(s, 10, 64); err == nil {
		return fmt.Sprintf("0x%x", n)
	}
	return s
}

func hexToU64(s string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	return n
}
