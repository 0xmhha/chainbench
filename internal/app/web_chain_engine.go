package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// WebChainEngine adapts provisioned native binaries and existing composition
// verbs to durable jobs. Private SSH leases resolve named remote targets without
// writing credentials into shared declarations or composition records.
type WebChainEngine struct {
	root, keys string
	documents  *DeploymentStore
	manifests  *ManifestStore
	assets     map[string]ManifestBinary
	authorize  func(DeploymentActor) error
	keysMu     sync.Mutex
}

func NewWebChainEngine(root, keys string, documents *DeploymentStore, manifests *ManifestStore, assets []ManifestBinary, authorize func(DeploymentActor) error) *WebChainEngine {
	byID := map[string]ManifestBinary{}
	for _, asset := range assets {
		byID[asset.ID] = asset
	}
	return &WebChainEngine{root: root, keys: keys, documents: documents, manifests: manifests, assets: byID, authorize: authorize}
}

func (e *WebChainEngine) binaryAsset(id string) (ManifestBinary, error) {
	if asset, ok := e.assets[id]; ok {
		return asset, nil
	}
	if e.manifests == nil {
		return ManifestBinary{}, ErrDeploymentNotFound
	}
	return e.manifests.BinaryAsset(id)
}

type webChainArguments struct {
	ManifestID     string                  `json:"manifestId"`
	AssetID        string                  `json:"assetId"`
	ServerRef      string                  `json:"serverRef"`
	Validators     int                     `json:"validators"`
	CaseRefs       []DeploymentDocumentRef `json:"caseRefs,omitempty"`
	ChainPresetRef *DeploymentDocumentRef  `json:"chainPresetRef,omitempty"`
}
type webChainPayload struct {
	Input             WebPlanInput           `json:"input"`
	Arguments         webChainArguments      `json:"arguments"`
	WorkspaceRevision int                    `json:"workspaceRevision"`
	Set               DeploymentDocument     `json:"set"`
	Config            DeploymentDocument     `json:"config"`
	Manifest          ManagedManifest        `json:"manifest"`
	Binary            ManifestBinaryEvidence `json:"binary"`
	ControlDir        string                 `json:"controlDir"`
	RecordDigest      string                 `json:"recordDigest"`
	Target            resource.Inspection    `json:"target"`
	ExecutionBinary   string                 `json:"executionBinary"`
	Keys              webKeySnapshot         `json:"keys"`
	TestRun           *webTestRun            `json:"testRun,omitempty"`
	Preset            *webPresetComposition  `json:"preset,omitempty"`
}

func (e *WebChainEngine) allowed(a DeploymentActor) error {
	if !a.canEdit() {
		return ErrDeploymentForbidden
	}
	if e.authorize != nil {
		return e.authorize(a)
	}
	return nil
}
func (e *WebChainEngine) Prepare(ctx context.Context, a DeploymentActor, in WebPlanInput) (WebPreparedJob, error) {
	var out WebPreparedJob
	if err := e.allowed(a); err != nil {
		return out, err
	}
	if in.Operation != "chain.setup" && in.Operation != "chain.deploy" && !webNodeControlOperation(in.Operation) && in.Operation != "test.run" {
		return out, errors.New("operation requires an execution adapter that is not available")
	}
	var args webChainArguments
	dec := json.NewDecoder(bytes.NewReader(in.Arguments))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&args); err != nil {
		return out, errors.New("select manifest, binary and server; arbitrary command arguments are unavailable")
	}
	if dec.Decode(new(any)) != io.EOF {
		return out, errors.New("one argument object required")
	}
	if args.ChainPresetRef != nil && (in.Operation != "chain.setup" && in.Operation != "chain.deploy" || args.Validators != 0) {
		return out, errors.New("a saved chain preset applies to composition and supplies its own node layout")
	}
	if args.Validators == 0 && in.Operation != "test.run" && args.ChainPresetRef == nil {
		args.Validators = 4
	}
	if in.Operation != "test.run" && args.ChainPresetRef == nil && (args.Validators < 1 || args.Validators > 128) {
		return out, errors.New("validator count must be between 1 and 128")
	}
	workspace, err := e.documents.Workspace(in.WorkspaceID)
	if err != nil {
		return out, err
	}
	if len(in.AssetRefs) < 1 || len(in.AssetRefs) > 257 || args.ChainPresetRef == nil && in.Operation != "test.run" && (len(in.AssetRefs) != 1 || in.AssetRefs[0] != args.AssetID) {
		return out, errors.New("select the binary asset and every registered declaration dependency")
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
		// The workspace explicitly pins immutable server/path declarations.
		// A newer document revision does not change that binding; Prepare and
		// job acceptance still reject changed workspace references or revisions.
		switch d.Kind {
		case "server-set":
			p.Set = d
		case "workspace-config":
			p.Config = d
		default:
			return out, errors.New("this composition adapter requires server-set and workspace-config documents only")
		}
	}
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return out, err
	}
	server, err := set.Select(args.ServerRef, 0)
	if err != nil {
		return out, err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return out, err
	}
	if len(in.CredentialBindings) > 1 {
		return out, errors.New("select only the target server's personal SSH binding")
	}
	for name, id := range in.CredentialBindings {
		if name != args.ServerRef || id == "" || e.documents.Bindings(a, in.WorkspaceID)[name] != id {
			return out, ErrDeploymentConflict
		}
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, in.CredentialBindings)
	if err != nil {
		return out, err
	}
	p.Target, err = (resource.Opener{Lookup: lookup}).Inspect(ctx, resource.TargetOf(server, wc.DataRoot))
	if err != nil {
		return out, err
	}
	if p.Target.Transport == "local" && len(in.CredentialBindings) != 0 {
		return out, errors.New("local operations do not accept SSH credential bindings")
	}
	if wc.Execution.Chain == resource.ChainAttach || wc.Inputs.Mode != "generated" {
		return out, errors.New("attached or existing-input networks cannot be composed by this adapter")
	}
	p.Manifest, err = e.manifests.Get(args.ManifestID)
	if err != nil {
		return out, err
	}
	plugin, err := ValidateManifest(p.Manifest.ManifestInput)
	if err != nil {
		return out, err
	}
	asset, err := e.binaryAsset(args.AssetID)
	if err != nil {
		return out, err
	}
	p.Binary, err = asset.Verify(ctx, plugin.Protocol().Name)
	if err != nil {
		return out, err
	}
	if p.Binary.OS != p.Target.OS || p.Binary.Architecture != p.Target.Architecture {
		return out, errors.New("selected native binary does not match the verified target platform")
	}
	p.ExecutionBinary = asset.Path
	if p.Target.Transport == "ssh" {
		p.ExecutionBinary, err = wc.Resolve(resource.PurposeBinaries, p.Binary.SHA256+"/"+filepath.Base(asset.Path))
		if err != nil {
			return out, err
		}
	}
	p.ControlDir, err = filepath.Abs(filepath.Join(e.root, "networks", workspace.ID))
	if err != nil {
		return out, err
	}
	requests := []resource.Request{}
	if in.Operation == "test.run" {
		if len(in.NodeIDs) != 0 || args.Validators != 0 {
			return out, errors.New("test jobs use the node layout declared by the selected cases")
		}
		requests, err = e.prepareTestRun(ctx, &p)
		if err != nil {
			return out, err
		}
	} else {
		if len(args.CaseRefs) != 0 {
			return out, errors.New("case references belong to a test job")
		}
		if args.ChainPresetRef != nil {
			requests, err = e.preparePresetComposition(ctx, &p)
			if err != nil {
				return out, err
			}
		} else {
			for range args.Validators {
				requests = append(requests, resource.Request{Role: node.RoleBP})
			}
		}
	}
	pool := set.PoolFor(server, p.Arguments.Validators, server.Slots)
	pool.Reservation = plugin.Family().PortReservation()
	placement, err := resource.Assign(pool, requests)
	if err != nil {
		return out, err
	}
	ports := []int{}
	for _, entry := range placement.Placements() {
		for _, port := range []int{entry.Ports.P2P, entry.Ports.Etcd, entry.Ports.EtcdClient, entry.Ports.HTTP, entry.Ports.WS, entry.Ports.Auth, entry.Ports.Metrics} {
			if port > 0 {
				ports = append(ports, port)
			}
		}
	}
	controlTarget, err := (resource.Opener{}).Inspect(ctx, resource.Spec{DataRoot: p.ControlDir})
	if err != nil {
		return out, err
	}
	expectedAssets := map[string]bool{args.AssetID: true}
	if p.Preset != nil {
		for _, id := range p.Preset.Document.AssetRefs {
			expectedAssets[id] = true
		}
	}
	if p.TestRun != nil {
		for _, c := range p.TestRun.Cases {
			for _, id := range c.Document.AssetRefs {
				expectedAssets[id] = true
			}
		}
	}
	if len(in.AssetRefs) != len(expectedAssets) {
		return out, errors.New("selected assets differ from the reviewed declaration")
	}
	for _, id := range in.AssetRefs {
		if !expectedAssets[id] {
			return out, errors.New("selected assets differ from the reviewed declaration")
		}
		delete(expectedAssets, id)
	}
	out.Claims = []WebResourceClaim{{HostIdentity: p.Target.HostIdentity, DataPath: p.Target.DataPath, Ports: ports}, {HostIdentity: controlTarget.HostIdentity, DataPath: controlTarget.DataPath}}
	record, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
	if err == nil {
		var state State
		if err = json.Unmarshal(record, &state); err != nil {
			return out, err
		}
		ownedTarget, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, state.Target)
		if err != nil {
			return out, err
		}
		if ownedTarget != p.Target {
			return out, ErrDeploymentConflict
		}
		p.RecordDigest = manifestHash(record)
		if in.Operation == "chain.setup" || in.Operation == "chain.deploy" {
			for _, ns := range state.Nodes {
				if ns.PID > 0 {
					return out, errors.New("existing nodes must be stopped before composing again")
				}
			}
		}
		if webNodeControlOperation(in.Operation) {
			p.ExecutionBinary, err = bindWebControlBinary(ctx, state, p, lookup)
			if err != nil {
				return out, err
			}
			if err = validateWebChainRecord(state, p); err != nil {
				return out, err
			}
			if err = webSelectedNode(state, in.NodeIDs); err != nil {
				return out, err
			}
			if in.Operation == "node.reset" {
				if err = verifyWebResetInputs(ctx, state, p, lookup); err != nil {
					return out, err
				}
			}
			if err = verifyWebNodeProcesses(ctx, state, in.NodeIDs, lookup); err != nil {
				return out, err
			}
			if in.Retention == "cleanup" {
				return out, errors.New("node controls preserve the network; cleanup applies to composition and test jobs")
			}
		}
	} else if !os.IsNotExist(err) {
		return out, err
	} else if webNodeControlOperation(in.Operation) {
		return out, ErrDeploymentNotFound
	}
	switch in.Operation {
	case "chain.setup", "chain.deploy":
		if p.Keys.SHA256 == "" {
			p.Keys, err = e.pinKeys(ctx)
			if err != nil {
				return out, err
			}
		}
		if len(in.NodeIDs) != 0 {
			return out, errors.New("composition does not select existing nodes")
		}
		out.Phases = []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init"}
		if in.Operation == "chain.deploy" {
			out.Phases = append(out.Phases, "start")
		}
	case "test.run":
		out.Phases = []string{"test.inputs", "test.binary", "test.run"}
	default:
		out.Phases = []string{in.Operation}
	}
	out.RequiredAccess = []string{p.Target.Transport + " filesystem and process access"}
	out.Changes = []string{fmt.Sprintf("%s: %d block producers using verified %s binary", in.Operation, p.Arguments.Validators, plugin.Protocol().Name)}
	out.Changes = append(out.Changes, fmt.Sprintf("Binary asset %s · SHA-256 %s", p.Binary.ID, p.Binary.SHA256))
	if webNodeControlOperation(in.Operation) {
		out.Changes = append(out.Changes, fmt.Sprintf("Verified node executable: %s · SHA-256 %s", p.ExecutionBinary, p.Binary.SHA256))
	}
	if in.Operation == "node.reset" {
		out.Changes = append(out.Changes, "Replace only the selected non-producer's node data with its recorded genesis; leave it stopped and preserve sibling nodes")
	}
	if p.TestRun != nil {
		display := p.TestRun.Plan
		if len(p.TestRun.Genesis) > 0 {
			display.Genesis.Existing = "asset:" + p.TestRun.Genesis[0].Asset.ID
		}
		out.Changes = append(out.Changes, display.String())
		for _, g := range p.TestRun.Genesis {
			out.Changes = append(out.Changes, fmt.Sprintf("Finished test genesis asset %s · SHA-256 %s", g.Asset.ID, g.Asset.Checksum))
		}
		for _, n := range placement.Placements() {
			out.Changes = append(out.Changes, fmt.Sprintf("Test placement %s=%s · P2P %d · RPC %d", n.Label, n.Role, n.Ports.P2P, n.Ports.HTTP))
		}
		for _, c := range p.TestRun.Cases {
			out.Changes = append(out.Changes, fmt.Sprintf("Test case %s · r%d · %s", c.Document.ID, c.Document.Revision, c.Document.Name))
		}
	}
	if p.Preset != nil {
		if p.Preset.Genesis != nil {
			out.Changes = append(out.Changes, fmt.Sprintf("Finished genesis asset %s · SHA-256 %s", p.Preset.Genesis.Asset.ID, p.Preset.Genesis.Asset.Checksum))
		}
		out.Changes = append(out.Changes, fmt.Sprintf("Chain preset %s · r%d · %s", p.Preset.Document.ID, p.Preset.Document.Revision, p.Preset.Document.Name), p.Preset.Plan.String())
	}
	if p.Keys.SHA256 != "" {
		out.Changes = append(out.Changes, "Key material pinned to SHA-256 "+p.Keys.SHA256)
	}
	out.Payload, err = json.Marshal(p)
	if err != nil {
		return out, err
	}
	out.Fingerprint = manifestHash(out.Payload)
	return out, nil
}

func webSelectedNode(state State, ids []string) error {
	if len(ids) != 1 {
		return errors.New("select exactly one owned node")
	}
	for _, ns := range state.Nodes {
		if string(ns.NodeLabel()) == ids[0] {
			return nil
		}
	}
	return ErrDeploymentNotFound
}

func (e *WebChainEngine) Execute(ctx context.Context, a DeploymentActor, prepared WebPreparedJob, report func(WebJobPhase) error) (WebJobResult, error) {
	var p webChainPayload
	if err := json.Unmarshal(prepared.Payload, &p); err != nil {
		return WebJobResult{}, err
	}
	if err := e.allowed(a); err != nil {
		return WebJobResult{}, err
	}
	if manifestHash(prepared.Payload) != prepared.Fingerprint {
		return WebJobResult{}, ErrDeploymentConflict
	}
	// Start already checked the latest document revisions before accepting.
	// Here recheck live targets and assets using that accepted snapshot, so
	// later edits cannot redirect it or invalidate its pinned shared inputs.
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return WebJobResult{}, err
	}
	server, err := set.ByName(p.Arguments.ServerRef)
	if err != nil {
		return WebJobResult{}, err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return WebJobResult{}, err
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return WebJobResult{}, err
	}
	target, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, resource.TargetOf(server, wc.DataRoot))
	if err != nil {
		return WebJobResult{}, err
	}
	if target != p.Target {
		return WebJobResult{}, ErrDeploymentConflict
	}
	control, err := (resource.Opener{}).Inspect(ctx, resource.Spec{DataRoot: p.ControlDir})
	if err != nil {
		return WebJobResult{}, err
	}
	if len(prepared.Claims) != 2 || control.HostIdentity != prepared.Claims[1].HostIdentity || control.DataPath != prepared.Claims[1].DataPath {
		return WebJobResult{}, ErrDeploymentConflict
	}
	plugin, err := ValidateManifest(p.Manifest.ManifestInput)
	if err != nil {
		return WebJobResult{}, err
	}
	asset, err := e.binaryAsset(p.Arguments.AssetID)
	if err != nil {
		return WebJobResult{}, err
	}
	binary, err := asset.Verify(ctx, plugin.Protocol().Name)
	if err != nil {
		return WebJobResult{}, err
	}
	if binary != p.Binary {
		return WebJobResult{}, ErrDeploymentConflict
	}
	record, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
	if err != nil && !os.IsNotExist(err) {
		return WebJobResult{}, err
	}
	if (err == nil && manifestHash(record) != p.RecordDigest) || (os.IsNotExist(err) && p.RecordDigest != "") {
		return WebJobResult{}, ErrDeploymentConflict
	}
	if webNodeControlOperation(p.Input.Operation) {
		var state State
		if err = json.Unmarshal(record, &state); err != nil {
			return WebJobResult{}, err
		}
		bound, err := bindWebControlBinary(ctx, state, p, lookup)
		if err != nil {
			return WebJobResult{}, err
		}
		if bound != p.ExecutionBinary {
			return WebJobResult{}, ErrDeploymentConflict
		}
	}
	return e.execute(ctx, a, p, report)
}

func (e *WebChainEngine) phase(ctx context.Context, a DeploymentActor, name string, report func(WebJobPhase) error, run func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.allowed(a); err != nil {
		return err
	}
	p := WebJobPhase{Name: name, State: "running", StartedAt: time.Now().UTC()}
	if err := report(p); err != nil {
		return err
	}
	err := run()
	p.FinishedAt = time.Now().UTC()
	p.State = "succeeded"
	if err != nil {
		p.State = "failed"
		p.Message = err.Error()
	}
	if recordErr := report(p); err == nil {
		err = recordErr
	}
	return err
}

// Cleanup affects only this engine's owned composition; attach has no such record.
func (e *WebChainEngine) Cleanup(ctx context.Context, a DeploymentActor, prepared WebPreparedJob) (WebJobResult, error) {
	if err := ctx.Err(); err != nil {
		return WebJobResult{}, err
	}
	if err := e.allowed(a); err != nil {
		return WebJobResult{}, err
	}
	var p webChainPayload
	if err := json.Unmarshal(prepared.Payload, &p); err != nil {
		return WebJobResult{}, err
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return WebJobResult{}, err
	}
	_, err = ChainRm(ctx, Deps{Command: "web owned composition cleanup", ServerLookup: lookup}, ChainRmIn{DataDir: p.ControlDir})
	if err != nil {
		return WebJobResult{NodeDisposition: "cleanup_failed", UnresolvedResources: []string{p.ControlDir}}, err
	}
	return WebJobResult{NodeDisposition: "cleaned", PartialEffects: []string{"owned composition cleaned"}}, nil
}
