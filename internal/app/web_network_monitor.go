package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webNetworkMonitorOperation collects remote node logs with the starting
// operator's own SSH binding until it is cancelled or that access is revoked.
const webNetworkMonitorOperation = "network.monitor"

// UseMonitor lets remote log collection jobs archive into the shared monitor.
func (e *WebChainEngine) UseMonitor(m *WebMonitor) { e.monitor = m }

// prepareMonitor reviews a remote log collection job. It changes nothing on
// the target and claims only its own control path, so node controls on the same
// network are not blocked while it runs.
func (e *WebChainEngine) prepareMonitor(ctx context.Context, a DeploymentActor, in WebPlanInput, args webChainArguments) (WebPreparedJob, error) {
	var out WebPreparedJob
	if e.monitor == nil {
		return out, errors.New("node log collection is not available on this server")
	}
	if args.ServerRef == "" || args.ManifestID != "" || args.AssetID != "" || args.Validators != 0 || args.ReplacementAssetID != "" || args.ConfigOverrides != nil || args.CaseRefs != nil || args.ChainPresetRef != nil {
		return out, errors.New("log collection takes only the target server")
	}
	if len(in.AssetRefs) != 0 || len(in.NodeIDs) != 0 || in.Retention != "retain" {
		return out, errors.New("log collection reads every recorded node and changes nothing")
	}
	workspace, err := e.documents.Workspace(in.WorkspaceID)
	if err != nil {
		return out, err
	}
	refs, _ := json.Marshal(workspace.Documents)
	selected, _ := json.Marshal(in.DocumentRefs)
	if string(refs) != string(selected) {
		return out, ErrDeploymentConflict
	}
	p := webChainPayload{Input: in, Arguments: args, WorkspaceRevision: workspace.Revision}
	for _, ref := range in.DocumentRefs {
		d, err := e.documents.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return out, err
		}
		switch d.Kind {
		case "server-set":
			p.Set = d
		case "workspace-config":
			p.Config = d
		default:
			return out, errors.New("log collection requires server-set and workspace-config documents only")
		}
	}
	if len(in.CredentialBindings) != 1 {
		return out, errors.New("remote log collection uses your own SSH binding for the target server")
	}
	for name, id := range in.CredentialBindings {
		if name != args.ServerRef || id == "" || e.documents.Bindings(a, in.WorkspaceID)[name] != id {
			return out, ErrDeploymentConflict
		}
	}
	p.Target, err = e.monitorTarget(ctx, a, p)
	if err != nil {
		return out, err
	}
	if p.ControlDir, err = filepath.Abs(filepath.Join(e.root, "networks", workspace.ID)); err != nil {
		return out, err
	}
	if err = recordedMonitorTarget(p); err != nil {
		return out, err
	}
	control, err := (resource.Opener{}).Inspect(ctx, resource.Spec{DataRoot: p.ControlDir})
	if err != nil {
		return out, err
	}
	out.Claims = []WebResourceClaim{{HostIdentity: control.HostIdentity, DataPath: filepath.Join(filepath.Dir(filepath.Dir(control.DataPath)), "monitor-jobs", workspace.ID)}}
	out.Phases = []string{webNetworkMonitorOperation}
	out.RequiredAccess = []string{"ssh read access to recorded node log files"}
	out.Changes = []string{"Read recorded node logs over your SSH binding until cancelled; nodes and files are not changed"}
	out.Payload, err = json.Marshal(p)
	if err != nil {
		return out, err
	}
	out.Fingerprint = manifestHash(out.Payload)
	return out, nil
}

// recordedMonitorTarget requires the owned network to live on the job's target.
func recordedMonitorTarget(p webChainPayload) error {
	raw, err := os.ReadFile(filepath.Join(p.ControlDir, "web-target.json"))
	if os.IsNotExist(err) {
		return ErrDeploymentNotFound
	}
	if err != nil {
		return err
	}
	var recorded resource.Inspection
	if err = json.Unmarshal(raw, &recorded); err != nil {
		return err
	}
	if recorded != p.Target {
		return ErrDeploymentConflict // The owned network lives elsewhere.
	}
	return nil
}

// monitorTarget inspects the pinned server with the job's own binding.
func (e *WebChainEngine) monitorTarget(ctx context.Context, a DeploymentActor, p webChainPayload) (resource.Inspection, error) {
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return resource.Inspection{}, err
	}
	server, err := set.ByName(p.Arguments.ServerRef)
	if err != nil {
		return resource.Inspection{}, err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return resource.Inspection{}, err
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return resource.Inspection{}, err
	}
	target, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, resource.TargetOf(server, wc.DataRoot))
	if err != nil {
		return resource.Inspection{}, err
	}
	if target.Transport == "local" {
		return resource.Inspection{}, errors.New("local node logs are collected without a job")
	}
	return target, nil
}

// executeMonitor archives remote logs every collection period until the job
// context ends. Every SSH command dials anew through the job's credential
// guard, so revocation stops further access even before cancellation lands.
func (e *WebChainEngine) executeMonitor(ctx context.Context, a DeploymentActor, p webChainPayload, report func(WebJobPhase) error) (WebJobResult, error) {
	result := WebJobResult{NodeDisposition: "retained"}
	target, err := e.monitorTarget(ctx, a, p)
	if err != nil {
		return result, err
	}
	if target != p.Target {
		return result, ErrDeploymentConflict
	}
	if err = recordedMonitorTarget(p); err != nil {
		return result, err
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return result, err
	}
	opener := resource.Opener{Lookup: lookup}
	open := func(ctx context.Context, state State, ns node.Record) (webLogSource, error) {
		// The lookup guard reports revocation as ErrDeploymentForbidden, which
		// ends collection; a refused SSH dial alone carries no such cause.
		if _, err := lookup(p.Arguments.ServerRef); err != nil {
			return nil, err
		}
		spec := webNodeTarget(state, ns)
		machine, err := opener.Inspect(ctx, spec)
		if err != nil {
			return nil, err
		}
		if machine != p.Target {
			return nil, ErrDeploymentConflict
		}
		access, err := opener.Open(spec)
		if err != nil {
			return nil, err
		}
		if access.Runner == nil {
			return nil, errors.New("remote log access requires an SSH runner")
		}
		return webRemoteLogs{run: access.Runner}, nil
	}
	network := filepath.Base(p.ControlDir)
	detach := e.monitor.attachRemote(network)
	defer detach()
	err = e.phase(ctx, a, webNetworkMonitorOperation, report, func() error {
		ticker := time.NewTicker(e.monitor.interval)
		defer ticker.Stop()
		for {
			if err := e.monitor.CollectRemoteLogs(ctx, network, open); err != nil && ctx.Err() == nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			select {
			case <-ctx.Done():
				return nil // Stopping is the normal end of collection.
			case <-ticker.C:
			}
		}
	})
	if err == nil {
		err = ctx.Err()
	}
	return result, err
}
