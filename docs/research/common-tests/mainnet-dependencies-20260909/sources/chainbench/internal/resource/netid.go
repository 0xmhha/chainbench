// Package netid resolves and validates the devp2p network id every node of a
// chain must run with. It exists because the network id is a silent-failure
// trap: go-wemix defaults it to 1111 (Wemix mainnet) independent of the chain
// id, while go-wbft defaults it to the chain id, so two binaries meant to form
// one chain will refuse to peer unless the same network id is set explicitly on
// both. There is deliberately no default here — the value comes from the
// manifest so a run's network id is always traceable.
package resource

import (
	"fmt"
	"strconv"
)

// Resolve validates the manifest-declared network id and returns it. It errors
// rather than substituting a default, so a missing value fails loudly.
func Resolve(networkID int64) (int64, error) {
	if networkID <= 0 {
		return 0, fmt.Errorf("netid: network id must be set explicitly (>0), got %d", networkID)
	}
	return networkID, nil
}

// Flag returns the launch flag that pins a node to networkID.
func Flag(networkID int64) []string {
	return []string{"--networkid", strconv.FormatInt(networkID, 10)}
}

// ValidateUniform confirms every node in a network is configured with the same
// (valid) network id — the condition for them to peer.
func ValidateUniform(ids []int64) error {
	if len(ids) == 0 {
		return fmt.Errorf("netid: no network ids to validate")
	}
	first := ids[0]
	for i, id := range ids {
		if id <= 0 {
			return fmt.Errorf("netid: node %d has invalid network id %d", i, id)
		}
		if id != first {
			return fmt.Errorf("netid: network id mismatch (node 0 = %d, node %d = %d); nodes will not peer", first, i, id)
		}
	}
	return nil
}
