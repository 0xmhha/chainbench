package serverset

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/place"
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

// Placement is everything the allocator needs plus where it came from, so a
// surface can report which file decided the ports rather than leaving an
// operator to guess.
type Placement struct {
	// Pool is the resource netmap allocates from. It is what new callers use;
	// Config/Mode/Capacity describe the same thing for the allocator being
	// retired, and go with it.
	Pool netmap.Pool
	// Config and Mode drive place.New / Allocate.
	Config place.Config
	Mode   place.Mode
	// Capacity bounds how many nodes may be placed.
	Capacity place.Capacity
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
	p := BuiltinPorts()
	return Placement{
		Pool:     BuiltinPool(portBand),
		Config:   place.Config{P2PBase: p.P2PBase, P2PStep: p.P2PStep, RPCBase: p.RPCBase, RPCStep: p.RPCStep},
		Mode:     place.LocalStepped,
		Capacity: place.Capacity{MinValidators: minValidators, PortBandSize: portBand},
		Source:   builtinSource,
	}
}

// Placement resolves one server into allocator inputs. A local server hosts
// several nodes on stepped ports; a remote one hosts its slots' worth on the
// same ports as its peers, which is why the mode differs while everything else
// is read from the same fields.
func (c *Config) Placement(s Server, minValidators, portBand int) Placement {
	mode := place.LocalStepped
	if s.IsRemote() {
		mode = place.RemotePerHost
	}
	source := fmt.Sprintf("%s[%s]", c.path, s.label(0))
	return Placement{
		Pool: netmap.Pool{
			Hosts:  []netmap.Host{{Name: s.Name, Addr: s.Host}},
			Slots:  s.Slots,
			Ports:  bandsOf(s.Ports),
			Source: source,
		},
		Config: place.Config{
			P2PBase: s.Ports.P2PBase, P2PStep: s.Ports.P2PStep,
			RPCBase: s.Ports.RPCBase, RPCStep: s.Ports.RPCStep,
			Hosts: []string{s.Host},
		},
		Mode: mode,
		Capacity: place.Capacity{
			MinValidators: minValidators,
			Hosts:         1,
			SlotsPerHost:  s.Slots,
			PortBandSize:  portBand,
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
	mode := place.LocalStepped
	if first.IsRemote() {
		mode = place.RemotePerHost
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
		Config: place.Config{
			P2PBase: first.Ports.P2PBase, P2PStep: first.Ports.P2PStep,
			RPCBase: first.Ports.RPCBase, RPCStep: first.Ports.RPCStep,
			Hosts: hosts,
		},
		Mode: mode,
		Capacity: place.Capacity{
			MinValidators: minValidators,
			Hosts:         len(hosts),
			SlotsPerHost:  slots / len(hosts),
			PortBandSize:  portBand,
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
