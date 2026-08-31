package deploy

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Deploy provisions and launches every server in the cluster over SSH, in launch
// order (endpoints/bootnodes before producers): it writes each node's config,
// ships + inits the genesis, and starts the node on its role binary. Keys are
// NOT shipped — the closed-network model reads them from the servers (see
// keys.go); the config points at the remote keystore. Returns the launched
// nodes. This runs against reachable remote servers (no-op-safe to build the
// specs first with BuildNodeSpecs / Describe for a dry run).
func Deploy(ctx context.Context, c *Cluster, cr *Credentials, hostKey remote.HostKeyCallback, genesis []byte, enodes []string, env func(string) string) ([]node.Node, error) {
	specs := BuildNodeSpecs(c, enodes)
	var nodes []node.Node
	for _, spec := range specs {
		s, ok := c.ServerByIndex(spec.Index)
		if !ok {
			return nodes, fmt.Errorf("deploy: no server for spec index %d", spec.Index)
		}
		rc, err := cr.For(c, s, env)
		if err != nil {
			return nodes, err
		}
		d := process.NewRemoteDriver(process.SSHRunner(rc, hostKey))

		if err := d.Provision(ctx, spec); err != nil {
			return nodes, fmt.Errorf("deploy: server %d provision: %w", spec.Index, err)
		}
		if len(genesis) > 0 {
			if err := d.InitDatadir(ctx, spec, genesis); err != nil {
				return nodes, fmt.Errorf("deploy: server %d init: %w", spec.Index, err)
			}
		}
		h, err := d.Launch(ctx, spec)
		if err != nil {
			return nodes, fmt.Errorf("deploy: server %d launch: %w", spec.Index, err)
		}
		nodes = append(nodes, node.Node{
			Index:  spec.Index,
			Role:   spec.Role,
			Host:   s.Host,
			RPCURL: c.RPCURL(s),
			Ports:  spec.Ports,
			PID:    h.PID,
		})
	}
	return nodes, nil
}
