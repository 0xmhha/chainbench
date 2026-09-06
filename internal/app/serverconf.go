package app

import (
	"github.com/0xmhha/chainbench/internal/resource"
)

// Server and target selection is the resource module's job — it owns the server
// set and the spelling of a target. These aliases let every surface name those
// through one layer (architecture-v2 §2).

// ServerRef selects a server from a server set.
type ServerRef = resource.ServerRef

// ResolveServerOut is what a composition needs from the server set.
type ResolveServerOut = resource.ResolveServerOut

// ResolveServer turns a server selection into a pool and a target.
func ResolveServer(_ Deps, ref ServerRef, minValidators, portBand int) (ResolveServerOut, error) {
	return resource.ResolveServer(ref, minValidators, portBand)
}

// TargetSpec says where a network's data plane lives: this machine, or a host
// reached over SSH, and the path under which its files land.
type TargetSpec = resource.Spec

// ParseTarget reads the single-path target spelling
// (/local/path, user@host:/path, ssh://user@host:port/path).
//
// A surface parses its own flags but must not invent its own grammar for this
// one: the same string typed at the CLI and passed to a tool has to mean the
// same machine, so both read it through here.
func ParseTarget(s string) (TargetSpec, error) { return resource.Parse(s) }
