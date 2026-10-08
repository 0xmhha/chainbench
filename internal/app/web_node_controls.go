package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webNodeControlOperation identifies operations that use an existing owned record.
func webNodeControlOperation(operation string) bool {
	return operation == "node.start" || operation == "node.stop" || operation == "node.reset"
}

// webResetNode checks role, recorded identity and the accepted node directory.
// A stopped record lacks sufficient process evidence for destructive reset until
// residual-process reconciliation is available; it does not imply a vacant path.
func webResetNode(state State, p webChainPayload) (node.Record, error) {
	if err := webSelectedNode(state, p.Input.NodeIDs); err != nil {
		return node.Record{}, err
	}
	for _, ns := range state.Nodes {
		if string(ns.NodeLabel()) != p.Input.NodeIDs[0] {
			continue
		}
		if node.Is(node.Role(ns.Role), node.RoleBP) {
			return node.Record{}, errors.New("a block producer cannot be reset")
		}
		if ns.PID <= 0 || (!node.Is(node.Role(ns.Role), node.RoleEN) && !node.Is(node.Role(ns.Role), node.RolePN)) || ns.Binary != "" {
			return node.Record{}, ErrDeploymentConflict
		}
		wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
		if err != nil {
			return node.Record{}, ErrDeploymentConflict
		}
		if !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(state.CompositionID) {
			return node.Record{}, ErrDeploymentConflict
		}
		layout := node.Layout{Root: wc.DataRoot, CompositionID: state.CompositionID, NodesDir: wc.Paths.Nodes, RuntimeDir: wc.Paths.Runtime, LogsDir: wc.Paths.Logs}
		if ns.DataDir != layout.DataDir(ns.NodeLabel()) || ns.ConfigPath != layout.ConfigPath(ns.NodeLabel()) || !filepath.IsAbs(ns.DataDir) {
			return node.Record{}, ErrDeploymentConflict
		}
		return ns, nil
	}
	return node.Record{}, ErrDeploymentNotFound
}

// verifyWebResetInputs verifies the target directory and the inputs reset reads
// before it can remove data. The accepted shared declarations stay authoritative.
func verifyWebResetInputs(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup) error {
	ns, err := webResetNode(state, p)
	if err != nil {
		return err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return err
	}
	for _, entry := range []struct {
		path string
		doc  DeploymentDocument
	}{{state.WorkspaceConfig, p.Config}, {state.ServerSet, p.Set}} {
		expected, err := ExportDeploymentDocument(entry.doc.DeploymentDocumentInput, "yaml")
		if err != nil {
			return err
		}
		actual, err := os.ReadFile(entry.path)
		if err != nil || manifestHash(actual) != manifestHash(expected) {
			return ErrDeploymentConflict
		}
	}
	spec := state.Target
	spec.DataRoot = ns.DataDir
	actual, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, spec)
	if err != nil {
		return err
	}
	expected := filepath.Join(p.Target.DataPath, wc.Paths.Nodes, state.CompositionID, string(ns.NodeLabel()))
	if actual.HostIdentity != p.Target.HostIdentity || actual.DataPath != expected {
		return ErrDeploymentConflict
	}
	access, err := (resource.Opener{Lookup: lookup}).Open(state.Target)
	if err != nil {
		return err
	}
	for _, path := range []string{ns.ConfigPath, state.GenesisPath} {
		if state.LaunchInputs[path] == "" {
			return ErrDeploymentConflict
		}
		checksum, err := access.Files.Checksum(ctx, path)
		if err != nil || checksum != state.LaunchInputs[path] {
			return ErrDeploymentConflict
		}
	}
	return nil
}

func verifyWebNodeProcesses(ctx context.Context, state State, selected []string, lookup resource.Lookup) error {
	if len(selected) != 1 {
		return ErrDeploymentConflict
	}
	for _, ns := range state.Nodes {
		if string(ns.NodeLabel()) != selected[0] || ns.PID == 0 {
			continue
		}
		access, err := (resource.Opener{Lookup: lookup}).Open(state.Target)
		if err != nil {
			return err
		}
		target, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, state.Target)
		if err != nil {
			return err
		}
		observer := webObserverFor(access, target)
		if observer == nil {
			return ErrDeploymentConflict
		}
		binary := state.Binary
		if ns.Binary != "" {
			binary = state.Binaries[ns.Binary]
		}
		if observeWebNodeProcess(ctx, observer, binary, ns).State != "running" {
			return ErrDeploymentConflict
		}
	}
	return nil
}
