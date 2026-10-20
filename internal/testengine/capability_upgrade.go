package testengine

import (
	"context"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// A concurrent successor binary follows the predecessor's consensus until the
// declared fork. Its binary identity stays unchanged; select the active reader
// from the declaration and an observed head, never by falling back on RPC error.
func probeConsensusForSpec(ctx context.Context, chain string, selected registry.ChainPlugin, spec dsl.Spec, n node.Node, dial func(string) *rpc.Client) capabilityNodeProbe {
	u := spec.EnvUpgrade
	if u == nil || u.At == nil || *u.At <= 0 || (u.Style != "" && u.Style != dsl.UpgradeConcurrent) || n.RPCURL == "" {
		return probeConsensusNode(ctx, selected, n, dial)
	}
	from, to := u.From, u.To
	if from == "" {
		from = dsl.BinaryFrom
	}
	if to == "" {
		to = dsl.BinaryTo
	}
	predecessor, successor := spec.Chain.BinaryChains[from], spec.Chain.BinaryChains[to]
	if predecessor == "" {
		predecessor = chain
	}
	if successor == "" {
		successor = chain
	}
	if selected.Manifest().ID != successor || predecessor == successor {
		return probeConsensusNode(ctx, selected, n, dial)
	}
	result := capabilityNodeProbe{Chain: selected.Manifest().ID, Node: n.Index, Role: string(n.Role), Status: "unknown", Reason: "recorded_chain_unavailable"}
	prior, err := registry.Get(predecessor)
	if err != nil {
		return result
	}
	if prior.Family().ID() == selected.Family().ID() {
		return probeConsensusNode(ctx, selected, n, dial)
	}
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	height, err := dial(n.RPCURL).BlockNumber(bounded)
	if err != nil {
		result.Source, result.Reason = "eth_blockNumber", "rpc_probe_unavailable"
		result.FailureKind, result.RPCErrorCode = consensusProbeFailure(err)
		return result
	}
	active := selected
	if height < uint64(*u.At) {
		active = prior
		if reader, ok := selected.(registry.PreForkValidatorReader); ok {
			label, validators, err := reader.PreForkValidators(bounded, u.Fork, prior, dial(n.RPCURL))
			result = consensusProbeResult(result, label, validators, err)
		} else {
			result.Source, result.Reason = "pre-fork consensus", "pre_fork_consensus_unavailable"
		}
	} else {
		result = probeConsensusNode(bounded, active, n, dial)
	}
	result.Chain = selected.Manifest().ID
	result.ConsensusChain, result.Height, result.ForkBlock = active.Manifest().ID, strconv.FormatUint(height, 10), u.At
	return result
}
