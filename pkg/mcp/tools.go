package mcp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/consensus"
	"github.com/0xmhha/chainbench/pkg/core/logs"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
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
	s.Register(reportTool())
	s.Register(statusTool())
	s.Register(setupPlanTool())
	s.Register(txpoolTool())
	s.Register(logTool())
	s.Register(accountStateTool())
	s.Register(contractCallTool())
	s.Register(txWaitTool())
	s.Register(txSendTool())
	s.Register(contractDeployTool())
	s.Register(networkAttachTool())
	s.Register(networkListTool())
	s.Register(networkInfoTool())
	s.Register(networkDetachTool())
	s.Register(remoteRPCTool())
	s.Register(testListTool())
	s.Register(consensusStatusTool())
	s.Register(consensusHealthTool())
	s.Register(consensusBlockInfoTool())
	s.Register(logTimelineTool())
	s.Register(networkPeersTool())
	return s
}

func reportTool() Tool {
	return Tool{
		Name:        "chainbench_report",
		Description: "Read stored run/test results from a setup's data dir. Args: data_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data_dir": map[string]any{"type": "string"},
			},
			"required": []string{"data_dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "data_dir", "")
			if dir == "" {
				return "", fmt.Errorf("data_dir is required")
			}
			store, err := obs.NewFileStore(filepath.Join(dir, "runs.json"))
			if err != nil {
				return "", err
			}
			runs := store.ListRuns()
			if len(runs) == 0 {
				return "no runs recorded", nil
			}
			var b strings.Builder
			var ok, failed int
			for _, r := range runs {
				fmt.Fprintf(&b, "%s [%s] %s %s\n", r.ID, r.Phase, r.Chain, r.Status)
				switch r.Status {
				case obs.RunSucceeded:
					ok++
				case obs.RunFailed:
					failed++
				}
			}
			fmt.Fprintf(&b, "total=%d ok=%d failed=%d", len(runs), ok, failed)
			return b.String(), nil
		},
	}
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

func statusTool() Tool {
	return Tool{
		Name:        "chainbench_status",
		Description: "Report the saved node set for a setup's data dir (chain, network, and each node's role/rpc/pid). Args: data_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data_dir": map[string]any{"type": "string"},
			},
			"required": []string{"data_dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "data_dir", "")
			if dir == "" {
				return "", fmt.Errorf("data_dir is required")
			}
			ns, err := state.LoadNodeSet(dir)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "chain=%s network=%s nodes=%d\n", ns.Chain, ns.Network, len(ns.Nodes))
			for _, n := range ns.Nodes {
				fmt.Fprintf(&b, "  node%d %s %s pid=%d\n", n.Index, n.Role, n.RPCURL, n.PID)
			}
			return b.String(), nil
		},
	}
}

func setupPlanTool() Tool {
	return Tool{
		Name:        "chainbench_setup_plan",
		Description: "Preview a local network plan (nodes, roles, ports, genesis path) without launching. Args: chain, validators, endpoints, data_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":      map[string]any{"type": "string"},
				"validators": map[string]any{"type": "integer"},
				"endpoints":  map[string]any{"type": "integer"},
				"data_dir":   map[string]any{"type": "string"},
			},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			p, err := registry.Get(argString(args, "chain", "stablenet"))
			if err != nil {
				return "", err
			}
			override := config.Values{
				"nodes.validators": strconv.Itoa(argInt(args, "validators", 4)),
				"nodes.endpoints":  strconv.Itoa(argInt(args, "endpoints", 1)),
			}
			plan, err := setup.BuildPlan(config.Resolve(nil, override), p, argString(args, "data_dir", "data"))
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "chain=%s network=%s dataRoot=%s genesis=%s\n",
				plan.Chain, plan.Network, plan.DataRoot, plan.GenesisPath)
			for _, n := range plan.Nodes {
				fmt.Fprintf(&b, "  node%d %s p2p=%d http=%d ws=%d\n",
					n.Index, n.Role, n.Ports.P2P, n.Ports.HTTP, n.Ports.WS)
			}
			return b.String(), nil
		},
	}
}

func txpoolTool() Tool {
	return Tool{
		Name:        "chainbench_txpool",
		Description: "Report a node's transaction pool status (pending/queued counts). Args: rpc.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rpc": map[string]any{"type": "string"},
			},
			"required": []string{"rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			url := argString(args, "rpc", "")
			if url == "" {
				return "", fmt.Errorf("rpc is required")
			}
			var st struct {
				Pending string `json:"pending"`
				Queued  string `json:"queued"`
			}
			if err := rpc.Dial(url).Call(ctx, "txpool_status", &st); err != nil {
				return "", err
			}
			return fmt.Sprintf("pending=%d queued=%d", hexCount(st.Pending), hexCount(st.Queued)), nil
		},
	}
}

func logTool() Tool {
	return Tool{
		Name:        "chainbench_log",
		Description: "Search a setup's per-node logs (data_dir/logs). Args: data_dir, pattern, regexp (bool), node (int), level (min severity), limit (int).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data_dir": map[string]any{"type": "string"},
				"pattern":  map[string]any{"type": "string"},
				"regexp":   map[string]any{"type": "boolean"},
				"node":     map[string]any{"type": "integer"},
				"level":    map[string]any{"type": "string"},
				"limit":    map[string]any{"type": "integer"},
			},
			"required": []string{"data_dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "data_dir", "")
			if dir == "" {
				return "", fmt.Errorf("data_dir is required")
			}
			regexp, _ := args["regexp"].(bool)
			matches, err := logs.Search(dir, logs.SearchOpts{
				Pattern: argString(args, "pattern", ""),
				Regexp:  regexp,
				Node:    argInt(args, "node", 0),
				Level:   argString(args, "level", ""),
				Limit:   argInt(args, "limit", 0),
			})
			if err != nil {
				return "", err
			}
			if len(matches) == 0 {
				return "no matching log lines", nil
			}
			var b strings.Builder
			for _, m := range matches {
				fmt.Fprintf(&b, "node%d:%d %s\n", m.Node, m.Line, m.Text)
			}
			fmt.Fprintf(&b, "%d line(s)", len(matches))
			return b.String(), nil
		},
	}
}

func accountStateTool() Tool {
	return Tool{
		Name:        "chainbench_account_state",
		Description: "Report an account's balance (wei), nonce, and whether it holds contract code. Args: rpc, address.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rpc":     map[string]any{"type": "string"},
				"address": map[string]any{"type": "string"},
			},
			"required": []string{"rpc", "address"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			url, addr := argString(args, "rpc", ""), argString(args, "address", "")
			if url == "" || addr == "" {
				return "", fmt.Errorf("rpc and address are required")
			}
			c := rpc.Dial(url)
			bal, err := c.BalanceAt(ctx, addr)
			if err != nil {
				return "", err
			}
			nonce, err := c.NonceAt(ctx, addr)
			if err != nil {
				return "", err
			}
			code, err := c.CodeAt(ctx, addr)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("address=%s balance=%s nonce=%d contract=%v",
				addr, bal.String(), nonce, code != "" && code != "0x" && code != "0x0"), nil
		},
	}
}

func contractCallTool() Tool {
	return Tool{
		Name:        "chainbench_contract_call",
		Description: "Read-only contract call (eth_call), returning the 0x-hex result. Args: rpc, to, data.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rpc":  map[string]any{"type": "string"},
				"to":   map[string]any{"type": "string"},
				"data": map[string]any{"type": "string"},
			},
			"required": []string{"rpc", "to"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			url, to := argString(args, "rpc", ""), argString(args, "to", "")
			if url == "" || to == "" {
				return "", fmt.Errorf("rpc and to are required")
			}
			return rpc.Dial(url).EthCall(ctx, to, argString(args, "data", ""))
		},
	}
}

func txWaitTool() Tool {
	return Tool{
		Name:        "chainbench_tx_wait",
		Description: "Wait for a transaction receipt and return it as JSON. Args: rpc, hash, timeout_seconds (default 30).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rpc":             map[string]any{"type": "string"},
				"hash":            map[string]any{"type": "string"},
				"timeout_seconds": map[string]any{"type": "integer"},
			},
			"required": []string{"rpc", "hash"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			url, hash := argString(args, "rpc", ""), argString(args, "hash", "")
			if url == "" || hash == "" {
				return "", fmt.Errorf("rpc and hash are required")
			}
			c := rpc.Dial(url)
			timeout := time.Duration(argInt(args, "timeout_seconds", 30)) * time.Second
			for waited := time.Duration(0); ; waited += time.Second {
				rec, err := c.TxReceipt(ctx, hash)
				if err != nil {
					return "", err
				}
				if rec != nil {
					return string(rec), nil
				}
				if waited >= timeout {
					return "", fmt.Errorf("timed out after %s waiting for tx %s", timeout, hash)
				}
				time.Sleep(time.Second)
			}
		},
	}
}

func txSendTool() Tool {
	return Tool{
		Name:        "chainbench_tx_send",
		Description: "Sign and send a transaction to an address (optionally with calldata). Args: chain, rpc, from_key (hex), to, data (hex), value (wei).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":    map[string]any{"type": "string"},
				"rpc":      map[string]any{"type": "string"},
				"from_key": map[string]any{"type": "string"},
				"to":       map[string]any{"type": "string"},
				"data":     map[string]any{"type": "string"},
				"value":    map[string]any{"type": "string"},
			},
			"required": []string{"rpc", "from_key", "to"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			w, err := openWalletFromArgs(ctx, args)
			if err != nil {
				return "", err
			}
			data, err := hexBytes(argString(args, "data", ""))
			if err != nil {
				return "", fmt.Errorf("bad data: %w", err)
			}
			wei, err := weiArg(args)
			if err != nil {
				return "", err
			}
			hash, err := w.Execute(ctx, argString(args, "to", ""), data, wei)
			if err != nil {
				return "", err
			}
			return "tx: " + hash, nil
		},
	}
}

func contractDeployTool() Tool {
	return Tool{
		Name:        "chainbench_contract_deploy",
		Description: "Deploy a contract from creation bytecode, returning the tx hash and contract address. Args: chain, rpc, from_key (hex), bytecode (hex), value (wei).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":    map[string]any{"type": "string"},
				"rpc":      map[string]any{"type": "string"},
				"from_key": map[string]any{"type": "string"},
				"bytecode": map[string]any{"type": "string"},
				"value":    map[string]any{"type": "string"},
			},
			"required": []string{"rpc", "from_key", "bytecode"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			w, err := openWalletFromArgs(ctx, args)
			if err != nil {
				return "", err
			}
			code, err := hexBytes(argString(args, "bytecode", ""))
			if err != nil {
				return "", fmt.Errorf("bad bytecode: %w", err)
			}
			if len(code) == 0 {
				return "", fmt.Errorf("bytecode is required")
			}
			wei, err := weiArg(args)
			if err != nil {
				return "", err
			}
			hash, addr, err := w.Deploy(ctx, code, wei)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("tx: %s\ncontract: %s", hash, addr), nil
		},
	}
}

// openWalletFromArgs opens an accounts wallet from the chain/from_key/rpc args.
func openWalletFromArgs(ctx context.Context, args map[string]any) (accounts.Wallet, error) {
	url := argString(args, "rpc", "")
	if url == "" {
		return nil, fmt.Errorf("rpc is required")
	}
	ap, err := accounts.ForChain(argString(args, "chain", "stablenet"))
	if err != nil {
		return nil, err
	}
	key, err := hexBytes(argString(args, "from_key", ""))
	if err != nil {
		return nil, fmt.Errorf("bad from_key: %w", err)
	}
	return ap.OpenWallet(ctx, key, url)
}

// hexBytes decodes a 0x-prefixed or bare hex string; "" yields nil.
func hexBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// weiArg parses the optional decimal "value" argument (wei), defaulting to 0.
func weiArg(args map[string]any) (*big.Int, error) {
	s := argString(args, "value", "")
	if s == "" {
		return big.NewInt(0), nil
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("bad value %q (decimal wei expected)", s)
	}
	return v, nil
}

// hexCount parses a 0x-hex count (e.g. txpool_status fields) to a uint64; a
// blank or malformed value yields 0.
func hexCount(s string) uint64 {
	n, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	if err != nil {
		return 0
	}
	return n
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
