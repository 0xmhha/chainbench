package testengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// shanghaiProbeCode exercises PUSH0 in a read-only contract-creation eth_call.
// Returning a zero word confirms execution instead of trusting a manifest tag.
const shanghaiProbeCode = "0x5f60005260206000f3"

func shanghaiCapabilityGate(chain string, dial func(string) *rpc.Client) func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, string) {
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord) (session.TestStatus, string) {
		if !slices.Contains(spec.Requires, "evm:shanghai") {
			return "", ""
		}
		result := struct {
			Capability string                `json:"capability"`
			Status     string                `json:"status"`
			Method     string                `json:"method"`
			Code       string                `json:"code"`
			Nodes      []capabilityNodeProbe `json:"nodes"`
		}{Capability: "evm:shanghai", Status: "supported", Method: "eth_call", Code: shanghaiProbeCode, Nodes: []capabilityNodeProbe{}}
		if len(env.Nodes()) == 0 {
			result.Status = "unknown"
		}
		for _, n := range env.Nodes() {
			probe := capabilityNodeProbe{Node: n.Index, Role: string(n.Role), Source: "eth_call:PUSH0", Status: "unknown", Reason: "rpc_probe_unavailable"}
			if n.RPCURL != "" {
				bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
				var output string
				err := dial(n.RPCURL).Call(bounded, "eth_call", &output, map[string]any{"data": shanghaiProbeCode, "gas": "0x186a0"}, "latest")
				cancel()
				switch {
				case err == nil && output == "0x"+strings.Repeat("0", 64):
					probe.Status, probe.Reason = "supported", ""
				case err == nil:
					probe.Reason = "rpc_response_unusable"
				case shanghaiUnavailable(err):
					probe.Status, probe.Reason = "unsupported", "push0_unavailable"
				}
			}
			result.Nodes = append(result.Nodes, probe)
			if probe.Status == "unknown" || (probe.Status == "unsupported" && result.Status != "unknown") {
				result.Status = probe.Status
			}
		}
		data, _ := json.Marshal(result)
		record.Observation("vm-capability-probes.json", data)
		if result.Status == "supported" {
			return "", ""
		}
		status := session.StatusBlocked
		if result.Status == "unsupported" {
			status = session.StatusSkip
			if spec.SkipsOn != nil && !slices.Contains(spec.SkipsOn, chain) {
				status = session.StatusFail
			}
		}
		return status, fmt.Sprintf("evm:shanghai capability %s (eth_call:PUSH0)", result.Status)
	}
}

func shanghaiUnavailable(err error) bool {
	var server *rpc.ServerError
	if !errors.As(err, &server) || server.Method != "eth_call" {
		return false
	}
	if server.Code >= -32768 && server.Code <= -32600 {
		return false
	}
	message := strings.ToLower(server.Message)
	return strings.Contains(message, "invalid opcode") && (strings.Contains(message, "push0") || strings.Contains(message, "opcode 0x5f"))
}
