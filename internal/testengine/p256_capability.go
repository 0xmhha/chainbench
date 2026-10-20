package testengine

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// p256ProbeValidInput is the corpus signature vector, independently verified
// with crypto/ecdsa by the gate's contract tests. It contains only public data.
const p256ProbeValidInput = "0x4fc05e7c6fa9dcfc4d09b26e3352487de9f248699f930af99fc652c7374dc89cc1e7730e0dd29f6e231649b948bf9dfa73b48f94e904615c3a6573fe05c3b6bf68781757cc58563fb2ea9243822fcb3b973461be164b8d57f986146d16f2ee5e1ccbe91c075fc7f4f033bfa248db8fccd3565de94bbfb12f3c59ff46c271bf83ce4014c68811f9a21a1fdb2c0e6113e06db7ca93b7404e78dc7ccd5ca89a4ca9"
const p256ProbeAddress = "0x0000000000000000000000000000000000000100"

type p256NodeProbe struct {
	capabilityNodeProbe
	ValidResult   string `json:"validResult,omitempty"`
	InvalidResult string `json:"invalidResult,omitempty"`
}

func p256CapabilityGate(chain string, dial func(string) *rpc.Client) func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, string) {
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord) (session.TestStatus, string) {
		if !slices.Contains(spec.Requires, "precompile:secp256r1") {
			return "", ""
		}
		invalid := p256ProbeValidInput[:66] + strings.Repeat("0", 64) + p256ProbeValidInput[130:]
		one := "0x" + strings.Repeat("0", 63) + "1"
		result := struct {
			Capability   string          `json:"capability"`
			Status       string          `json:"status"`
			Method       string          `json:"method"`
			Address      string          `json:"address"`
			ValidInput   string          `json:"validInput"`
			InvalidInput string          `json:"invalidInput"`
			Nodes        []p256NodeProbe `json:"nodes"`
		}{Capability: "precompile:secp256r1", Status: "supported", Method: "eth_call", Address: p256ProbeAddress, ValidInput: p256ProbeValidInput, InvalidInput: invalid, Nodes: []p256NodeProbe{}}
		if len(env.Nodes()) == 0 {
			result.Status = "unknown"
		}
		for _, n := range env.Nodes() {
			probe := p256NodeProbe{capabilityNodeProbe: capabilityNodeProbe{Node: n.Index, Role: string(n.Role), Source: "eth_call:secp256r1", Status: "unknown", Reason: "rpc_probe_unavailable"}}
			if n.RPCURL != "" {
				bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
				var validOutput, invalidOutput string
				client := dial(n.RPCURL)
				err := client.Call(bounded, "eth_call", &validOutput, map[string]any{"to": p256ProbeAddress, "data": p256ProbeValidInput, "gas": "0x186a0"}, "latest")
				if err == nil {
					err = client.Call(bounded, "eth_call", &invalidOutput, map[string]any{"to": p256ProbeAddress, "data": invalid, "gas": "0x186a0"}, "latest")
				}
				cancel()
				// Persist only the two known public response shapes, never arbitrary
				// upstream error text or an unrelated server's response contents.
				if validOutput == "0x" || validOutput == one {
					probe.ValidResult = validOutput
				}
				if invalidOutput == "0x" || invalidOutput == one {
					probe.InvalidResult = invalidOutput
				}
				switch {
				case err != nil:
				case validOutput == one && invalidOutput == "0x":
					probe.Status, probe.Reason = "supported", ""
				case validOutput == "0x" && invalidOutput == "0x":
					probe.Status, probe.Reason = "unsupported", "p256_verification_unavailable"
				case invalidOutput == one:
					probe.Reason = "negative_control_accepted"
				default:
					probe.Reason = "precompile_response_unusable"
				}
			}
			result.Nodes = append(result.Nodes, probe)
			if probe.Status == "unknown" || (probe.Status == "unsupported" && result.Status != "unknown") {
				result.Status = probe.Status
			}
		}
		data, _ := json.Marshal(result)
		record.Observation("precompile-capability-probes.json", data)
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
		return status, fmt.Sprintf("precompile:secp256r1 capability %s (valid/invalid signature probes)", result.Status)
	}
}
