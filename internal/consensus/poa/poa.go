// Package poa implements the "poa" consensus family used by the wemix chain
// (decision D9): etcd-based membership with governance contracts deployed by a
// boot node, consensus RPC under the "wemix" namespace. The etcd bootstrap
// sequence (boot node -> deploy governance -> init etcd -> start others, per
// ../script/wemix-upgrade) attaches to Bootstrap when the setup pipeline lands
// in phase G7; G0 provides the static, RPC-facing surface.
package poa

import (
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Family is the poa (wemix/etcd) consensus-family strategy.
type Family struct{}

// New returns the poa consensus family.
func New() Family { return Family{} }

func (Family) ID() string               { return "poa" }
func (Family) RPCNamespace() string     { return "wemix" }
func (Family) ValidatorsMethod() string { return "wemix_getValidators" }

// BuildGenesis renders the poa base genesis: the validator set is not in the
// genesis (membership is set at bootstrap), so only ChainID and Coinbase apply.
func (Family) BuildGenesis(template []byte, p registry.GenesisParams) ([]byte, error) {
	return BuildGenesis(template, p.ChainID, p.Coinbase)
}

// StartFlags returns the node launch flags for a role. The boot node performs
// governance deployment and etcd initialization out of band.
func (Family) StartFlags(role node.Role) []string {
	flags := []string{
		"--allow-insecure-unlock",
	}
	// Both spellings of the producing role seal (see the wbft family): a
	// producer launched without --mine stalls the chain, and which word the
	// composition recorded must not decide that.
	canonical, err := netmap.NormalizeRole(string(role))
	if err == nil && (canonical == node.RoleBP || canonical == node.RoleBoot) {
		flags = append(flags, "--mine")
	}
	return flags
}

// PortReservation: a poa node embeds etcd, which listens on two further ports
// — peer at p2p+1 and client at p2p+2 (go-wemix wemix/etcdutil.go). Three is
// the span, and a step of two — which the previous global rule accepted — puts
// the next node's p2p port on this node's etcd client.
func (Family) PortReservation() portplan.Reservation {
	return portplan.Reservation{P2PSpan: 3, RPCSpan: 3}
}

// SupportsRole: poa produces blocks and serves endpoints, and one producer
// carries the governance bootstrap. It has no proxy tier — etcd occupies that
// place — so a pn declared here is refused rather than quietly ignored.
func (Family) SupportsRole(role node.Role) bool {
	canonical, err := netmap.NormalizeRole(string(role))
	if err != nil {
		return false
	}
	switch canonical {
	case node.RoleBP, node.RoleEN, node.RoleBoot:
		return true
	default:
		return false
	}
}
