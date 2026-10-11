package app

import (
	"errors"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// validateWebControlLayout limits the adapter to the selected server, or the
// whole set for a spread composition, and the owned composition's paths. The core may resolve each node by server name,
// so checking only the composition target would not establish its effect scope.
func validateWebControlLayout(state State, p webChainPayload, wc resource.WorkspaceConfig) error {
	if state.FormatVersion != 3 || !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(state.CompositionID) || len(state.Nodes) == 0 {
		return webConflict("the network record has an unsupported format or no nodes")
	}
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return err
	}
	servers, err := webPlanServers(set, p.Arguments, p.Input.Operation)
	if err != nil {
		return err
	}
	if state.Target.DataRoot != wc.DataRoot {
		return webConflict("the network was recorded under another data root")
	}
	layout := node.Layout{Root: wc.DataRoot, CompositionID: state.CompositionID, NodesDir: wc.Paths.Nodes, RuntimeDir: wc.Paths.Runtime, LogsDir: wc.Paths.Logs}
	indexes, labels, producers := map[int]bool{}, map[node.Label]bool{}, 0
	for _, ns := range state.Nodes {
		label := ns.NodeLabel()
		role, err := node.NormalizeRole(ns.Role)
		if err != nil || ns.Index < 1 || ns.Index > len(state.Nodes) || indexes[ns.Index] || labels[label] || ns.PID < 0 || !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`).MatchString(string(label)) {
			return webConflict("a recorded node has an invalid index, label, role or PID")
		}
		indexes[ns.Index], labels[label] = true, true
		if node.Is(role, node.RoleBP) {
			producers++
		}
		placed := false
		for _, server := range servers {
			placed = placed || (ns.Server == "" || ns.Server == server.Name) && ns.Host == server.Host
		}
		if !placed || ns.DataDir != layout.DataDir(label) || ns.ConfigPath != layout.ConfigPath(label) || ns.LogPath != layout.LogPath(label) || !filepath.IsAbs(ns.DataDir) {
			return webConflict("a recorded node is outside the selected servers or its paths moved")
		}
	}
	if producers < 1 || producers > 128 || producers != state.BPCount || p.Arguments.Validators != 0 && p.Arguments.Validators != producers {
		return errors.New("selected placement differs from the owned network")
	}
	return nil
}

// webRecordedControlPlacement preserves recorded ports, including occupied
// inventory slots. Each endpoint set must be a permitted slot on this server;
// duplicates still fail through the same placement map used by composition.
func webRecordedControlPlacement(state State, p *webChainPayload, pool resource.Pool) (*node.Map, error) {
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return nil, err
	}
	if err = validateWebControlLayout(state, *p, wc); err != nil {
		return nil, err
	}
	if err = pool.Validate(); err != nil {
		return nil, err
	}
	if len(state.Nodes) > pool.Cap() || !p.Arguments.AllServers && len(pool.Hosts) != 1 {
		return nil, webConflict("the recorded nodes do not fit the selected servers")
	}
	hosts := map[string]bool{}
	for _, h := range pool.Hosts {
		hosts[h.Addr] = true
	}
	permitted := map[node.Endpoints]bool{}
	for slot := 1; slot <= pool.Slots; slot++ {
		ports, err := resource.PlanBands(slot, pool.Ports, pool.Reservation)
		if err != nil {
			return nil, err
		}
		if pool.Ports.Metrics != nil && pool.Ports.Metrics.Step == 0 && slot > 1 {
			ports.Metrics = 0
		}
		permitted[ports] = true
	}
	nodes := slices.Clone(state.Nodes)
	slices.SortFunc(nodes, func(a, b node.Record) int { return a.Index - b.Index })
	placements := make([]node.Placement, 0, len(nodes))
	for _, ns := range nodes {
		if !hosts[ns.Host] || !permitted[ns.Endpoints] {
			return nil, webConflict("a recorded node uses an address or ports the server pool does not allow")
		}
		placements = append(placements, node.Placement{Index: ns.Index, Label: ns.NodeLabel(), Role: node.Role(ns.Role), Host: ns.Host, Ports: ns.Endpoints, DataDir: ns.DataDir})
	}
	placement, err := node.NewMap(placements)
	if err != nil {
		return nil, err
	}
	p.Arguments.Validators = state.BPCount
	return placement, nil
}

// webNodeTarget matches the core's per-node machine resolution.
func webNodeTarget(state State, ns node.Record) resource.Spec {
	if ns.Server != "" {
		return resource.Spec{Server: ns.Server, Host: ns.Host, DataRoot: state.Target.DataRoot}
	}
	return state.Target
}
