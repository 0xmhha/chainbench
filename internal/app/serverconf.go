package app

import (
	"github.com/0xmhha/chainbench/internal/resource"
)

// Server selection is the resource module's job — it owns the server set. These
// aliases keep the app layer's callers on their existing names while the
// surfaces migrate to calling the module directly (worklist V3.3/V6).

// ServerRef selects a server from a server set.
type ServerRef = resource.ServerRef

// ResolveServerOut is what a composition needs from the server set.
type ResolveServerOut = resource.ResolveServerOut

// ResolveServer turns a server selection into a pool and a target.
func ResolveServer(_ Deps, ref ServerRef, minValidators, portBand int) (ResolveServerOut, error) {
	return resource.ResolveServer(ref, minValidators, portBand)
}
