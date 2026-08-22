package engine

import (
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/portplan"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// PlacedNode pairs a node's placement request with its resolved location, so
// plan assembly has both the binary/sync choices (from the request) and the
// address and ports (from netmap).
type PlacedNode struct {
	Req       place.NodeReq
	Placement netmap.Placement
}

// AssemblePlan builds a driver.Plan from allocator-resolved placements and
// pre-built genesis bytes. It is the place-driven equivalent of setup.BuildPlan,
// which instead derives ports positionally via node.Offset from a fixed base:
// here the ports come from the allocator, so OS-assigned and remote-per-host
// modes compose without changing this function. It performs no I/O, so it is
// unit-testable and inspectable before anything runs.
//
// Each node's launch order (1-based Index) is its slice position. genesis is the
// already-built genesis.json content (the caller obtains it from the consensus
// family); caps are the advertised capabilities.
func AssemblePlan(plugin registry.ChainPlugin, placed []PlacedNode, genesis []byte, dataRoot string, caps []string) (driver.Plan, error) {
	if len(placed) == 0 {
		return driver.Plan{}, fmt.Errorf("engine: assemble plan: no nodes")
	}
	m := plugin.Manifest()

	specs := make([]driver.NodeSpec, 0, len(placed))
	for i, pn := range placed {
		idx := i + 1
		ports := endpointsFrom(pn.Placement.Ports)
		dataDir := pn.Placement.DataDir
		if dataDir == "" {
			dataDir = filepath.Join(dataRoot, fmt.Sprintf("node%d", idx))
		}
		configPath := filepath.Join(dataRoot, fmt.Sprintf("config_node%d.toml", idx))
		binary := pn.Req.Binary
		if binary == "" {
			binary = m.Binary
		}
		// Args are deliberately not assembled here: the launch argv is
		// single-sited in the launcher's arming step (launchopt Builder), which
		// also holds the identity flags this function cannot know.
		specs = append(specs, driver.NodeSpec{
			Index:      idx,
			Role:       pn.Req.Role,
			SyncMode:   pn.Req.Sync,
			Host:       pn.Placement.Host,
			Binary:     binary,
			DataDir:    dataDir,
			ConfigPath: configPath,
			LogPath:    filepath.Join(dataRoot, "logs", fmt.Sprintf("node%d.log", idx)),
			Ports:      ports,
		})
	}

	return driver.Plan{
		Chain:        m.ID,
		Network:      "local",
		DataRoot:     dataRoot,
		GenesisPath:  filepath.Join(dataRoot, "genesis.json"),
		Genesis:      genesis,
		Capabilities: caps,
		Nodes:        specs,
	}, nil
}

// endpointsFrom maps allocator ports (portplan) to the launch endpoint set.
// Etcd is derived by the wemix binary as P2P+1 and is not a launch endpoint,
// so it is intentionally dropped. Metrics (0 on tight rpc steps) makes the
// node's scrape endpoint reachable for the metric verification source.
func endpointsFrom(p portplan.Ports) node.Endpoints {
	return node.Endpoints{P2P: p.P2P, HTTP: p.HTTP, WS: p.WS, Auth: p.Auth, Metrics: p.Metrics}
}
