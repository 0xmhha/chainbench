package inspector

import (
	"context"
	"net"
	"sort"
	"strconv"
)

// Host is one machine a plan expects to reach, and the port that proves it —
// the port a login goes to, not a node port: a host is reachable when its
// door answers, whether or not any node is listening yet.
type Host struct {
	Name string
	Addr string
	Port int
}

// HostPort is the dial target.
func (h Host) HostPort() string { return net.JoinHostPort(h.Addr, strconv.Itoa(h.Port)) }

// Hosts reports which of the given hosts cannot be reached: a dial to the
// host's door that does not connect within the probe timeout. This machine is
// always reachable and is not dialled.
func Hosts(ctx context.Context, hosts []Host, dial DialFunc) []Host {
	if dial == nil {
		dial = defaultDial
	}
	var down []Host
	for _, h := range hosts {
		if isLocal(h.Addr) || h.Port <= 0 {
			continue
		}
		conn, err := dial(ctx, h.HostPort())
		if err != nil {
			down = append(down, h)
			continue
		}
		_ = conn.Close()
	}
	sort.Slice(down, func(i, j int) bool { return down[i].Addr < down[j].Addr })
	return down
}
