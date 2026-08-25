package netmap

import (
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/netmap/internal/serverset"
)

// The server set is this module's data: which machines exist, how they are
// reached, and what port slots they offer. The format lives in the internal
// serverset package — no other module reads it directly (the compiler
// enforces that), so what leaves this file is the whole external surface.

// DefaultSetFile is the server-set path used when --server-set is omitted.
const DefaultSetFile = serverset.DefaultConfigFile

// DefaultSampleFile is the tracked template an operator copies.
const DefaultSampleFile = serverset.DefaultSampleFile

// Set is a parsed server set.
type Set = serverset.Config

// Server is one machine of the set.
type Server = serverset.Server

// Placement carries the port bands, addressing mode, and capacity bound a
// composition allocates within.
type Placement = serverset.Placement

// LoadSet reads and validates the server set at path.
func LoadSet(path string) (*Set, error) { return serverset.Load(path) }

// Builtin is the placement used with no server set: this machine, stepped
// ports. It names itself as such so port provenance stays visible.
func Builtin(minValidators, portBand int) Placement {
	return serverset.Builtin(minValidators, portBand)
}

// ServerRef selects a server from a server set.
type ServerRef struct {
	// SetPath is the server-set file; empty uses DefaultSetFile when it
	// exists, and otherwise no server set at all.
	SetPath string
	// Name and Index select within the server set. Both empty picks the only
	// server in a single-server file.
	Name  string
	Index int
	// Fleet spreads the network across every server in the set, one node per
	// host, instead of placing it on one.
	Fleet bool
}

// ResolveServerOut is what a composition needs from the server set.
type ResolveServerOut struct {
	// Placement carries the port bands, addressing mode, and capacity bound.
	Placement Placement
	// Target is where the data plane lives. It is the zero value when no
	// server set applied, leaving the workspace's own target in place.
	Target machine.Spec
	// HasTarget reports whether Target should replace the workspace's.
	HasTarget bool
}

// ResolveServer turns a server selection into a placement and a target. With
// no selection and no default server set on disk it returns the built-in
// placement, which names itself as such so the port provenance stays visible.
func ResolveServer(ref ServerRef, minValidators, portBand int) (ResolveServerOut, error) {
	path := ref.SetPath
	if path == "" {
		// A server set at the default path is used when present, but its
		// absence is only an error if the caller named a server — or if the
		// file still sits there under its pre-rename name, which must not
		// silently degrade a configured placement to the built-in local one.
		if _, err := os.Stat(serverset.DefaultConfigFile); err != nil {
			if hint := serverset.LegacyNameHint(serverset.DefaultConfigFile); hint != "" {
				return ResolveServerOut{}, fmt.Errorf("netmap: %s", hint)
			}
			if ref.Name != "" || ref.Index != 0 || ref.Fleet {
				return ResolveServerOut{}, fmt.Errorf(
					"netmap: --server needs a server set: %s not found (copy %s)",
					serverset.DefaultConfigFile, serverset.DefaultSampleFile)
			}
			return ResolveServerOut{Placement: serverset.Builtin(minValidators, portBand)}, nil
		}
		path = serverset.DefaultConfigFile
	}
	cfg, err := serverset.Load(path)
	if err != nil {
		return ResolveServerOut{}, err
	}
	if ref.Fleet {
		pl, err := cfg.Fleet(minValidators, portBand)
		if err != nil {
			return ResolveServerOut{}, err
		}
		return ResolveServerOut{Placement: pl, Target: fleetTarget(pl), HasTarget: true}, nil
	}
	srv, err := cfg.Select(ref.Name, ref.Index)
	if err != nil {
		return ResolveServerOut{}, err
	}
	pl := cfg.Placement(srv, minValidators, portBand)
	return ResolveServerOut{Placement: pl, Target: serverTarget(srv), HasTarget: true}, nil
}

// serverTarget describes where one server's data plane lives. A remote server
// is recorded as a KindServer spec naming its set entry — not flattened to a
// host/user pair — so every later step resolves the login from the server set
// file, the single source of a named server's credentials. Host is still
// carried for display and RPC addressing; it never authenticates anything.
func serverTarget(s Server) machine.Spec {
	spec := machine.Spec{Kind: machine.KindLocal, DataRoot: s.DataRoot}
	if s.IsRemote() {
		spec.Kind = machine.KindServer
		spec.Server = s.Name
		spec.Host = s.Host
	}
	return spec
}

// fleetTarget describes a fleet's data plane. Its host is the first server's:
// the per-node addresses live on the node table, which the allocator fills.
func fleetTarget(pl Placement) machine.Spec {
	spec := machine.Spec{Kind: machine.KindLocal, DataRoot: pl.DataRoot}
	if pl.Remote {
		spec.Kind = machine.KindRemote
		if len(pl.Pool.Hosts) > 0 {
			spec.Host = pl.Pool.Hosts[0].Addr
		}
	}
	return spec
}
