package probe

// chainSignature describes how to detect one chain family.
// Order matters: evaluate top-to-bottom, first match wins.
// "istanbul_*" namespace is shared by stablenet and wbft; disambiguate via knownChainIDs.
type chainSignature struct {
	chainType     string
	namespace     string         // namespace name reported in Result.Namespaces
	probeMethod   string         // RPC method to hit; "" = skip (fallback rule)
	knownChainIDs map[int64]bool // non-nil = require chainID membership
}

var signatures = []chainSignature{
	{
		chainType:   "wemix",
		namespace:   "wemix",
		probeMethod: "wemix_getReward",
	},
	{
		chainType:     "stablenet",
		namespace:     "istanbul",
		probeMethod:   "istanbul_getValidators",
		knownChainIDs: map[int64]bool{8283: true},
	},
	{
		chainType:   "wbft",
		namespace:   "istanbul",
		probeMethod: "istanbul_getValidators",
	},
	// ethereum: implicit fallback, no probe.
}

// isKnownOverride returns true if the supplied override string maps to a known chain_type.
func isKnownOverride(s string) bool {
	switch s {
	case "stablenet", "wbft", "wemix", "ethereum":
		return true
	default:
		return false
	}
}
