package node

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

// WithDefaults fills a zero reservation, so a caller may pass one it has not
// filled in and still get the historical behaviour.
func (r Reservation) WithDefaults() Reservation {
	if r.P2PSpan < 1 {
		r.P2PSpan = DefaultReservation.P2PSpan
	}
	if r.RPCSpan < 1 {
		r.RPCSpan = DefaultReservation.RPCSpan
	}
	return r
}
