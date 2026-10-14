package testengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

type capabilityNodeProbe struct {
	Chain          string `json:"chain,omitempty"`
	Node           int    `json:"node"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	Source         string `json:"source,omitempty"`
	Reason         string `json:"reason,omitempty"`
	ValidatorCount int    `json:"validatorCount,omitempty"`
}

// Consensus support is checked on every recorded endpoint before execution.
// Other RPC/VM probes remain part of R3, not inferred from manifest entries.
func consensusCapabilityGate(chain string, dial func(string) *rpc.Client) func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, string) {
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord) (session.TestStatus, string) {
		if !slices.Contains(spec.Requires, "consensus") {
			return "", ""
		}
		result := struct {
			Capability     string                `json:"capability"`
			Status         string                `json:"status"`
			Source         string                `json:"source,omitempty"`
			Node           int                   `json:"node,omitempty"`
			Reason         string                `json:"reason,omitempty"`
			ValidatorCount int                   `json:"validatorCount,omitempty"`
			Nodes          []capabilityNodeProbe `json:"nodes"`
		}{Capability: "consensus", Status: "unknown", Nodes: []capabilityNodeProbe{}}
		nodes := env.Nodes()
		plugin, err := registry.Get(chain)
		if err != nil || len(nodes) == 0 {
			result.Reason = "no_probe_target"
		} else {
			result.Status = "supported"
			for i, n := range nodes {
				selected := plugin
				var selectErr error
				if n.Chain != "" {
					selected, selectErr = registry.Get(n.Chain)
				}
				probe := capabilityNodeProbe{Node: n.Index, Role: string(n.Role), Chain: n.Chain, Status: "unknown", Reason: "recorded_chain_unavailable"}
				if selectErr == nil {
					probe = probeConsensusNode(ctx, selected, n, dial)
				}
				result.Nodes = append(result.Nodes, probe)
				if i == 0 {
					// Preserve the primary-node fields for existing readers.
					result.Node, result.Source, result.ValidatorCount = probe.Node, probe.Source, probe.ValidatorCount
				}
				if probe.Status == "unknown" || (probe.Status == "unsupported" && result.Status != "unknown") {
					result.Status, result.Reason = probe.Status, probe.Reason
				}
			}
		}
		status := session.StatusBlocked
		switch result.Status {
		case "supported":
			status = ""
		case "unsupported":
			status = session.StatusSkip
		}
		bytes, _ := json.Marshal(result)
		record.Observation("capability-probes.json", bytes)
		reason := result.Reason
		if status == session.StatusSkip && spec.SkipsOn != nil && !slices.Contains(spec.SkipsOn, chain) {
			status = session.StatusFail
			reason = "undeclared_runtime_capability_skip"
		}
		if status == "" {
			return "", ""
		}
		return status, fmt.Sprintf("consensus capability %s: %s", result.Status, reason)
	}
}

func probeConsensusNode(ctx context.Context, plugin registry.ChainPlugin, n node.Node, dial func(string) *rpc.Client) capabilityNodeProbe {
	result := capabilityNodeProbe{Node: n.Index, Role: string(n.Role), Chain: plugin.Manifest().ID, Status: "unknown", Reason: "no_probe_target"}
	if n.RPCURL == "" {
		return result
	}
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	label, validators, err := registry.RunningValidators(bounded, plugin, dial(n.RPCURL))
	result.Source = label
	switch {
	case err == nil && validators != nil:
		result.Status, result.Reason, result.ValidatorCount = "supported", "", len(validators)
	case err != nil:
		var serverErr *rpc.ServerError
		if errors.As(err, &serverErr) && serverErr.Code == -32601 {
			result.Status, result.Reason = "unsupported", "rpc_method_unavailable"
		} else {
			result.Reason = "rpc_probe_unavailable"
		}
	default:
		result.Reason = "rpc_response_unusable"
	}
	return result
}
