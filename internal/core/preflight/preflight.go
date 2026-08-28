// Package preflight composes the atomic setup checks into one gate that runs
// before a network launches. Every condition here caused a silent, hard-to-
// diagnose failure at least once (mismatched network id, colliding etcd/RPC
// ports, a go-wbft validator wrongly listed as a wemix member, a genesis
// missing the croissant section or petersburg fork). Failing loudly up front,
// from an explicit plan, is the whole point.
package preflight

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/netid"
)

// NetworkPlan is the fully-resolved description of a network about to launch.
// All values are explicit (from manifest + profile) so a failure names exactly
// what was configured.
type NetworkPlan struct {
	// NetworkIDs is each node's configured devp2p network id.
	NetworkIDs []int64
	// Ports is each node's resolved port set.
	Ports []node.Endpoints
	// Genesis is the genesis bytes every node initializes from (identical).
	Genesis []byte
	// WemixMembers are the addresses registered as wemix+etcd producers.
	WemixMembers []string
	// WbftValidators are the croissant.init post-fork validator addresses.
	// For an upgrade network these must be disjoint from WemixMembers.
	WbftValidators []string
}

// Validate runs every atomic check plus the cross-cutting role rule. It returns
// the first violation.
func Validate(p NetworkPlan) error {
	if err := netid.ValidateUniform(p.NetworkIDs); err != nil {
		return err
	}
	if err := resource.ValidatePorts(p.Ports); err != nil {
		return err
	}
	if len(p.NetworkIDs) != len(p.Ports) {
		return fmt.Errorf("preflight: %d network ids but %d port sets", len(p.NetworkIDs), len(p.Ports))
	}
	if len(p.Genesis) > 0 {
		if err := genesis.ValidateForks(p.Genesis); err != nil {
			return err
		}
	}
	// A go-wbft validator listed as a wemix member registers in governance but
	// never joins etcd (it runs wbft), shows as "down", and stalls the producer.
	member := map[string]bool{}
	for _, a := range p.WemixMembers {
		member[strings.ToLower(a)] = true
	}
	for _, v := range p.WbftValidators {
		if member[strings.ToLower(v)] {
			return fmt.Errorf("preflight: address %s is both a wemix member and a wbft validator; keep validators out of the wemix member list", v)
		}
	}
	return nil
}
