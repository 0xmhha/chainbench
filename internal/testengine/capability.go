package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// wsCapability is advertised by locally launched nodes, which serve a WebSocket
// endpoint (--ws) for the eth_subscribe cases.
const wsCapability = "ws"

// attachCapability is the sole capability an attached network advertises: RPC
// reachability. Attach makes no producer/consensus/ws assumptions.
const attachCapability = "rpc"

// satisfies reports whether every required capability is present in provided.
func satisfies(required, provided []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]bool, len(provided))
	for _, c := range provided {
		set[c] = true
	}
	for _, r := range required {
		if !set[r] {
			return false
		}
	}
	return true
}

// applicableWithCaps composes chain applicability with capability gating: a spec
// applies only when its chain matches (see applicableTo) and the target network
// provides every capability the spec requires. A spec that requires a capability
// the target lacks is skipped, not failed.
func applicableWithCaps(chain string, provided []string) func(testspec.Spec) bool {
	chainOK := applicableTo(chain)
	return func(s testspec.Spec) bool {
		return chainOK(s) && satisfies(s.Requires, provided)
	}
}

// localCapabilities is the capability set a locally launched network of plugin
// advertises: the chain manifest's capabilities plus "ws" (launched nodes serve
// a WebSocket endpoint).
func localCapabilities(plugin registry.ChainPlugin) []string {
	caps := append([]string(nil), plugin.Manifest().Capabilities...)
	return append(caps, wsCapability)
}
