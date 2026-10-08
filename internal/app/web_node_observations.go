package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/resource"
)

// ObserveNetwork uses only the caller's credentials and never rewrites records.
// An unavailable probe is explicit, rather than treating cached PIDs as live.
func (e *WebChainEngine) ObserveNetwork(ctx context.Context, a DeploymentActor, id string) (WebNetwork, error) {
	if err := e.allowed(a); err != nil {
		return WebNetwork{}, err
	}
	workspace, err := e.documents.Workspace(id)
	if err != nil {
		return WebNetwork{}, err
	}
	dir := filepath.Join(e.root, "networks", workspace.ID)
	raw, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
	if os.IsNotExist(err) {
		return WebNetwork{}, ErrDeploymentNotFound
	}
	if err != nil {
		return WebNetwork{}, err
	}
	var state State
	if err = json.Unmarshal(raw, &state); err != nil {
		return WebNetwork{}, err
	}
	metadata, err := os.ReadFile(filepath.Join(dir, "web-target.json"))
	if err != nil {
		return WebNetwork{}, ErrDeploymentConflict
	}
	var target resource.Inspection
	if err = json.Unmarshal(metadata, &target); err != nil {
		return WebNetwork{}, err
	}
	network := WebNetwork{ID: workspace.ID, WorkspaceID: workspace.ID, Ownership: "owned", Version: workspace.Revision, Nodes: []WebNode{}}
	for _, ns := range state.Nodes {
		network.Nodes = append(network.Nodes, WebNode{ID: string(ns.NodeLabel()), NetworkID: workspace.ID, Role: ns.Role, HostIdentity: target.HostIdentity, DataPath: ns.DataDir, PID: ns.PID, State: "unknown", ObservationReason: "probe_unavailable", SupportedControls: []string{}, ObservedAt: time.Now().UTC()})
	}
	ledgerMatches := verifyWebRecordLedger(state, dir) == nil
	var declaration DeploymentDocument
	for _, ref := range workspace.Documents {
		d, err := e.documents.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return WebNetwork{}, err
		}
		if d.Kind == "server-set" {
			declaration = d
		}
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, declaration.DeploymentDocumentInput, e.documents.Bindings(a, workspace.ID))
	if err != nil {
		return network, nil
	}
	opener := resource.Opener{Lookup: lookup}
	actual, err := opener.Inspect(ctx, state.Target)
	if err != nil {
		return network, nil
	}
	if actual != target {
		return network, nil
	}
	for i, ns := range state.Nodes {
		spec := webNodeTarget(state, ns)
		machine, err := opener.Inspect(ctx, spec)
		if err != nil || machine != target {
			continue
		}
		access, err := opener.Open(spec)
		if err != nil {
			continue
		}
		observer := webObserverFor(access, machine)
		binary := state.Binary
		if ns.Binary != "" {
			binary = state.Binaries[ns.Binary]
		}
		observed := observeWebNodeProcess(ctx, observer, binary, ns)
		if ns.PID == 0 {
			observed = observeWebNodeVacancy(ctx, observer, binary, ns)
		}
		if !ledgerMatches && observed.State != "unrecorded_running" {
			observed.State, observed.Reason = "ownership_mismatch", "ledger_mismatch"
		}
		network.Nodes[i].State = observed.State
		network.Nodes[i].ObservedPID = observed.PID
		network.Nodes[i].ObservationReason = observed.Reason
		network.Nodes[i].ObservedAt = time.Now().UTC()
		if observed.State == "running" || observed.State == "stopped" {
			network.Nodes[i].SupportedControls = []string{"node.start", "node.stop"}
			if observed.State == "running" && (ns.Role == "en" || ns.Role == "pn") {
				network.Nodes[i].SupportedControls = append(network.Nodes[i].SupportedControls, "node.reset")
			}
		}
	}
	return network, nil
}

func (s *WebJobs) ObserveNetwork(ctx context.Context, a DeploymentActor, id string) (WebNetwork, error) {
	if err := s.allowed(a); err != nil {
		return WebNetwork{}, err
	}
	observer, ok := s.engine.(interface {
		ObserveNetwork(context.Context, DeploymentActor, string) (WebNetwork, error)
	})
	if !ok {
		return WebNetwork{}, ErrDeploymentNotFound
	}
	return observer.ObserveNetwork(ctx, a, id)
}
