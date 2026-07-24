package mcp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/consensus"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/verify"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

// Default returns a Server with the built-in chainbench tools registered. Chain
// and test-case plugins must be imported by the binary for registration.
func Default(name, version string) *Server {
	s := NewServer(name, version)
	s.Register(chainsTool())
	s.Register(faucetTool())
	s.Register(verifyTool())
	s.Register(testTool())
	s.Register(consensusTool())
	s.Register(nodeRPCTool())
	return s
}

func nodeRPCTool() Tool {
	return Tool{
		Name:        "chainbench_node_rpc",
		Description: "Call an arbitrary JSON-RPC method on a node and return the raw result. Args: rpc, method, params (JSON array).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rpc":    map[string]any{"type": "string"},
				"method": map[string]any{"type": "string"},
				"params": map[string]any{"type": "array"},
			},
			"required": []string{"rpc", "method"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			url := argString(args, "rpc", "")
			method := argString(args, "method", "")
			if url == "" || method == "" {
				return "", fmt.Errorf("rpc and method are required")
			}
			var params []any
			if p, ok := args["params"].([]any); ok {
				params = p
			}
			var raw json.RawMessage
			if err := rpc.Dial(url).Call(ctx, method, &raw, params...); err != nil {
				return "", err
			}
			return string(raw), nil
		},
	}
}

func consensusTool() Tool {
	return Tool{
		Name:        "chainbench_consensus",
		Description: "List the validator/producer set via the chain's consensus RPC method. Args: chain, rpc.",
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
			p, err := registry.Get(chain)
			if err != nil {
				return "", err
			}
			method := p.Manifest().Consensus.ValidatorsMethod
			vals, err := consensus.Validators(ctx, rpc.Dial(argString(args, "rpc", "")), method)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "validators (%s via %s): %d\n", chain, method, len(vals))
			for i, v := range vals {
				fmt.Fprintf(&b, "  %d. %s\n", i+1, v)
			}
			return b.String(), nil
		},
	}
}

func chainsTool() Tool {
	return Tool{
		Name:        "chainbench_chains",
		Description: "List the chains chainbench supports (id, consensus family, binary, chain id, RPC namespace).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			var b strings.Builder
			for _, id := range registry.Names() {
				p, err := registry.Get(id)
				if err != nil {
					return "", err
				}
				m := p.Manifest()
				fmt.Fprintf(&b, "%s\tfamily=%s\tbinary=%s\tchain_id=%d\tnamespace=%s\n",
					m.ID, m.ConsensusFamily, m.Binary, m.ChainID, m.Consensus.RPCNamespace)
			}
			return b.String(), nil
		},
	}
}

func faucetTool() Tool {
	return Tool{
		Name:        "chainbench_faucet",
		Description: "Send funds from a genesis-allocated key to an account. Args: chain, rpc, from_key (hex), to (0x-addr), amount (wei).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":    map[string]any{"type": "string"},
				"rpc":      map[string]any{"type": "string"},
				"from_key": map[string]any{"type": "string"},
				"to":       map[string]any{"type": "string"},
				"amount":   map[string]any{"type": "string"},
			},
			"required": []string{"rpc", "from_key", "to", "amount"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			ap, err := accounts.ForChain(argString(args, "chain", "stablenet"))
			if err != nil {
				return "", err
			}
			key, err := hex.DecodeString(strings.TrimPrefix(argString(args, "from_key", ""), "0x"))
			if err != nil {
				return "", fmt.Errorf("bad from_key: %w", err)
			}
			amt, ok := new(big.Int).SetString(argString(args, "amount", ""), 10)
			if !ok {
				return "", fmt.Errorf("bad amount (decimal wei expected)")
			}
			hash, err := ap.Faucet(ctx, key, argString(args, "to", ""), amt, argString(args, "rpc", ""))
			if err != nil {
				return "", err
			}
			return "tx: " + hash, nil
		},
	}
}

func verifyTool() Tool {
	return Tool{
		Name:        "chainbench_verify",
		Description: "Verify an existing network is producing blocks and report node info. Args: chain, rpc (string or array of RPC URLs).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain": map[string]any{"type": "string"},
				"rpc":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			ns, err := nodeSetFromArgs(args)
			if err != nil {
				return "", err
			}
			rep, err := verify.Run(ctx, ns, verify.Options{}, nil)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "producing: %v\n", rep.Producing)
			for _, n := range rep.Nodes {
				fmt.Fprintf(&b, "node%d %s chain_id=%d block=%d peers=%d ok=%v\n",
					n.Index, n.RPCURL, n.ChainID, n.BlockNumber, n.PeerCount, n.OK)
			}
			return b.String(), nil
		},
	}
}

func testTool() Tool {
	return Tool{
		Name:        "chainbench_test",
		Description: "Run test cases against a network. Args: chain, rpc (array) or data_dir, optional name/category filters.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":    map[string]any{"type": "string"},
				"rpc":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"data_dir": map[string]any{"type": "string"},
				"name":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"category": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			ns, err := nodeSetFromArgs(args)
			if err != nil {
				return "", err
			}
			rep, err := testrun.Run(ctx, ns, testrun.Options{
				Names:      argStrings(args, "name"),
				Categories: argStrings(args, "category"),
			})
			if err != nil {
				return "", err
			}
			var b strings.Builder
			for _, r := range rep.Results {
				fmt.Fprintf(&b, "%s [%s] %s %s\n", r.Name, r.Category, r.Status, r.Message)
			}
			pass, fail, skip := rep.Counts()
			fmt.Fprintf(&b, "pass=%d fail=%d skip=%d", pass, fail, skip)
			return b.String(), nil
		},
	}
}

// nodeSetFromArgs builds a NodeSet from rpc endpoints (attach) or data_dir
// (a setup's saved state).
func nodeSetFromArgs(args map[string]any) (node.NodeSet, error) {
	if urls := argStrings(args, "rpc"); len(urls) > 0 {
		eps := make([]attach.Endpoint, len(urls))
		for i, u := range urls {
			eps[i] = attach.Endpoint{RPCURL: u}
		}
		return attach.Build(argString(args, "chain", ""), "attached", eps)
	}
	if dir := argString(args, "data_dir", ""); dir != "" {
		return state.LoadNodeSet(dir)
	}
	return node.NodeSet{}, fmt.Errorf("provide rpc (array) or data_dir")
}
