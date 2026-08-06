package place

import (
	"fmt"
	"net"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

const localHost = "127.0.0.1"

// Config holds the inputs an Allocator needs that are not per-request: the
// local port bands (reused by portplan) and, for RemotePerHost, the ordered
// server IPs.
type Config struct {
	P2PBase int
	P2PStep int
	RPCBase int
	RPCStep int
	Hosts   []string
}

// allocator is the concrete Allocator over a fixed Config.
type allocator struct {
	cfg Config
}

// New returns an Allocator bound to cfg.
func New(cfg Config) Allocator { return &allocator{cfg: cfg} }

// Allocate validates capacity (fail-fast) then resolves placements for the mode.
func (a *allocator) Allocate(reqs []NodeReq, mode Mode, capacity Capacity) ([]NodePlacement, error) {
	if err := a.checkCapacity(reqs, mode, capacity); err != nil {
		return nil, err
	}
	switch mode {
	case LocalStepped:
		return a.localStepped(reqs)
	case LocalOSAssigned:
		return a.localOSAssigned(reqs)
	case RemotePerHost:
		return a.remotePerHost(reqs)
	default:
		return nil, fmt.Errorf("place: unknown mode %d", mode)
	}
}

// checkCapacity enforces the BFT minimum and the per-mode maximum before any
// placement is produced, so an over-sized request fails cheaply.
func (a *allocator) checkCapacity(reqs []NodeReq, mode Mode, capacity Capacity) error {
	if capacity.MinValidators > 0 {
		v := 0
		for _, r := range reqs {
			if r.Role == node.RoleValidator {
				v++
			}
		}
		if v < capacity.MinValidators {
			return fmt.Errorf("place: %d validators below BFT minimum %d", v, capacity.MinValidators)
		}
	}
	switch mode {
	case RemotePerHost:
		if len(reqs) > len(a.cfg.Hosts) {
			return fmt.Errorf("place: %d nodes exceed %d remote hosts (one node per host)", len(reqs), len(a.cfg.Hosts))
		}
	case LocalStepped, LocalOSAssigned:
		if capacity.PortBandSize > 0 && len(reqs) > capacity.PortBandSize {
			return fmt.Errorf("place: %d nodes exceed local port band %d", len(reqs), capacity.PortBandSize)
		}
	}
	return nil
}

// localStepped assigns deterministic stepped ports on the loopback host.
func (a *allocator) localStepped(reqs []NodeReq) ([]NodePlacement, error) {
	out := make([]NodePlacement, 0, len(reqs))
	allPorts := make([]portplan.Ports, 0, len(reqs))
	for i, r := range reqs {
		p, err := portplan.Plan(i+1, a.cfg.P2PBase, a.cfg.P2PStep, a.cfg.RPCBase, a.cfg.RPCStep)
		if err != nil {
			return nil, fmt.Errorf("place: %s: %w", r.Name, err)
		}
		allPorts = append(allPorts, p)
		out = append(out, NodePlacement{Name: r.Name, Host: localHost, Ports: p})
	}
	if err := portplan.Validate(allPorts); err != nil {
		return nil, fmt.Errorf("place: %w", err)
	}
	return out, nil
}

// localOSAssigned lets the OS pick free port blocks (avoiding fixed-port
// double-binds on back-to-back runs). Each node reserves a contiguous P2P block
// (p2p, etcd=p2p+1) and RPC block (http, ws=http+1, auth=http+2). All reserving
// listeners are held until every node is placed, so no two nodes — nor any
// derived port — collide; they are released just before the nodes bind.
func (a *allocator) localOSAssigned(reqs []NodeReq) ([]NodePlacement, error) {
	var held []net.Listener
	defer func() {
		for _, l := range held {
			_ = l.Close()
		}
	}()

	out := make([]NodePlacement, 0, len(reqs))
	allPorts := make([]portplan.Ports, 0, len(reqs))
	for _, r := range reqs {
		p2pLs, p2p, err := reserveContiguous(2) // p2p, etcd
		if err != nil {
			return nil, fmt.Errorf("place: %s: %w", r.Name, err)
		}
		held = append(held, p2pLs...)
		httpLs, http, err := reserveContiguous(3) // http, ws, auth
		if err != nil {
			return nil, fmt.Errorf("place: %s: %w", r.Name, err)
		}
		held = append(held, httpLs...)

		p := portplan.Ports{P2P: p2p, Etcd: p2p + 1, HTTP: http, WS: http + 1, Auth: http + 2}
		allPorts = append(allPorts, p)
		out = append(out, NodePlacement{Name: r.Name, Host: localHost, Ports: p})
	}
	if err := portplan.Validate(allPorts); err != nil {
		return nil, fmt.Errorf("place: OS-assigned ports collided: %w", err)
	}
	return out, nil
}

// remotePerHost gives every node the same base ports on a distinct server IP.
func (a *allocator) remotePerHost(reqs []NodeReq) ([]NodePlacement, error) {
	base, err := portplan.Plan(1, a.cfg.P2PBase, a.cfg.P2PStep, a.cfg.RPCBase, a.cfg.RPCStep)
	if err != nil {
		return nil, fmt.Errorf("place: %w", err)
	}
	out := make([]NodePlacement, 0, len(reqs))
	for i, r := range reqs {
		out = append(out, NodePlacement{Name: r.Name, Host: a.cfg.Hosts[i], Ports: base})
	}
	return out, nil
}

// reserveContiguous finds count consecutive free loopback ports and returns
// the held listeners (kept open by the caller) plus the base port. The OS picks
// the base via :0; the remaining ports are probed explicitly, retrying with a
// fresh base if the block is not fully free.
func reserveContiguous(count int) ([]net.Listener, int, error) {
	const attempts = 20
	for try := 0; try < attempts; try++ {
		first, err := net.Listen("tcp", localHost+":0")
		if err != nil {
			return nil, 0, err
		}
		base := first.Addr().(*net.TCPAddr).Port
		held := []net.Listener{first}
		ok := true
		for off := 1; off < count; off++ {
			l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", localHost, base+off))
			if err != nil {
				ok = false
				break
			}
			held = append(held, l)
		}
		if ok {
			return held, base, nil
		}
		for _, l := range held {
			_ = l.Close()
		}
	}
	return nil, 0, fmt.Errorf("place: could not reserve %d contiguous free ports", count)
}
