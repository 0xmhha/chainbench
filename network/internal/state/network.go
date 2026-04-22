package state

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// LoadActiveOptions controls how state files map to a Network.
type LoadActiveOptions struct {
	// StateDir is the directory containing pids.json and current-profile.yaml.
	// Empty means "state/" relative to the process working directory.
	StateDir string
	// Name is Network.name. Empty defaults to "local".
	Name string
}

// ErrStateNotFound is returned when required state files are absent.
// Callers in the command layer map this to UPSTREAM_ERROR.
var ErrStateNotFound = errors.New("state: active chain state not found")

// LoadActive reads pids.json + current-profile.yaml under opts.StateDir and
// builds a types.Network. Nodes are sorted by numeric pids key ascending.
func LoadActive(opts LoadActiveOptions) (*types.Network, error) {
	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir = "state"
	}
	name := opts.Name
	if name == "" {
		name = "local"
	}

	pids, err := ReadPIDsFile(filepath.Join(stateDir, "pids.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStateNotFound, err)
	}
	profile, err := ReadProfileFile(filepath.Join(stateDir, "current-profile.yaml"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStateNotFound, err)
	}

	chainType := profile.Chain.Type
	if chainType == "" {
		chainType = "stablenet"
	}

	nodes, err := buildNodes(pids)
	if err != nil {
		return nil, err
	}

	net := &types.Network{
		Name:      name,
		ChainType: types.NetworkChainType(chainType),
		ChainId:   int(profile.Chain.ChainID),
		Nodes:     nodes,
	}
	return net, nil
}

func buildNodes(p *PIDsFile) ([]types.Node, error) {
	keys := make([]string, 0, len(p.Nodes))
	for k := range p.Nodes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, _ := strconv.Atoi(keys[i])
		b, _ := strconv.Atoi(keys[j])
		return a < b
	})

	out := make([]types.Node, 0, len(keys))
	for _, k := range keys {
		info := p.Nodes[k]
		id := "node" + k
		role := mapRole(info.Type)
		ws := fmt.Sprintf("ws://127.0.0.1:%d", info.WSPort)
		node := types.Node{
			Id:           id,
			Provider:     types.NodeProvider("local"),
			Http:         fmt.Sprintf("http://127.0.0.1:%d", info.HTTPPort),
			Ws:           &ws,
			Role:         &role,
			ProviderMeta: types.NodeProviderMeta{"pid_key": id},
		}
		out = append(out, node)
	}
	return out, nil
}

func mapRole(t string) types.NodeRole {
	switch t {
	case "validator":
		return types.NodeRole("validator")
	case "endpoint":
		return types.NodeRole("endpoint")
	default:
		return types.NodeRole("observer")
	}
}
