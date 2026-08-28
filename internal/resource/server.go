package resource

import (
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/machine"
)

// Selecting a server: which entry of the set a composition runs on, and what
// that resolves to — the resource to allocate from, and the machine the data
// plane lives on.

// ServerRef selects a server from a server set.
type ServerRef struct {
	// SetPath is the server-set file; empty uses DefaultSetFile when it
	// exists, and otherwise no server set at all.
	SetPath string `json:"setPath,omitempty"`
	// Name and Index select within the server set. Both empty picks the only
	// server in a single-server file.
	Name  string `json:"name,omitempty"`
	Index int    `json:"index,omitempty"`
	// All spreads the network across every server in the set, one node per
	// host, instead of placing it on one.
	All bool `json:"all,omitempty"`
}

// ResolveServerOut is what a composition needs from the server set.
type ResolveServerOut struct {
	// Pool is the resource to allocate from: which hosts, how many slots, and
	// the port bands each slot steps through.
	Pool Pool
	// Target is where the data plane lives. It is the zero value when no
	// server set applied, leaving the workspace's own target in place.
	Target machine.Spec
	// HasTarget reports whether Target should replace the workspace's.
	HasTarget bool
}

// ResolveServer turns a server selection into the resource to allocate from
// and the machine the data plane lives on. With no selection and no default
// server set on disk it returns the built-in pool, which names itself as such
// so the port provenance stays visible.
func ResolveServer(ref ServerRef, minValidators, portBand int) (ResolveServerOut, error) {
	path := ref.SetPath
	if path == "" {
		// A server set at the default path is used when present, but its
		// absence is only an error if the caller named a server — or if the
		// file still sits there under its pre-rename name, which must not
		// silently degrade a configured placement to the built-in local one.
		if _, err := os.Stat(DefaultSetFile); err != nil {
			if hint := LegacyNameHint(DefaultSetFile); hint != "" {
				return ResolveServerOut{}, fmt.Errorf("resource: %s", hint)
			}
			if ref.Name != "" || ref.Index != 0 || ref.All {
				return ResolveServerOut{}, fmt.Errorf(
					"resource: --server needs a server set: %s not found (copy %s)",
					DefaultSetFile, DefaultSampleFile)
			}
			return ResolveServerOut{Pool: Builtin(minValidators, portBand)}, nil
		}
		path = DefaultSetFile
	}
	cfg, err := LoadSet(path)
	if err != nil {
		return ResolveServerOut{}, err
	}
	if ref.All {
		pool, err := cfg.Pool(minValidators, portBand)
		if err != nil {
			return ResolveServerOut{}, err
		}
		return ResolveServerOut{Pool: pool, Target: cfg.setTarget(), HasTarget: true}, nil
	}
	srv, err := cfg.Select(ref.Name, ref.Index)
	if err != nil {
		return ResolveServerOut{}, err
	}
	return ResolveServerOut{
		Pool:      cfg.PoolFor(srv, minValidators, portBand),
		Target:    serverTarget(srv),
		HasTarget: true,
	}, nil
}

// serverTarget describes where one server's data plane lives. A remote server
// is recorded as a KindServer spec naming its set entry — not flattened to a
// host/user pair — so every later step resolves the login from the server set
// file, the single source of a named server's credentials. Host is still
// carried for display and RPC addressing; it never authenticates anything.
func serverTarget(s Server) machine.Spec {
	spec := machine.Spec{DataRoot: s.DataRoot}
	if s.IsRemote() {
		spec.Server = s.Name
		spec.Host = s.Host
	}
	return spec
}

// setTarget describes the whole set's data plane. Its host is the first
// server's: the per-node addresses live on the node table, which the allocator
// fills.
func (c *Set) setTarget() machine.Spec {
	if len(c.Servers) == 0 {
		return machine.Spec{}
	}
	return serverTarget(c.resolve(c.Servers[0]))
}
