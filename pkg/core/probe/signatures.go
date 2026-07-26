package probe

import (
	"sort"

	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// chainSignature describes how to detect one chain from an RPC endpoint. It is
// derived from a registered chain's manifest (Probe.Method + Consensus namespace
// + optional Probe.ChainIDs gate), never hardcoded here — a new chain is
// detectable purely by adding its manifest.
type chainSignature struct {
	chainType   string
	namespace   string  // namespace reported in Result.Namespaces
	probeMethod string  // RPC method to hit; "" = not detectable
	chainIDs    []int64 // non-empty = require endpoint chain id ∈ chainIDs (disambiguation)
}

// registrySignatures builds the ordered detection list from the chain registry.
// Order matters (first match wins): chains carrying a chain-id gate are tried
// before ungated ones, so a probe method shared by two chains (e.g.
// istanbul_getValidators for stablenet+wbft) resolves to the gated chain when
// the endpoint's id matches. Within a group the order is by chain type, for
// determinism.
func registrySignatures() []chainSignature {
	var sigs []chainSignature
	for _, name := range registry.Names() {
		p, err := registry.Get(name)
		if err != nil {
			continue
		}
		m := p.Manifest()
		if m.Probe.Method == "" {
			continue
		}
		sigs = append(sigs, chainSignature{
			chainType:   m.ID,
			namespace:   m.Consensus.RPCNamespace,
			probeMethod: m.Probe.Method,
			chainIDs:    m.Probe.ChainIDs,
		})
	}
	sort.SliceStable(sigs, func(i, j int) bool {
		gi, gj := len(sigs[i].chainIDs) > 0, len(sigs[j].chainIDs) > 0
		if gi != gj {
			return gi // gated (specific) before ungated (general)
		}
		return sigs[i].chainType < sigs[j].chainType
	})
	return sigs
}

// isKnownOverride reports whether s is a registered chain type (or the implicit
// "ethereum" fallback), so the override list stays in sync with the registry
// rather than being hardcoded here.
func isKnownOverride(s string) bool {
	if s == "ethereum" {
		return true
	}
	_, err := registry.Get(s)
	return err == nil
}
