// Package portplan computes each node's full set of listening ports from two
// disjoint bands, and validates that nothing collides. It encodes a rule that
// caused repeated silent failures: the wemix binary derives its etcd port as
// p2p_port + 1, so if p2p ports are packed one apart, or if the p2p band
// overlaps the RPC band, etcd fails to bind ("address already in use") and
// block production stalls with no obvious cause. Ports come from the profile
// (bands + steps); there is no code default.
package portplan

import "fmt"

// Ports is one node's listening ports. Etcd is not a launch flag — it is
// derived by the wemix binary as P2P+1 and is tracked here only so collisions
// are caught up front.
type Ports struct {
	P2P  int
	Etcd int // = P2P + 1 (wemix), reserved so no other service may use it
	HTTP int
	WS   int // = HTTP + 1
	Auth int // = HTTP + 2 (engine auth-rpc)
	// Metrics is the node's metrics endpoint (= HTTP + 3). It is only
	// assigned when the rpc step leaves room for it (rpcStep >= 4); a plan
	// with a tighter step simply has no metrics port (0), which downstream
	// treats as metrics-off rather than an error, so existing 3-step profiles
	// keep working.
	Metrics int
}

// Plan computes node index's ports (index is 1-based). p2p ports advance by
// p2pStep from p2pBase (etcd = p2p+1); rpc ports advance by rpcStep from
// rpcBase (ws = http+1, auth = http+2). It requires p2pStep >= 2 (to leave room
// for etcd) and rpcStep >= 3 (http, ws, auth).
func Plan(index, p2pBase, p2pStep, rpcBase, rpcStep int) (Ports, error) {
	if index < 1 {
		return Ports{}, fmt.Errorf("portplan: node index must be >= 1, got %d", index)
	}
	if p2pStep < 2 {
		return Ports{}, fmt.Errorf("portplan: p2p_step must be >= 2 to reserve etcd=p2p+1, got %d", p2pStep)
	}
	if rpcStep < 3 {
		return Ports{}, fmt.Errorf("portplan: rpc_step must be >= 3 for http/ws/auth, got %d", rpcStep)
	}
	p2p := p2pBase + (index-1)*p2pStep
	http := rpcBase + (index-1)*rpcStep
	p := Ports{P2P: p2p, Etcd: p2p + 1, HTTP: http, WS: http + 1, Auth: http + 2}
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
