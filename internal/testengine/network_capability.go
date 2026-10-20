package testengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

type networkNodeProbe struct {
	Chain      string `json:"chain,omitempty"`
	Node       int    `json:"node"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Source     string `json:"source"`
	Reason     string `json:"reason,omitempty"`
	Address    string `json:"address,omitempty"`
	Height     string `json:"height,omitempty"`
	CodeBytes  int    `json:"codeBytes,omitempty"`
	CodeSHA256 string `json:"codeSHA256,omitempty"`
}

type networkCapabilityProbe struct {
	Capability string             `json:"capability"`
	Status     string             `json:"status"`
	Nodes      []networkNodeProbe `json:"nodes"`
}

// networkCapabilityGate verifies required capabilities using observed behavior.
// The RPC head, WS acknowledgement and deployed-code/builtin reads are bounded
// and non-mutating. Unknown observations outrank known unsupported features.
func networkCapabilityGate(chain string, dial func(string) *rpc.Client, headers func(string) http.Header) func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, string) {
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord) (session.TestStatus, string) {
		probes := []networkCapabilityProbe{}
		status := session.TestStatus("")
		for _, capability := range slices.Compact(slices.Sorted(slices.Values(spec.Requires))) {
			if capability != "rpc" && capability != "ws" && !strings.HasPrefix(capability, registry.CapContract) {
				continue
			}
			result := networkCapabilityProbe{Capability: capability, Status: "supported", Nodes: []networkNodeProbe{}}
			if len(env.Nodes()) == 0 {
				result.Status = "unknown"
			}
			for _, n := range env.Nodes() {
				// A handoff can leave this run named after its predecessor. Only
				// old records without a node chain inherit the run's dialect.
				if n.Chain == "" {
					n.Chain = chain
				}
				probe := networkNodeProbe{Chain: n.Chain, Node: n.Index, Role: string(n.Role), Status: "unknown", Reason: "recorded_chain_unavailable"}
				_, chainErr := registry.Get(n.Chain)
				if !strings.HasPrefix(capability, registry.CapContract) || chainErr == nil {
					bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
					probe = probeNetworkNode(bounded, capability, n, chainContracts(n.Chain), dial, headers)
					cancel()
				}
				result.Nodes = append(result.Nodes, probe)
				if probe.Status == "unknown" || probe.Status == "unsupported" && result.Status != "unknown" {
					result.Status = probe.Status
				}
			}
			probes = append(probes, result)
			candidate := session.TestStatus("")
			switch result.Status {
			case "unknown":
				candidate = session.StatusBlocked
			case "unsupported":
				candidate = session.StatusSkip
				if spec.SkipsOn != nil && !slices.Contains(spec.SkipsOn, chain) {
					candidate = session.StatusFail
				}
			}
			status = preferCapabilityVerdict(status, candidate)
		}
		if len(probes) > 0 {
			data, _ := json.Marshal(map[string]any{"source": "runtime_network_probes", "probes": probes})
			record.Observation("network-capability-probes.json", data)
		}
		if status != "" {
			return status, "required network capability could not be confirmed; see network-capability-probes.json"
		}
		return "", ""
	}
}

func preferCapabilityVerdict(current, candidate session.TestStatus) session.TestStatus {
	rank := func(status session.TestStatus) int {
		switch status {
		case session.StatusBlocked:
			return 3
		case session.StatusFail:
			return 2
		case session.StatusSkip:
			return 1
		}
		return 0
	}
	if rank(candidate) > rank(current) {
		return candidate
	}
	return current
}

func probeNetworkNode(ctx context.Context, capability string, n node.Node, contracts map[string]string, dial func(string) *rpc.Client, headers func(string) http.Header) networkNodeProbe {
	result := networkNodeProbe{Chain: n.Chain, Node: n.Index, Role: string(n.Role), Status: "unknown", Source: "eth_blockNumber", Reason: "rpc_probe_unavailable"}
	if strings.HasPrefix(capability, registry.CapContract) {
		label := strings.TrimPrefix(capability, registry.CapContract)
		result.Source, result.Address = "eth_getCode", contracts[label]
		if label == "accountManager" {
			result.Source = "eth_call:isBlacklisted(address)"
		}
	}
	if capability == "ws" {
		result.Source = "eth_subscribe:newHeads"
		if n.WSURL == "" {
			result.Reason = "ws_endpoint_unrecorded"
			return result
		}
		var auth http.Header
		if headers != nil {
			auth = headers(n.RPCURL)
		}
		sub, err := rpc.SubscribeWithHeaders(ctx, n.WSURL, auth, "newHeads")
		if err != nil {
			classifyNetworkError(&result, err)
			return result
		}
		_ = sub.Close()
		result.Status, result.Reason = "supported", ""
		return result
	}
	if n.RPCURL == "" {
		return result
	}
	client := dial(n.RPCURL)
	if capability == "rpc" {
		var raw string
		err := client.Call(ctx, "eth_blockNumber", &raw)
		if err != nil {
			classifyNetworkError(&result, err)
			return result
		}
		if chainIDQuantity.MatchString(raw) {
			if height, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 64); err == nil {
				result.Status, result.Reason, result.Height = "supported", "", strconv.FormatUint(height, 10)
				return result
			}
		}
		result.Reason = "rpc_response_unusable"
		return result
	}
	label := strings.TrimPrefix(capability, registry.CapContract)
	result.Source = "eth_getCode"
	result.Address = contracts[label]
	if result.Address == "" {
		result.Reason = "contract_address_unresolved"
		return result
	}
	if label == "accountManager" {
		// This binary-provided system contract has no deployed EVM code. A
		// valid boolean getter response distinguishes it from an empty account.
		result.Source = "eth_call:isBlacklisted(address)"
		data := accounts.Selector("isBlacklisted(address)") + strings.Repeat("0", 64)
		var value string
		err := client.Call(ctx, "eth_call", &value, map[string]any{"to": result.Address, "data": data, "gas": "0x186a0"}, "latest")
		if err != nil {
			classifyNetworkError(&result, err)
			return result
		}
		if value == "0x" {
			result.Status, result.Reason = "unsupported", "builtin_contract_unavailable"
			return result
		}
		if value == "0x"+strings.Repeat("0", 64) || value == "0x"+strings.Repeat("0", 63)+"1" {
			result.Status, result.Reason = "supported", ""
			return result
		}
		result.Reason = "contract_response_unusable"
		return result
	}
	code, err := client.CodeAt(ctx, result.Address)
	if err != nil {
		classifyNetworkError(&result, err)
		return result
	}
	if code == "0x" {
		result.Status, result.Reason = "unsupported", "contract_code_absent"
		return result
	}
	if !strings.HasPrefix(code, "0x") {
		result.Reason = "contract_response_unusable"
		return result
	}
	bytes, err := hex.DecodeString(strings.TrimPrefix(code, "0x"))
	if err != nil || len(bytes) == 0 {
		result.Reason = "contract_response_unusable"
		return result
	}
	sum := sha256.Sum256(bytes)
	result.Status, result.Reason, result.CodeBytes, result.CodeSHA256 = "supported", "", len(bytes), hex.EncodeToString(sum[:])
	return result
}

func classifyNetworkError(result *networkNodeProbe, err error) {
	var server *rpc.ServerError
	if errors.As(err, &server) && server.Code == -32601 {
		result.Status, result.Reason = "unsupported", "rpc_method_unavailable"
	}
}
