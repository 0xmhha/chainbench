package serverset

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/netmap"
)

// The inventory's whole point downstream is this file: it turns a server entry
// into the allocator inputs, so a caller composing a network says "place these
// nodes on this server" and never branches on local vs remote. The difference
// survives only as the mode and the host the allocator hands back.

// BuiltinPorts is the port plan used when no inventory names one. It exists so
// a developer can compose a network without writing a config first; anything
// site-specific belongs in the inventory, which overrides these.
func BuiltinPorts() Ports {
	return Ports{P2PBase: builtinP2PBase, P2PStep: builtinP2PStep, RPCBase: builtinRPCBase, RPCStep: builtinRPCStep}
}

// Built-in port plan. The steps satisfy the same floors the inventory is
// validated against, and the rpc step leaves room for metrics.
const (
	builtinP2PBase = 31000
	builtinP2PStep = 10
	builtinRPCBase = 8600
	builtinRPCStep = 10
)

// builtinSource is what Placement.Source reports when no inventory was loaded.
const builtinSource = "built-in defaults (no server config)"

// Placement is the resource a network is composed on plus where that came
// from, so a surface can report which file decided the ports rather than
// leaving an operator to guess.
type Placement struct {
	// Pool is the resource netmap allocates from. It is what new callers use;
	// Config/Mode/Capacity describe the same thing for the allocator being
	// retired, and go with it.
	Pool netmap.Pool
	// DataRoot is where the nodes' data plane lives on the target.
	DataRoot string
	// Remote reports whether the nodes run over SSH.
	Remote bool
	// Source names the origin of the port plan for display.
	Source string
}

// Builtin is the placement used with no inventory: this machine, stepped ports,
// no data root of its own (the caller's workspace decides).
func Builtin(minValidators, portBand int) Placement {
	return Placement{
		Pool:   BuiltinPool(portBand),
		Source: builtinSource,
	}
}

// Placement resolves one server into the pool a network is allocated from.
// Local and remote read the same fields: the difference survives as the address
// and as whether the data plane is reached over SSH, not as a mode.
func (c *Config) Placement(s Server, minValidators, portBand int) Placement {
	source := fmt.Sprintf("%s[%s]", c.path, s.label(0))
	return Placement{
		Pool: netmap.Pool{
			Hosts:  []netmap.Host{{Name: s.Name, Addr: s.Host}},
			Slots:  s.Slots,
			Ports:  bandsOf(s.Ports),
			Source: source,
		},
		DataRoot: s.DataRoot,
		Remote:   s.IsRemote(),
		Source:   source,
	}
}

// Fleet resolves every remote server into one placement, for a network spread
// one node per host. It errors on a mixed inventory: a network half on this
// machine and half over SSH has two port regimes at once, and the allocator
// cannot express that.
func (c *Config) Fleet(minValidators, portBand int) (Placement, error) {
	if len(c.Servers) == 0 {
		return Placement{}, fmt.Errorf("serverset: %s configures no servers", c.path)
	}
	first := c.resolve(c.Servers[0])
	hosts := make([]string, 0, len(c.Servers))
	slots := 0
	for _, raw := range c.Servers {
		s := c.resolve(raw)
		if s.IsRemote() != first.IsRemote() {
			return Placement{}, fmt.Errorf(
				"serverset: %s mixes local and remote servers — a fleet must be all one kind", c.path)
		}
		hosts = append(hosts, s.Host)
		slots += s.Slots
	}
	fleetSource := fmt.Sprintf("%s[fleet of %d]", c.path, len(hosts))
	poolHosts := make([]netmap.Host, 0, len(c.Servers))
	for _, raw := range c.Servers {
		srv := c.resolve(raw)
		poolHosts = append(poolHosts, netmap.Host{Name: srv.Name, Addr: srv.Host})
	}
	return Placement{
		Pool: netmap.Pool{
			Hosts:  poolHosts,
			Slots:  slots / len(hosts),
			Ports:  bandsOf(first.Ports),
			Source: fleetSource,
		},
		DataRoot: first.DataRoot,
		Remote:   first.IsRemote(),
		Source:   fleetSource,
	}, nil
}

// bandsOf converts an inventory port plan into the bands netmap steps through.
func bandsOf(p Ports) netmap.Bands {
	return netmap.Bands{P2PBase: p.P2PBase, P2PStep: p.P2PStep, RPCBase: p.RPCBase, RPCStep: p.RPCStep}
}
