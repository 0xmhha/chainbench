package probe

import "github.com/0xmhha/chainbench/pkg/core/registry"

// chainSignature describes how to detect one chain family from an RPC endpoint.
// Order matters: evaluate top-to-bottom, first match wins. The "istanbul_*"
// namespace is shared by stablenet and wbft; disambiguate resolves the ambiguity
// by requiring the endpoint's chain id to equal the chainType's registered
// manifest chain id (so the id is the manifest's, never a constant here).
type chainSignature struct {
	chainType    string
	namespace    string // namespace reported in Result.Namespaces
	probeMethod  string // RPC method to hit; "" = skip (fallback rule)
	disambiguate bool   // require endpoint chain id == manifest chain id for chainType
}

var signatures = []chainSignature{
	{
		chainType:   "wemix",
		namespace:   "wemix",
		probeMethod: "wemix_getReward",
	},
	{
		chainType:    "stablenet",
		namespace:    "istanbul",
		probeMethod:  "istanbul_getValidators",
		disambiguate: true,
	},
	{
		chainType:   "wbft",
		namespace:   "istanbul",
		probeMethod: "istanbul_getValidators",
	},
	// ethereum: implicit fallback, no probe.
}

// manifestChainID returns the registered chain's manifest chain id, or (0,
// false) when the chain type is not registered.
func manifestChainID(chainType string) (int64, bool) {
	p, err := registry.Get(chainType)
	if err != nil {
		return 0, false
	}
	return p.Manifest().ChainID, true
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
