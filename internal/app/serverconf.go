package app

import (
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/serverset"
)

// Resolving the server set is one job shared by every surface that
// composes a network: read the file if the operator has one, pick the server
// they named, and hand back both the placement (ports, mode, capacity) and the
// target (where the data plane lives). Host addresses and ports never appear in
// a spec or a flag — they come from the gitignored server set.

// ServerRef selects a server from a server set.
type ServerRef struct {
	// SetPath is the server-set file; empty uses serverset.DefaultConfigFile
	// when it exists, and otherwise no server set at all.
	SetPath string
	// Name and Index select within the server set. Both empty picks the only
	// server in a single-server file.
	Name  string
	Index int
	// Fleet spreads the network across every server in the server set, one node
	// per host, instead of placing it on one.
	Fleet bool
}

// ResolveServerOut is what a composition needs from the server set.
type ResolveServerOut struct {
	// Placement carries the port bands, addressing mode, and capacity bound.
	Placement serverset.Placement
	// Target is where the data plane lives. It is the zero value when no
	// server set applied, leaving the workspace's own target in place.
	Target machine.Spec
	// HasTarget reports whether Target should replace the workspace's.
	HasTarget bool
}

// ResolveServer turns a server selection into a placement and a target. With no
// selection and no default server set on disk it returns the built-in placement,
// which names itself as such so the port provenance stays visible.
func ResolveServer(d Deps, ref ServerRef, minValidators, portBand int) (ResolveServerOut, error) {
	path := ref.SetPath
	if path == "" {
		// A server set at the default path is used when present, but its
		// absence is only an error if the caller named a server — or if the
		// file still sits there under its pre-rename name, which must not
		// silently degrade a configured placement to the built-in local one.
		if _, err := os.Stat(serverset.DefaultConfigFile); err != nil {
			if hint := serverset.LegacyNameHint(serverset.DefaultConfigFile); hint != "" {
				return ResolveServerOut{}, fmt.Errorf("app: %s", hint)
			}
			if ref.Name != "" || ref.Index != 0 || ref.Fleet {
				return ResolveServerOut{}, fmt.Errorf(
					"app: --server needs a server set: %s not found (copy %s)",
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
// is recorded as a TargetServer spec naming its set entry — not flattened to a
// host/user pair — so every later step resolves the login from the server set
// file, the single source of a named server's credentials. Host is still
// carried for display and RPC addressing; it never authenticates anything.
func serverTarget(s serverset.Server) machine.Spec {
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
func fleetTarget(pl serverset.Placement) machine.Spec {
	spec := machine.Spec{Kind: machine.KindLocal, DataRoot: pl.DataRoot}
	if pl.Remote {
		spec.Kind = machine.KindRemote
		if len(pl.Pool.Hosts) > 0 {
			spec.Host = pl.Pool.Hosts[0].Addr
		}
	}
	return spec
}
