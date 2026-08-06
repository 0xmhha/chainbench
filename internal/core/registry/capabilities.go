package registry

// Capabilities is an optional ChainPlugin extension (DDD context C6, the
// anti-corruption layer). A plugin that implements it declares chain-specific
// composition and support facts; callers attach it via a type assertion, so
// plugins that do not implement it keep working.
type Capabilities interface {
	// NodeComposition describes how this chain composes its N nodes.
	NodeComposition() NodeComposition
	// SupportedForks lists the hardfork names this chain understands.
	SupportedForks() []string
	// SupportedAssertions lists the assertion namespaces (e.g. "istanbul", "wemix").
	SupportedAssertions() []string
	// TestCapabilities lists feature tags a test may require of this chain.
	TestCapabilities() []string
}

// NodeComposition describes a chain's node makeup and initialization hooks
// (e.g. etcd, ncp, staking) needed to form a working network.
type NodeComposition struct {
	// Roles maps a role name to its required count for a minimal network.
	Roles map[string]int
	// InitHooks names chain-specific initialization steps, in order.
	InitHooks []string
}
