package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// Default returns a Server with the built-in chainbench tools registered. Chain
// and test-case plugins must be imported by the binary for registration.
func Default(name, version string) *Server {
	s := NewServer(name, version)
	s.Register(chainsTool())
	s.Register(faucetTool())
	s.Register(verifyTool())
	s.Register(runTool())
	s.Register(testListTool())
	s.Register(hardforkTool())
	s.Register(upgradeTool())
	s.Register(validateTool())
	s.Register(consensusTool())
	s.Register(nodeRPCTool())
	s.Register(nodeStopTool())
	s.Register(nodeStartTool())
	s.Register(reportTool())
	s.Register(statusTool())
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
	s.Register(consensusStatusTool())
	s.Register(consensusHealthTool())
	s.Register(consensusBlockInfoTool())
	s.Register(logTimelineTool())
	s.Register(networkPeersTool())
	s.Register(chainNewTool())
	s.Register(chainUpTool())
	s.Register(chainStatusTool())
	s.Register(chainShowTool())
	s.Register(resourcePoolTool())
	s.Register(resourcePlanTool())
	s.Register(chainKeysTool())
	s.Register(chainPlaceTool())
	s.Register(chainGenesisTool())
	s.Register(chainConfigTool())
	s.Register(chainBuildTool())
	s.Register(chainDeployTool())
	s.Register(chainInitTool())
	s.Register(chainStartTool())
	s.Register(chainStopTool())
	s.Register(chainRestartTool())
	s.Register(chainResumeTool())
	s.Register(chainRmTool())
	s.Register(chainLogsTool())
	s.Register(chainHealthTool())
	s.Register(networkTopologyTool())
	// Key material. Registered as a group so adding a verb touches one place.
	for _, t := range keyringTools() {
		s.Register(t)
	}
	s.Register(stopTool())
	// Layered capability surface: one tool per registered capability
	// (chainbench.<version>.<chain>.<name>) + a chainbench.capabilities
	// discovery tool. Populated only if a features project is imported.
	s.RegisterCapabilities()
	return s
}

func reportTool() Tool {
	return Tool{
		Name:     "chainbench_report",
		ReadOnly: true,
		Description: "Read a run's report from a session directory. Args: workspaceDir, all " +
			"(combine every session under the directory into one tally, instead of reading the most recent).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string"},
				"all":          map[string]any{"type": "boolean"},
			},
			"required": []string{"workspaceDir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "workspaceDir", "")
			if dir == "" {
				return "", fmt.Errorf("workspaceDir is required")
			}
			all := app.ArgBool(args, "all", false)
			rep, err := app.Report(context.Background(), app.Deps{}, app.ReportIn{Dir: dir, All: all})
			if err != nil {
				return "", err
			}
			if len(rep.Tests) == 0 {
				return "no runs recorded", nil
			}
			// A combined report names the run per row: seq is unique within a run, so
			// several runs each have a seq 1 and the number alone misleads.
			combined := rep.Session == app.CombinedSession
			var b strings.Builder
			for _, t := range rep.Tests {
				if combined {
					fmt.Fprintf(&b, "%s %d %s [%s] %s\n", t.Session, t.Seq, t.ID, t.Env, t.Status)
					continue
				}
				fmt.Fprintf(&b, "%d %s [%s] %s\n", t.Seq, t.ID, t.Env, t.Status)
			}
			if combined {
				fmt.Fprintf(&b, "%d session(s) combined pass=%d fail=%d blocked=%d skip=%d",
					app.ReportSessions(rep), rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Blocked, rep.Summary.Skip)
				return b.String(), nil
			}
			fmt.Fprintf(&b, "session=%s pass=%d fail=%d blocked=%d skip=%d",
				rep.Session, rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Blocked, rep.Summary.Skip)
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
			raw, err := app.NodeCall(ctx, app.Deps{}, app.NodeCallIn{RPC: url, Method: method, Params: params})
			if err != nil {
				return "", err
			}
			return string(raw), nil
		},
	}
}

func consensusTool() Tool {
	return Tool{
		Name:        "chainbench_consensus",
		ReadOnly:    true,
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
			p, err := app.Chain(app.Deps{}, chain)
			if err != nil {
				return "", err
			}
			method := p.Manifest().Consensus.ValidatorsMethod
			res, err := app.Validators(ctx, app.Deps{}, argString(args, "chain", "stablenet"), "", "", argString(args, "rpc", ""))
			vals := res.Validators
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
		ReadOnly:    true,
		Description: "List the chains chainbench supports (id, consensus family, binary, chain id, RPC namespace).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			var b strings.Builder
			for _, id := range app.Chains(app.Deps{}) {
				p, err := app.Chain(app.Deps{}, id)
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
			hash, err := app.Faucet(ctx, app.Deps{}, app.FaucetIn{
				Chain: chainRefFromArgs(args), FromKey: argString(args, "from_key", ""),
				To: argString(args, "to", ""), Amount: argString(args, "amount", ""),
			})
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
		ReadOnly:    true,
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
			res, err := app.VerifyNetwork(ctx, app.Deps{}, app.VerifyNetworkIn{
				DataDir: argString(args, "workspaceDir", ""),
				Chain:   argString(args, "chain", ""),
				RPCURLs: argStrings(args, "rpc"),
			})
			if err != nil {
				return "", err
			}
			rep := res.Report
			var b strings.Builder
			// Reported beside producing: a split network produces on every side,
			// so producing alone is not the verdict. Not-checked says so.
			fmt.Fprintf(&b, "producing: %v\n", rep.Producing)
			switch {
			case !rep.Agreement.Checked:
				fmt.Fprintf(&b, "agreement: not checked (%s)\n", rep.Agreement.Detail)
			case rep.Agreement.Agreed:
				fmt.Fprintf(&b, "agreement: yes, at block %d\n", rep.Agreement.Height)
			default:
				fmt.Fprintf(&b, "agreement: NO — %s\n", rep.Agreement.Detail)
			}
			for _, n := range rep.Nodes {
				fmt.Fprintf(&b, "node%d %s chain_id=%d block=%d peers=%d ok=%v\n",
					n.Index, n.RPCURL, n.ChainID, n.BlockNumber, n.PeerCount, n.OK)
			}
			return b.String(), nil
		},
	}
}

func statusTool() Tool {
	return Tool{
		Name:        "chainbench_status",
		ReadOnly:    true,
		Description: "Report a workspace's node set (chain, network, and each node's role/rpc/pid). Args: workspaceDir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string"},
			},
			"required": []string{"workspaceDir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := argString(args, "workspaceDir", "")
			if dir == "" {
				return "", fmt.Errorf("workspaceDir is required")
			}
			res, err := app.NetworkStatus(ctx, app.Deps{}, app.NetworkStatusIn{DataDir: dir})
			if err != nil {
				return "", err
			}
			ns := res.Nodes
			var b strings.Builder
			fmt.Fprintf(&b, "chain=%s network=%s nodes=%d\n", ns.Chain, ns.Network, len(ns.Nodes))
			for _, n := range ns.Nodes {
				fmt.Fprintf(&b, "  node%d %s %s pid=%d\n", n.Index, n.Role, n.RPCURL, n.PID)
			}
			return b.String(), nil
		},
	}
}

func txpoolTool() Tool {
	return Tool{
		Name:        "chainbench_txpool",
		ReadOnly:    true,
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
			raw, err := app.NodeCall(ctx, app.Deps{}, app.NodeCallIn{RPC: url, Method: "txpool_status"})
			if err != nil {
				return "", err
			}
			if err := json.Unmarshal(raw, &st); err != nil {
				return "", fmt.Errorf("mcp: txpool: %w", err)
			}
			return fmt.Sprintf("pending=%d queued=%d", hexCount(st.Pending), hexCount(st.Queued)), nil
		},
	}
}

func logTool() Tool {
	return Tool{
		Name:        "chainbench_log",
		ReadOnly:    true,
		Description: "Search a workspace's per-node logs (workspaceDir/logs). Args: workspaceDir, pattern, regexp (bool), node (int), level (min severity), limit (int).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string"},
				"pattern":      map[string]any{"type": "string"},
				"regexp":       map[string]any{"type": "boolean"},
				"node":         map[string]any{"type": "integer"},
				"level":        map[string]any{"type": "string"},
				"limit":        map[string]any{"type": "integer"},
			},
			"required": []string{"workspaceDir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "workspaceDir", "")
			if dir == "" {
				return "", fmt.Errorf("workspaceDir is required")
			}
			regexp, _ := args["regexp"].(bool)
			matches, err := app.LogSearch(context.Background(), app.Deps{}, app.LogSearchIn{
				Dir: dir,
				SearchOpts: app.LogSearchFilter{
					Pattern: argString(args, "pattern", ""),
					Regexp:  regexp,
					Node:    argInt(args, "node", 0),
					Level:   argString(args, "level", ""),
					Limit:   argInt(args, "limit", 0),
				},
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
		ReadOnly:    true,
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
			out, err := app.AccountState(ctx, app.Deps{}, app.AccountStateIn{RPC: url, Address: addr})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("address=%s balance=%s nonce=%d contract=%v",
				out.Address, out.Balance, out.Nonce, out.Contract), nil
		},
	}
}

func contractCallTool() Tool {
	return Tool{
		Name:        "chainbench_contract_call",
		ReadOnly:    true,
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
			return app.ContractCall(ctx, app.Deps{}, app.ContractCallIn{
				RPC: url, To: to, Data: argString(args, "data", ""),
			})
		},
	}
}

func txWaitTool() Tool {
	return Tool{
		Name:        "chainbench_tx_wait",
		ReadOnly:    true,
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
			r, err := app.TxWait(ctx, app.Deps{}, app.TxWaitIn{
				RPC: url, Hash: hash,
				Timeout: time.Duration(argInt(args, "timeout_seconds", 30)) * time.Second,
			})
			if err != nil {
				return "", err
			}
			return asJSON(r)
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
			hash, err := app.TxSend(ctx, app.Deps{}, app.TxSendIn{
				Chain: chainRefFromArgs(args), FromKey: argString(args, "from_key", ""),
				To: argString(args, "to", ""), Data: argString(args, "data", ""),
				Value: argString(args, "value", "0"),
			})
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
			out, err := app.ContractDeploy(ctx, app.Deps{}, app.ContractDeployIn{
				Chain: chainRefFromArgs(args), FromKey: argString(args, "from_key", ""),
				Bytecode: argString(args, "bytecode", ""), Value: argString(args, "value", "0"),
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("tx: %s\ncontract: %s", out.Tx, out.Address), nil
		},
	}
}

// chainRefFromArgs names which chain's rules apply and where to reach it.
func chainRefFromArgs(args map[string]any) app.ChainRef {
	return app.ChainRef{
		Chain: argString(args, "chain", "stablenet"),
		RPC:   argString(args, "rpc", ""),
	}
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
