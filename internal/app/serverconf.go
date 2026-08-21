package app

import (
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/target"
	"github.com/0xmhha/chainbench/internal/serverset"
)

// Resolving the server inventory is one job shared by every surface that
// composes a network: read the file if the operator has one, pick the server
// they named, and hand back both the placement (ports, mode, capacity) and the
// target (where the data plane lives). Host addresses and ports never appear in
// a spec or a flag — they come from the gitignored inventory.

// ServerRef selects a server from an inventory.
type ServerRef struct {
	// ConfigPath is the inventory file; empty uses serverset.DefaultConfigFile
	// when it exists, and otherwise no inventory at all.
	ConfigPath string
	// Name and Index select within the inventory. Both empty picks the only
	// server in a single-server file.
	Name  string
	Index int
	// Fleet spreads the network across every server in the inventory, one node
	// per host, instead of placing it on one.
	Fleet bool
}

// ResolveServerOut is what a composition needs from the inventory.
type ResolveServerOut struct {
	// Placement carries the port bands, addressing mode, and capacity bound.
	Placement serverset.Placement
	// Target is where the data plane lives. It is the zero value when no
	// inventory applied, leaving the workspace's own target in place.
	Target target.TargetSpec
	// HasTarget reports whether Target should replace the workspace's.
	HasTarget bool
}

// ResolveServer turns a server selection into a placement and a target. With no
// selection and no default inventory on disk it returns the built-in placement,
// which names itself as such so the port provenance stays visible.
func ResolveServer(d Deps, ref ServerRef, minValidators, portBand int) (ResolveServerOut, error) {
	path := ref.ConfigPath
	if path == "" {
		// An inventory at the default path is used when present, but its
		// absence is only an error if the caller named a server.
		if _, err := os.Stat(serverset.DefaultConfigFile); err != nil {
			if ref.Name != "" || ref.Index != 0 || ref.Fleet {
				return ResolveServerOut{}, fmt.Errorf(
					"app: --server needs an inventory: %s not found (copy %s)",
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

// serverTarget describes where one server's data plane lives. The local and
// remote cases differ only in the kind and the host, which is the point of the
// inventory carrying both in one shape.
func serverTarget(s serverset.Server) target.TargetSpec {
	spec := target.TargetSpec{Kind: target.TargetLocal, DataRoot: s.DataRoot}
	if s.IsRemote() {
		spec.Kind = target.TargetRemote
		spec.Host = s.Host
		spec.User = s.SSH.User
		spec.Port = s.SSH.Port
	}
	return spec
}

// fleetTarget describes a fleet's data plane. Its host is the first server's:
// the per-node addresses live on the node table, which the allocator fills.
func fleetTarget(pl serverset.Placement) target.TargetSpec {
	spec := target.TargetSpec{Kind: target.TargetLocal, DataRoot: pl.DataRoot}
	if pl.Remote {
		spec.Kind = target.TargetRemote
		if len(pl.Config.Hosts) > 0 {
			spec.Host = pl.Config.Hosts[0]
		}
	}
	return spec
}
