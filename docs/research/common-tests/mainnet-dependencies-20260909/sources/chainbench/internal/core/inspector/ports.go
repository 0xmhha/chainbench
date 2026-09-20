package inspector

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"
)

// probeTimeout bounds a single dial. It is short: a listening socket on a
// reachable host answers immediately, and anything slower is indistinguishable
// from a closed port for this purpose.
const probeTimeout = 300 * time.Millisecond

// Addr is one address a plan intends to listen on, named by what it is for so a
// report can say "node2 http" rather than a bare number.
type Addr struct {
	Host string
	Port int
	// Node is the 1-based node index the port belongs to.
	Node int
	// Purpose is the port's role in the node ("p2p", "http", "etcd", ...).
	Purpose string
}

// String renders an address the way an operator reads it.
func (a Addr) String() string {
	return fmt.Sprintf("%s:%d (node%d %s)", a.Host, a.Port, a.Node, a.Purpose)
}

// HostPort is the dial target.
func (a Addr) HostPort() string { return net.JoinHostPort(a.Host, strconv.Itoa(a.Port)) }

// DialFunc opens a connection to a host:port. Injected so the scan is testable
// without binding real ports, and so a remote target can supply its own reach.
type DialFunc func(ctx context.Context, hostPort string) (net.Conn, error)

// isLocal reports whether an address is this machine, where a stronger check
// than dialling is available.
func isLocal(host string) bool {
	switch host {
	case "", "127.0.0.1", "localhost", "::1", "0.0.0.0":
		return true
	}
	return false
}

// localBusy reports whether a local port is taken, by trying to take it two
// ways.
//
// Dialling is the wrong question locally: a node that binds a wildcard socket
// need not answer a loopback connection, and lsof showed *:8600 held while a
// dial to 127.0.0.1:8600 was refused. Binding asks the kernel the question the
// launch is about to ask — but which bind matters, because listeners are opened
// with SO_REUSEADDR and the two forms miss opposite cases. Measured against a
// running network:
//
//	port 8600, held on the wildcard : bind 127.0.0.1 SUCCEEDS, bind : fails
//	port 8603, held on loopback     : bind 127.0.0.1 fails,    bind : SUCCEEDS
//
// So either bind alone reports a busy port free. A port is free only when both
// succeed.
func localBusy(host string, port int) bool {
	for _, addr := range []string{net.JoinHostPort(host, strconv.Itoa(port)), ":" + strconv.Itoa(port)} {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return true
		}
		_ = ln.Close()
	}
	return false
}

// Scan reports which of the given addresses already have a listener.
//
// A successful connection is the evidence. Not being able to connect is not
// reported as free-with-certainty and not reported as an error either: a
// refused connection means nothing is there, and anything else (a filtered
// port, an unreachable host) is a fact about reachability rather than about
// occupancy, which the caller learns soon enough when it tries to launch.
func Ports(ctx context.Context, addrs []Addr, dial DialFunc) []Addr {
	injected := dial != nil
	if dial == nil {
		dial = defaultDial
	}
	var busy []Addr
	for _, a := range addrs {
		if a.Port <= 0 {
			continue
		}
		if isLocal(a.Host) && !injected {
			if localBusy(a.Host, a.Port) {
				busy = append(busy, a)
			}
			continue
		}
		conn, err := dial(ctx, a.HostPort())
		if err != nil {
			continue
		}
		_ = conn.Close()
		busy = append(busy, a)
	}
	sort.Slice(busy, func(i, j int) bool {
		if busy[i].Node != busy[j].Node {
			return busy[i].Node < busy[j].Node
		}
		return busy[i].Port < busy[j].Port
	})
	return busy
}

// defaultDial is a plain TCP dial with a short timeout.
func defaultDial(ctx context.Context, hostPort string) (net.Conn, error) {
	d := net.Dialer{Timeout: probeTimeout}
	return d.DialContext(ctx, "tcp", hostPort)
}
