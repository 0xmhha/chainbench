package resource

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

// The server set's whole point downstream is this file: it turns a server entry
// into the allocator inputs, so a caller composing a network says "place these
// nodes on this server" and never branches on local vs remote. The difference
// survives only as the mode and the host the allocator hands back.

// BuiltinPorts is the port plan used when no server set names one. It exists so
// a developer can compose a network without writing a config first; anything
// site-specific belongs in the server set, which overrides these.
func BuiltinPorts() Ports {
	return Ports{P2PBase: builtinP2PBase, P2PStep: builtinP2PStep, RPCBase: builtinRPCBase, RPCStep: builtinRPCStep}
}

// Built-in port plan. The steps satisfy the same floors the server set is
// validated against, and the rpc step leaves room for metrics.
const (
	builtinP2PBase = 31000
	builtinP2PStep = 10
	builtinRPCBase = 8600
	builtinRPCStep = 10
)

// builtinSource is what a pool's Source reports when no server set was loaded.
const builtinSource = "built-in defaults (no server config)"

// Builtin is the pool used with no server set: this machine, stepped ports. It
// names itself as its own source so port provenance stays visible.
func Builtin(minValidators, portBand int) netmap.Pool {
	return BuiltinPool(portBand)
}

// PoolFor resolves one server into the pool a network is allocated from. Local
// and remote read the same fields: the difference survives as the address and
// as whether the data plane is reached over SSH, not as a mode.
func (c *Set) PoolFor(s Server, minValidators, portBand int) netmap.Pool {
	return netmap.Pool{
		Hosts:  []netmap.Host{{Name: s.Name, Addr: s.Host}},
		Slots:  s.Slots,
		Ports:  bandsOf(s.Ports),
		Source: fmt.Sprintf("%s[%s]", c.path, s.label(0)),
	}
}

// Pool resolves the whole set into one pool, for a network spread one node per
// host.
//
// It refuses a set that mixes local and remote servers: that is two port
// regimes at once, and the allocator cannot express it. Slots need no such
// check — v2 declares one pool, so expand gives every host the same slot count
// (a per-host count would be a format change, tracked as P1.3 in
// docs/dev/architecture/module-plan.md). The count is read from the first
// server rather than summed and divided, which is what the retired whole-set
// resolver did.
func (c *Set) Pool(minValidators, portBand int) (netmap.Pool, error) {
	if len(c.Servers) == 0 {
		return netmap.Pool{}, fmt.Errorf("resource: %s configures no servers", c.path)
	}
	first := c.resolve(c.Servers[0])
	hosts := make([]netmap.Host, 0, len(c.Servers))
	for _, raw := range c.Servers {
		s := c.resolve(raw)
		if s.IsRemote() != first.IsRemote() {
			return netmap.Pool{}, fmt.Errorf(
				"resource: %s mixes local and remote servers — every server must be the same kind", c.path)
		}
		hosts = append(hosts, netmap.Host{Name: s.Name, Addr: s.Host})
	}
	return netmap.Pool{
		Hosts:  hosts,
		Slots:  first.Slots,
		Ports:  bandsOf(first.Ports),
		Source: fmt.Sprintf("%s[%d servers]", c.path, len(hosts)),
	}, nil
}

// bandsOf converts a server-set port plan into the bands netmap steps through.
func bandsOf(p Ports) netmap.Bands {
	b := netmap.Bands{P2PBase: p.P2PBase, P2PStep: p.P2PStep, RPCBase: p.RPCBase, RPCStep: p.RPCStep}
	if p.WS != nil {
		b.WS = &portplan.Band{Base: p.WS.Base, Step: p.WS.Step}
	}
	if p.Auth != nil {
		b.Auth = &portplan.Band{Base: p.Auth.Base, Step: p.Auth.Step}
	}
	if p.Metrics != nil {
		b.Metrics = &portplan.Band{Base: p.Metrics.Base, Step: p.Metrics.Step}
	}
	return b
}
