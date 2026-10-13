package testengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// This first runtime probe covers consensus APIs. Additional RPC/VM probes and
// their native evidence remain part of R3, not inferred from manifest entries.
func consensusCapabilityGate(chain string, dial func(string) *rpc.Client) func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, string) {
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord) (session.TestStatus, string) {
		if !slices.Contains(spec.Requires, "consensus") {
			return "", ""
		}
		result := struct {
			Capability     string `json:"capability"`
			Status         string `json:"status"`
			Source         string `json:"source,omitempty"`
			Node           int    `json:"node,omitempty"`
			Reason         string `json:"reason,omitempty"`
			ValidatorCount int    `json:"validatorCount,omitempty"`
		}{Capability: "consensus", Status: "unknown"}
		status := session.StatusBlocked
		nodes := env.Nodes()
		plugin, err := registry.Get(chain)
		if err == nil && len(nodes) > 0 {
			result.Node = nodes[0].Index
			bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
			label, validators, probeErr := registry.RunningValidators(bounded, plugin, dial(nodes[0].RPCURL))
			cancel()
			result.Source = label
			switch {
			case probeErr == nil && validators != nil:
				result.Status = "supported"
				result.ValidatorCount = len(validators)
				status = ""
			case probeErr != nil:
				var serverErr *rpc.ServerError
				if errors.As(probeErr, &serverErr) && serverErr.Code == -32601 {
					result.Status = "unsupported"
					result.Reason = "rpc_method_unavailable"
					status = session.StatusSkip
				} else {
					result.Reason = "rpc_probe_unavailable"
				}
			default:
				result.Reason = "rpc_response_unusable"
			}
		} else {
			result.Reason = "no_probe_target"
		}
		bytes, _ := json.Marshal(result)
		record.Observation("capability-probes.json", bytes)
		if status == session.StatusSkip && spec.SkipsOn != nil && !slices.Contains(spec.SkipsOn, chain) {
			status = session.StatusFail
			result.Reason = "undeclared_runtime_capability_skip"
		}
		if status == "" {
			return "", ""
		}
		return status, fmt.Sprintf("consensus capability %s: %s", result.Status, result.Reason)
	}
}
