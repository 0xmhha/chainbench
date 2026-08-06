package place

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

// Mode selects the addressing and port strategy.
type Mode int

const (
	// LocalStepped assigns deterministic stepped ports on 127.0.0.1.
	LocalStepped Mode = iota
	// LocalOSAssigned binds :0 then reclaims, avoiding fixed-port double-binds.
	LocalOSAssigned
	// RemotePerHost uses identical ports on distinct server IPs (one node per host).
	RemotePerHost
)

// NodeReq is one node's placement request.
type NodeReq struct {
	Name   string
	Role   node.Role
	Sync   string
	Binary string
}

// NodePlacement is the resolved host and reserved ports for one node. Ports is
// portplan.Ports, which reserves the wemix-derived etcd port (P2P+1).
type NodePlacement struct {
	Name     string
	Host     string
	Ports    portplan.Ports
	DataPath string
}

// Capacity bounds the node count: the minimum validators for a BFT quorum and
// the maximum the target can host (local port band, or remote hosts x slots).
type Capacity struct {
	MinValidators int
	Hosts         int
	SlotsPerHost  int
	PortBandSize  int
}

// Allocator resolves node placements, validating capacity first (fail-fast):
// too few validators for a BFT quorum, or more nodes than the target can host,
// is an error returned before any placement is produced.
type Allocator interface {
	Allocate(reqs []NodeReq, mode Mode, capacity Capacity) ([]NodePlacement, error)
}
