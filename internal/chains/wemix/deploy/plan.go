package deploy

import (
	"fmt"
	"path"
	"strings"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
)

// wemixMinerRecommit is the go-wemix manifest's miner_recommit form (the older
// go-ethereum in go-wemix wants nanoseconds). Mirrors pkg/chains/wemix/manifest.json.
const wemixMinerRecommit = "nanos"

// NodeRole maps a cluster Role to a chainbench node.Role.
func NodeRole(r Role) node.Role {
	switch r {
	case RoleWemixBP, RoleWbftBP:
		return node.RoleValidator
	case RoleEndpoint:
		return node.RoleEndpoint
	case RoleBootnode:
		return node.RoleBoot
	default:
		return node.RoleEndpoint
	}
}

// dataRoot returns the cluster's remote node datadir (default /data/go-wbft).
func (c *Cluster) dataRoot() string {
	if c.DataRoot != "" {
		return c.DataRoot
	}
	return "/data/go-wbft"
}

// ports returns the port map for a server. All servers share the same ports —
// they are on distinct hosts.
func (c *Cluster) ports() node.Endpoints {
	p2p := c.P2PPort
	if p2p == 0 {
		p2p = 30303
	}
	return node.Endpoints{
		P2P:     p2p,
		HTTP:    c.RPCPort,
		WS:      c.WSPort,
		Auth:    c.RPCPort + 2,
		Metrics: 6060,
	}
}

// BuildNodeSpec maps a cluster server to a chainbench driver.NodeSpec: the
// remote binary (by role), datadir, ports, generated node config (poa/wemix
// namespace, optional static-nodes peering), and launch args. enodes is the
// static-nodes peer list (all servers; may be empty to rely on discovery).
func BuildNodeSpec(c *Cluster, s Server, enodes []string) driver.NodeSpec {
	ports := c.ports()
	role := NodeRole(s.Role)
	fam := poa.New()
	dd := c.dataRoot()
	configPath := path.Join(dd, "config.toml")
	cfg := nodeconfig.Generate(nodeconfig.Params{
		Role:          role,
		Ports:         ports,
		RPCNamespace:  fam.RPCNamespace(),
		MinerRecommit: wemixMinerRecommit,
		SyncMode:      c.SyncModeFor(s),
		StaticNodes:   enodes,
	})
	return driver.NodeSpec{
		Index:         s.Index,
		Role:          role,
		Host:          s.Host,
		Binary:        c.BinaryFor(s),
		DataDir:       dd,
		ConfigPath:    configPath,
		ConfigContent: cfg,
		LogPath:       path.Join(dd, "node.log"),
		Ports:         ports,
		Args:          nodeconfig.LaunchArgs(dd, configPath, ports, fam.StartFlags(role)),
	}
}

// BuildNodeSpecs builds the per-server specs in launch order (endpoints and
// bootnodes before producers/validators).
func BuildNodeSpecs(c *Cluster, enodes []string) []driver.NodeSpec {
	order := c.LaunchOrder()
	specs := make([]driver.NodeSpec, 0, len(order))
	for _, s := range order {
		specs = append(specs, BuildNodeSpec(c, s, enodes))
	}
	return specs
}

// Describe renders a human-readable deploy plan (a dry-run of what would be
// provisioned and launched, in order).
func Describe(c *Cluster, specs []driver.NodeSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "deploy plan: %d server(s), croissant block %d, genesis %s\n",
		len(specs), c.CroissantBlock, c.GenesisFile)
	fmt.Fprintf(&b, "%-6s %-10s %-16s %-8s %s\n", "INDEX", "ROLE", "HOST", "RPCPORT", "BINARY")
	for _, s := range specs {
		fmt.Fprintf(&b, "%-6d %-10s %-16s %-8d %s\n", s.Index, s.Role, s.Host, s.Ports.HTTP, s.Binary)
	}
	return b.String()
}
