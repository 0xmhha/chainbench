package process

import (
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// planNetwork is the network label a plan assembled from placements carries.
const planNetwork = "local"

// PlanOf is the plan a set of placed nodes launches from: each request and its
// placement become one node record, and the record becomes the launch spec the
// same way a composed workspace's does (SpecOf), so a spec-built
// network and a composed one describe a node identically. Launch order is
// slice position; a request naming no binary takes the manifest's. Args are
// deliberately left empty — the argv is assembled single-sited in Direct.Arm,
// which also holds the identity flags this cannot know. It performs no I/O.
func PlanOf(plugin registry.ChainPlugin, reqs []node.LaunchReq, placements []node.Placement, genesis []byte, dataRoot string, caps []string) (Plan, error) {
	if len(reqs) == 0 {
		return Plan{}, fmt.Errorf("launcher: plan: no nodes")
	}
	if len(placements) != len(reqs) {
		return Plan{}, fmt.Errorf("launcher: plan: %d placements for %d requests", len(placements), len(reqs))
	}
	m := plugin.Manifest()
	layout := node.Layout{Root: dataRoot}
	specs := make([]NodeSpec, 0, len(reqs))
	for i, req := range reqs {
		pl := placements[i]
		label := pl.Label
		if label == "" {
			label = node.LabelFor(i + 1)
		}
		dataDir := pl.DataDir
		if dataDir == "" {
			dataDir = layout.DataDir(label)
		}
		spec := SpecOf(node.Record{
			Index:      i + 1,
			Label:      string(label),
			Role:       string(req.Role),
			SyncMode:   req.Sync,
			Host:       pl.Host,
			DataDir:    dataDir,
			ConfigPath: layout.ConfigPath(label),
			LogPath:    layout.LogPath(label),
			Endpoints:  pl.Ports,
		})
		spec.Binary = req.Binary
		if spec.Binary == "" {
			spec.Binary = m.Binary
		}
		specs = append(specs, spec)
	}
	return Plan{
		Chain:        m.ID,
		Network:      planNetwork,
		DataRoot:     dataRoot,
		GenesisPath:  filepath.Join(dataRoot, "genesis.json"),
		Genesis:      genesis,
		Capabilities: caps,
		Nodes:        specs,
	}, nil
}
