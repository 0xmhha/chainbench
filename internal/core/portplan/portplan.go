// Package portplan computes each node's full set of listening ports from two
// disjoint bands, and validates that nothing collides. It encodes a rule that
// caused repeated silent failures: the wemix binary derives its etcd port as
// p2p_port + 1, so if p2p ports are packed one apart, or if the p2p band
// overlaps the RPC band, etcd fails to bind ("address already in use") and
// block production stalls with no obvious cause. Ports come from the profile
// (bands + steps); there is no code default.
package portplan

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Ports is one node's listening ports: P2P (with Etcd derived at P2P+1), HTTP,
// WS (HTTP+1), Auth (HTTP+2) and Metrics (HTTP+3, only when the rpc step
// leaves room — a tighter step simply has no metrics port, which downstream
// reads as metrics-off rather than an error).
//
// It is an alias, not a copy. There used to be three representations of a
// node's ports and two of them dropped the etcd port on the way to runtime, so
// a live wemix node could not be asked which port its etcd was on — the one
// port whose collision rule exists because the binary derives it. One type,
// living in the vocabulary package, is what stops that happening again.
type Ports = node.Endpoints

// Reservation is how many consecutive ports one node needs from each band. It
// is a family fact, not a global one: the wbft binaries listen on p2p alone,
// while a wemix node's embedded etcd takes two more — peer at p2p+1 and client
// at p2p+2 (go-wemix wemix/etcdutil.go). A step smaller than the span puts the
// next node's p2p port on top of this node's etcd, and the chain stalls with
// no obvious cause, which is the failure this type exists to make impossible.
type Reservation struct {
	// P2PSpan is how many ports the p2p side consumes, p2p included.
	P2PSpan int
	// RPCSpan is how many the rpc side consumes, http included.
	RPCSpan int
}

// DefaultReservation is what a caller that has not asked a family gets: room
// for the etcd peer port, which is what every plan reserved before families
// could speak for themselves.
var DefaultReservation = Reservation{P2PSpan: 2, RPCSpan: 3}

// withDefaults fills a zero reservation, so a caller may pass one it has not
// filled in and still get the historical behaviour.
func (r Reservation) withDefaults() Reservation {
	if r.P2PSpan < 1 {
		r.P2PSpan = DefaultReservation.P2PSpan
	}
	if r.RPCSpan < 1 {
		r.RPCSpan = DefaultReservation.RPCSpan
	}
	return r
}

// Plan computes node index's ports (index is 1-based). p2p ports advance by
// p2pStep from p2pBase; rpc ports advance by rpcStep from rpcBase (ws = http+1,
// auth = http+2, metrics = http+3 when the step allows). The reservation says
// how much room each band must leave, and which derived ports exist at all.
func Plan(index, p2pBase, p2pStep, rpcBase, rpcStep int, res Reservation) (Ports, error) {
	if index < 1 {
		return Ports{}, fmt.Errorf("portplan: node index must be >= 1, got %d", index)
	}
	res = res.withDefaults()
	if p2pStep < res.P2PSpan {
		return Ports{}, fmt.Errorf("portplan: p2p_step must be >= %d for this chain family (it reserves %d consecutive p2p-side ports), got %d",
			res.P2PSpan, res.P2PSpan, p2pStep)
	}
	if rpcStep < res.RPCSpan {
		return Ports{}, fmt.Errorf("portplan: rpc_step must be >= %d for http/ws/auth, got %d", res.RPCSpan, rpcStep)
	}
	p2p := p2pBase + (index-1)*p2pStep
	http := rpcBase + (index-1)*rpcStep
	p := Ports{P2P: p2p, HTTP: http, WS: http + 1, Auth: http + 2}
	if res.P2PSpan >= 2 {
		p.Etcd = p2p + 1
	}
	if res.P2PSpan >= 3 {
		p.EtcdClient = p2p + 2
	}
	if rpcStep >= 4 {
		p.Metrics = http + 3
	}
	return p, nil
}

// Validate confirms no two ports collide across the whole network, including
// the derived etcd ports. Any duplicate is a silent bind failure waiting to
// happen, so this errors.
func Validate(ports []Ports) error {
	seen := map[int]string{}
	claim := func(p int, who string) error {
		if prev, ok := seen[p]; ok {
			return fmt.Errorf("portplan: port %d used by both %s and %s", p, prev, who)
		}
		seen[p] = who
		return nil
	}
	for i, n := range ports {
		for _, c := range []struct {
			port int
			name string
		}{
			{n.P2P, fmt.Sprintf("node%d.p2p", i+1)},
			{n.Etcd, fmt.Sprintf("node%d.etcd", i+1)},
			{n.HTTP, fmt.Sprintf("node%d.http", i+1)},
			{n.WS, fmt.Sprintf("node%d.ws", i+1)},
			{n.Auth, fmt.Sprintf("node%d.auth", i+1)},
			{n.Metrics, fmt.Sprintf("node%d.metrics", i+1)},
		} {
			// 0 = unassigned (metrics on a tight rpc step); only assigned
			// ports can collide. Plan never yields 0 for the required ports.
			if c.port == 0 {
				continue
			}
			if err := claim(c.port, c.name); err != nil {
				return err
			}
		}
	}
	return nil
}
