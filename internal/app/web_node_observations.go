package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
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
		n := WebNode{ID: string(ns.NodeLabel()), NetworkID: workspace.ID, Role: ns.Role, HostIdentity: target.HostIdentity, DataPath: ns.DataDir, PID: ns.PID, State: "unknown", ObservationReason: "probe_unavailable", SupportedControls: []string{}, ObservedAt: time.Now().UTC()}
		n.ControlLimits = webControlLimits(n, network.Ownership, ns.Config != "")
		network.Nodes = append(network.Nodes, n)
	}
	// A spread network records each server's machine; a node is observed
	// only on the machine recorded for its own server.
	machines := map[string]resource.Inspection{}
	if raw, err := os.ReadFile(filepath.Join(dir, "web-targets.json")); err == nil {
		var recorded []webRecordedMachine
		if err = json.Unmarshal(raw, &recorded); err != nil {
			return WebNetwork{}, err
		}
		for _, m := range recorded {
			machines[m.Server] = m.Target
		}
	} else if webRecordSpansServers(state) {
		return network, nil
	}
	expectedOf := func(ns node.Record) resource.Inspection {
		if m, ok := machines[ns.Server]; ok {
			return m
		}
		return target
	}
	for i, ns := range state.Nodes {
		network.Nodes[i].HostIdentity = expectedOf(ns).HostIdentity
	}
	ledgerMatches := verifyWebRecordLedger(state, dir) == nil
	p := webChainPayload{ControlDir: dir, Target: target}
	for _, ref := range workspace.Documents {
		d, err := e.documents.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return WebNetwork{}, err
		}
		if d.Kind == "server-set" {
			p.Set = d
		}
		if d.Kind == "workspace-config" {
			p.Config = d
		}
	}
	bindings := e.documents.Bindings(a, workspace.ID)
	p.Input.CredentialBindings = bindings
	lookup, err := e.targetReader(ctx, a, p)
	if err != nil {
		return network, nil
	}
	lookup = lookup.observationSnapshot()
	opener := lookup
	actual, err := opener.Inspect(ctx, state.Target)
	if err != nil {
		return network, nil
	}
	if actual != target {
		return network, nil
	}
	// What each node's machine shows comes from `chainbench chain observe`,
	// read with the caller's own logins; the Web adds what only it knows: the
	// machine it recorded, the ledger it compares and the bindings it pinned.
	p.Input.CredentialBindings = bindings
	observed, err := e.observeThroughCLI(ctx, a, p)
	if err != nil {
		return network, nil
	}
	for i, ns := range state.Nodes {
		machine, err := opener.Inspect(ctx, webNodeTarget(state, ns))
		if err != nil || machine != expectedOf(ns) {
			continue
		}
		seen, ok := observed[ns.Index]
		if !ok {
			continue
		}
		if !ledgerMatches && seen.State != "unrecorded_running" {
			seen.State, seen.Reason = "ownership_mismatch", "ledger_mismatch"
		}
		network.Nodes[i].State = string(seen.State)
		network.Nodes[i].ObservedPID = seen.PID
		network.Nodes[i].ObservationReason = seen.Reason
		network.Nodes[i].ObservedAt = time.Now().UTC()
		if seen.State == "running" || seen.State == "stopped" {
			if ns.Binary != "" {
				selected := p
				selected.Input.NodeIDs = []string{string(ns.NodeLabel())}
				binding, err := e.observedWebBinary(ctx, state, selected, lookup)
				if err != nil || binding == nil {
					network.Nodes[i].State = "ownership_mismatch"
					network.Nodes[i].ObservationReason = "binary_binding_unavailable"
					continue
				}
				network.Nodes[i].BinaryAssetID = binding.Evidence.ID
				network.Nodes[i].BinarySHA256 = binding.Evidence.SHA256
			}
			if seen.Executable != "ready" {
				network.Nodes[i].ObservationReason = "binary_launch_unavailable"
				if seen.State == "running" {
					network.Nodes[i].SupportedControls = []string{"node.stop"}
				}
				continue
			}
			network.Nodes[i].SupportedControls = []string{"node.start", "node.stop", "node.restart"}
			if ns.Config == "" {
				network.Nodes[i].SupportedControls = append(network.Nodes[i].SupportedControls, "node.swap")
			}
			if ns.Role == "en" || ns.Role == "pn" {
				network.Nodes[i].SupportedControls = append(network.Nodes[i].SupportedControls, "node.reset")
			}
		}
	}
	// What observation established decides each control and its reason.
	for i, ns := range state.Nodes {
		network.Nodes[i].ControlLimits = webControlLimits(network.Nodes[i], network.Ownership, ns.Config != "")
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

// observeThroughCLI runs `chainbench chain observe` on the network at
// p.ControlDir with the caller's logins and returns each node's observation by
// index.
func (e *WebChainEngine) observeThroughCLI(ctx context.Context, a DeploymentActor, p webChainPayload) (map[int]chainsetup.NodeObservation, error) {
	if e.cli == nil {
		return nil, errNoChainCLI
	}
	logins, cleanup, err := e.jobLogins(ctx, a, p)
	if err != nil {
		return nil, err
	}
	raw, err := e.cli.Run(ctx, "chain", "observe", "--json", "--workspace-dir", p.ControlDir, "--server-logins", logins)
	if err = errors.Join(err, cleanup()); err != nil {
		return nil, err
	}
	var out ChainObserveOut
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("chainbench chain observe: unreadable output: %w", err)
	}
	byIndex := make(map[int]chainsetup.NodeObservation, len(out.Nodes))
	for _, n := range out.Nodes {
		byIndex[n.Index] = n
	}
	return byIndex, nil
}
