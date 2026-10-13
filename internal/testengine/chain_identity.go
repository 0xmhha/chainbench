package testengine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

var chainIDQuantity = regexp.MustCompile(`^0x(?:0|[1-9a-fA-F][0-9a-fA-F]*)$`)

// identityAndCapabilityGate validates the declared identity first. A matching
// capability cannot authorize running a case on a different or unknown chain.
func identityAndCapabilityGate(chain string, dial func(string) *rpc.Client) func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, string) {
	capability := consensusCapabilityGate(chain, dial)
	vm := shanghaiCapabilityGate(chain, dial)
	p256 := p256CapabilityGate(chain, dial)
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord) (session.TestStatus, string) {
		status, reason := checkDeclaredChainIdentity(ctx, spec, env, record, dial)
		if status != "" {
			return status, reason
		}
		if status, reason := capability(ctx, spec, env, record); status != "" {
			return status, reason
		}
		if status, reason := vm(ctx, spec, env, record); status != "" {
			return status, reason
		}
		return p256(ctx, spec, env, record)
	}
}

func checkDeclaredChainIdentity(ctx context.Context, spec dsl.Spec, env session.Environment, record session.TestRecord, dial func(string) *rpc.Client) (session.TestStatus, string) {
	expected, present, err := dsl.ExpectedChainID(spec.EnvValues)
	if !present {
		return "", ""
	}
	result := struct {
		Source   string           `json:"source"`
		Method   string           `json:"method"`
		Expected string           `json:"expected"`
		Status   string           `json:"status"`
		Nodes    []map[string]any `json:"nodes"`
	}{Source: "preset.values.chainId", Method: "eth_chainId", Expected: strconv.FormatUint(expected, 10), Status: "matched", Nodes: []map[string]any{}}
	status := session.TestStatus("")
	if err != nil {
		result.Status = "invalid_declaration"
		status = session.StatusFail
	}
	if err == nil && len(env.Nodes()) == 0 {
		result.Status = "unknown"
		status = session.StatusBlocked
	}
	if err == nil {
		mismatch, unknown := false, false
		for _, n := range env.Nodes() {
			item := map[string]any{"node": n.Index, "role": n.Role, "status": "unknown"}
			bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
			var raw string
			var probeErr error
			if n.RPCURL == "" {
				probeErr = fmt.Errorf("missing RPC endpoint")
			} else {
				probeErr = dial(n.RPCURL).Call(bounded, "eth_chainId", &raw)
			}
			cancel()
			switch {
			case probeErr != nil:
				item["reason"] = "chain_id_probe_unavailable"
				unknown = true
			case !chainIDQuantity.MatchString(raw):
				item["reason"] = "chain_id_response_unusable"
				unknown = true
			default:
				actual, parseErr := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 64)
				if parseErr != nil {
					item["reason"] = "chain_id_response_unusable"
					unknown = true
					break
				}
				item["actual"] = strconv.FormatUint(actual, 10)
				if actual != expected {
					item["status"] = "mismatched"
					mismatch = true
				} else {
					item["status"] = "matched"
				}
			}
			result.Nodes = append(result.Nodes, item)
		}
		if mismatch {
			result.Status = "mismatched"
			status = session.StatusFail
		} else if unknown {
			result.Status = "unknown"
			status = session.StatusBlocked
		}
	}
	encoded, _ := json.Marshal(result)
	record.Observation("chain-identity.json", encoded)
	if status == "" {
		return "", ""
	}
	return status, fmt.Sprintf("declared chain identity %s (preset.values.chainId)", result.Status)
}
