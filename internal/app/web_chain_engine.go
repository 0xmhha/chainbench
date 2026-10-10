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
	"slices"
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
	monitor    *WebMonitor
	// verifyAsset checks a registered binary against a chain's build
	// contract; nil uses the native probe.
	verifyAsset func(context.Context, ManifestBinary, string) (ManifestBinaryEvidence, error)
}

// errIncompatibleBinary refuses a registered binary of another chain.
var errIncompatibleBinary = errors.New("binary identity is incompatible with selected chain")

func (e *WebChainEngine) verify(ctx context.Context, asset ManifestBinary, chain string) (ManifestBinaryEvidence, error) {
	if e.verifyAsset != nil {
		return e.verifyAsset(ctx, asset, chain)
	}
	return asset.Verify(ctx, chain)
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
	ManifestID         string                  `json:"manifestId"`
	AssetID            string                  `json:"assetId"`
	ServerRef          string                  `json:"serverRef"`
	Validators         int                     `json:"validators"`
	ReplacementAssetID string                  `json:"replacementAssetId,omitempty"`
	ConfigOverrides    map[string]string       `json:"configOverrides,omitempty"`
	CaseRefs           []DeploymentDocumentRef `json:"caseRefs,omitempty"`
	// BinaryAssets maps each binary name a test declares besides default
	// to the registered asset that runs under it.
	BinaryAssets   map[string]string      `json:"binaryAssets,omitempty"`
	ChainPresetRef *DeploymentDocumentRef `json:"chainPresetRef,omitempty"`
}
type webChainPayload struct {
	Input              WebPlanInput           `json:"input"`
	Arguments          webChainArguments      `json:"arguments"`
	WorkspaceRevision  int                    `json:"workspaceRevision"`
	Set                DeploymentDocument     `json:"set"`
	Config             DeploymentDocument     `json:"config"`
	Manifest           ManagedManifest        `json:"manifest"`
	Binary             ManifestBinaryEvidence `json:"binary"`
	ControlDir         string                 `json:"controlDir"`
	RecordDigest       string                 `json:"recordDigest"`
	Target             resource.Inspection    `json:"target"`
	ExecutionBinary    string                 `json:"executionBinary"`
	Keys               webKeySnapshot         `json:"keys"`
	TestRun            *webTestRun            `json:"testRun,omitempty"`
	Attach             *webTestAttach         `json:"attach,omitempty"`
	NamedBinaries      []webNamedBinary       `json:"namedBinaries,omitempty"`
	Preset             *webPresetComposition  `json:"preset,omitempty"`
	CurrentNodeBinary  *webNodeBinary         `json:"currentNodeBinary,omitempty"`
	NodeBindingsDigest string                 `json:"nodeBindingsDigest,omitempty"`
	Replacement        *webBinaryReplacement  `json:"replacement,omitempty"`
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
	if in.Operation != "chain.setup" && in.Operation != "chain.deploy" && !webNodeControlOperation(in.Operation) && in.Operation != "test.run" && in.Operation != webTestAttachOperation && in.Operation != webNetworkMonitorOperation {
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
	if in.Operation == webNetworkMonitorOperation {
		return e.prepareMonitor(ctx, a, in, args)
	}
	if args.ReplacementAssetID != "" && in.Operation != "node.swap" {
		return out, errors.New("a replacement executable requires a node replacement job")
	}
	if in.Operation != "node.swap" && args.ConfigOverrides != nil {
		return out, errors.New("configuration changes require a node replacement job")
	}
	if args.ChainPresetRef != nil && (in.Operation != "chain.setup" && in.Operation != "chain.deploy" || args.Validators != 0) {
		return out, errors.New("a saved chain preset applies to composition and supplies its own node layout")
	}
	if args.Validators == 0 && in.Operation != "test.run" && !webRecordedOperation(in.Operation) && args.ChainPresetRef == nil {
		args.Validators = 4
	}
	if in.Operation != "test.run" && !webRecordedOperation(in.Operation) && args.ChainPresetRef == nil && (args.Validators < 1 || args.Validators > 128) {
		return out, errors.New("validator count must be between 1 and 128")
	}
	workspace, err := e.documents.Workspace(in.WorkspaceID)
	if err != nil {
		return out, err
	}
	if len(in.AssetRefs) < 1 || len(in.AssetRefs) > 257 || args.ChainPresetRef == nil && !webCaseOperation(in.Operation) && args.ReplacementAssetID == "" && (len(in.AssetRefs) != 1 || in.AssetRefs[0] != args.AssetID) {
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
	var controlState *State
	if webRecordedOperation(in.Operation) {
		if len(args.CaseRefs) != 0 && in.Operation != webTestAttachOperation {
			return out, errors.New("case references belong to a test job")
		}
		if in.Operation == webTestAttachOperation && (len(in.NodeIDs) != 0 || args.Validators != 0 || in.Retention == "cleanup") {
			return out, errors.New("an attach job reads the whole recorded network and never removes it")
		}
		record, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
		if os.IsNotExist(err) {
			return out, ErrDeploymentNotFound
		}
		if err != nil {
			return out, err
		}
		controlState = &State{}
		if err = json.Unmarshal(record, controlState); err != nil {
			return out, err
		}
		p.RecordDigest = manifestHash(record)
		if in.Operation == webTestAttachOperation {
			if err = e.prepareTestAttach(ctx, &p, *controlState); err != nil {
				return out, err
			}
		}
	} else if in.Operation == "test.run" {
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
	var placement *node.Map
	if controlState != nil {
		// Claims come from the owned network, including non-producers and
		// inventory-assigned slots, rather than a new producer-only allocation.
		placement, err = webRecordedControlPlacement(*controlState, &p, pool)
	} else {
		placement, err = resource.Assign(pool, requests)
	}
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
	slices.Sort(ports)
	ports = slices.Compact(ports)
	controlTarget, err := (resource.Opener{}).Inspect(ctx, resource.Spec{DataRoot: p.ControlDir})
	if err != nil {
		return out, err
	}
	expectedAssets := map[string]bool{args.AssetID: true}
	if args.ReplacementAssetID != "" {
		expectedAssets[args.ReplacementAssetID] = true
	}
	if p.Preset != nil {
		for _, id := range p.Preset.Document.AssetRefs {
			expectedAssets[id] = true
		}
	}
	for _, named := range p.NamedBinaries {
		expectedAssets[named.AssetID] = true
	}
	if p.TestRun != nil {
		for _, c := range p.TestRun.Cases {
			for _, id := range c.Document.AssetRefs {
				expectedAssets[id] = true
			}
		}
	}
	if p.Attach != nil {
		for _, c := range p.Attach.Cases {
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
	if webLaunchesNodes(in.Operation) {
		out.Claims = append(out.Claims, webExecutableClaims(p)...)
	}
	record, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
	if err == nil {
		if controlState != nil && manifestHash(record) != p.RecordDigest {
			return out, ErrDeploymentConflict
		}
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
			if err = verifyWebNetworkProcesses(ctx, state, p, lookup, true); err != nil {
				return out, err
			}
		}
		if in.Operation == webTestAttachOperation {
			if err = verifyWebNetworkProcesses(ctx, state, p, lookup, false); err != nil {
				return out, err
			}
			if p.ExecutionBinary, err = bindWebControlBinary(ctx, state, p, lookup); err != nil {
				return out, err
			}
			if err = validateWebChainRecord(state, p); err != nil {
				return out, err
			}
			if p.Keys, err = e.bindWebConfigKeys(ctx, state); err != nil {
				return out, err
			}
		}
		if webNodeControlOperation(in.Operation) {
			p.CurrentNodeBinary, p.NodeBindingsDigest, err = e.currentWebNodeBinary(ctx, state, p, lookup)
			if err != nil {
				return out, err
			}
			if err = e.prepareWebBinaryReplacement(ctx, state, &p); err != nil {
				return out, err
			}
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
			if err = verifyWebRecordLedger(state, p.ControlDir); err != nil {
				return out, err
			}
			if in.Operation == "node.reset" {
				if err = verifyWebResetInputs(ctx, state, p, lookup); err != nil {
					return out, err
				}
			}
			if in.Operation == "node.restart" {
				if err = verifyWebNodeInputs(ctx, state, p, lookup); err != nil {
					return out, err
				}
			}
			if in.Operation == "node.swap" {
				if _, err = webConfigChanges(state, p); err != nil {
					return out, err
				}
				if err = verifyWebNodeInputs(ctx, state, p, lookup); err != nil {
					return out, err
				}
				if len(p.Arguments.ConfigOverrides) > 0 {
					p.Keys, err = e.bindWebConfigKeys(ctx, state)
					if err != nil {
						return out, err
					}
				}
				if p.Replacement != nil {
					if err = inspectWebBinaryParent(ctx, p, p.Replacement.Path, lookup); err != nil {
						return out, err
					}
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
	} else if webRecordedOperation(in.Operation) {
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
	case webTestAttachOperation:
		out.Phases = []string{webTestAttachOperation}
	default:
		out.Phases = []string{in.Operation}
	}
	out.RequiredAccess = []string{p.Target.Transport + " filesystem and process access"}
	out.Changes = []string{fmt.Sprintf("%s: %d block producers using verified %s binary", in.Operation, p.Arguments.Validators, plugin.Protocol().Name)}
	out.Changes = append(out.Changes, fmt.Sprintf("Binary asset %s · SHA-256 %s", p.Binary.ID, p.Binary.SHA256))
	for _, named := range p.NamedBinaries {
		out.Changes = append(out.Changes, fmt.Sprintf("Binary %q runs %s asset %s · SHA-256 %s at %s", named.Name, named.Binary.Chain, named.Binary.ID, named.Binary.SHA256, named.ExecutionBinary))
	}
	if webNodeControlOperation(in.Operation) {
		out.Changes = append(out.Changes, fmt.Sprintf("Verified node executable: %s · SHA-256 %s", p.ExecutionBinary, p.Binary.SHA256))
		for _, n := range placement.Placements() {
			out.Changes = append(out.Changes, fmt.Sprintf("Recorded placement %s=%s · P2P %d · RPC %d", n.Label, n.Role, n.Ports.P2P, n.Ports.HTTP))
		}
	}
	if in.Operation == "node.reset" {
		out.Changes = append(out.Changes, "Replace only the selected non-producer's node data with its recorded genesis; leave it stopped and preserve sibling nodes")
	}
	if in.Operation == "node.restart" {
		out.Changes = append(out.Changes, "Restart only the selected owned node with its recorded executable and arguments; preserve config, genesis, node data and sibling processes")
	}
	if in.Operation == "node.swap" {
		changes, err := webConfigChanges(*controlState, p)
		if err != nil {
			return out, err
		}
		if len(changes) > 0 {
			out.Changes = append(out.Changes, "Replace only the selected generated config and rebuild its recorded launch arguments; preserve genesis, data and sibling nodes")
			out.Changes = append(out.Changes, changes...)
		}
	}
	if p.CurrentNodeBinary != nil {
		out.Changes = append(out.Changes, fmt.Sprintf("Current node binary asset %s · SHA-256 %s", p.CurrentNodeBinary.Evidence.ID, p.CurrentNodeBinary.Evidence.SHA256))
	}
	if p.Replacement != nil {
		out.Changes = append(out.Changes, fmt.Sprintf("Replace only %s executable with registered asset %s · SHA-256 %s; preserve genesis, node data and sibling processes", p.Input.NodeIDs[0], p.Replacement.Evidence.ID, p.Replacement.Evidence.SHA256))
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
	if p.Attach != nil {
		out.Changes = append(out.Changes, "Read-only run against the recorded network: its endpoints, key set and capabilities replace the ones each case declares; no node is started, stopped or removed")
		for _, n := range p.Attach.Nodes.Nodes {
			out.Changes = append(out.Changes, fmt.Sprintf("Attach node%d=%s · RPC %s", n.Index, n.Role, n.RPCURL))
		}
		for _, c := range p.Attach.Cases {
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
	if p.Input.Operation == webNetworkMonitorOperation {
		return e.executeMonitor(ctx, a, p, report)
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
	if len(prepared.Claims) < 2 || control.HostIdentity != prepared.Claims[1].HostIdentity || control.DataPath != prepared.Claims[1].DataPath {
		return WebJobResult{}, ErrDeploymentConflict
	}
	launches := []WebResourceClaim{}
	if webLaunchesNodes(p.Input.Operation) {
		launches = webExecutableClaims(p)
	}
	if len(prepared.Claims) != 2+len(launches) {
		return WebJobResult{}, ErrDeploymentConflict
	}
	for i, launch := range launches {
		if got := prepared.Claims[2+i]; got.HostIdentity != launch.HostIdentity || got.Executable != launch.Executable || got.DataPath != "" || len(got.Ports) != 0 {
			return WebJobResult{}, ErrDeploymentConflict
		}
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
	for _, named := range p.NamedBinaries {
		asset, err := e.binaryAsset(named.AssetID)
		if err != nil {
			return WebJobResult{}, err
		}
		if now, err := e.verify(ctx, asset, named.Binary.Chain); err != nil || now != named.Binary {
			return WebJobResult{}, ErrDeploymentConflict
		}
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
	if p.Input.WorkspaceID == "" || p.ControlDir != filepath.Join(e.root, "networks", p.Input.WorkspaceID) {
		return WebJobResult{}, ErrDeploymentConflict
	}
	if err = recheckWebNetworkRecord(ctx, p, lookup, false); err != nil {
		return WebJobResult{NodeDisposition: "cleanup_failed", UnresolvedResources: []string{p.ControlDir, p.Target.DataPath}}, err
	}
	d := Deps{Command: "web owned composition cleanup", ServerLookup: lookup}
	// The recheck above verified every recorded process as this network's own;
	// a test or deployment leaves them running, and removal requires them stopped.
	if _, err = ChainStop(ctx, d, ChainStopIn{DataDir: p.ControlDir}); err != nil {
		return WebJobResult{NodeDisposition: "cleanup_failed", UnresolvedResources: []string{p.ControlDir, p.Target.DataPath}}, err
	}
	_, err = ChainRm(ctx, d, ChainRmIn{DataDir: p.ControlDir})
	if err != nil {
		return WebJobResult{NodeDisposition: "cleanup_failed", UnresolvedResources: []string{p.ControlDir}, PartialEffects: []string{"owned nodes stopped before removal"}}, err
	}
	return WebJobResult{NodeDisposition: "cleaned", PartialEffects: []string{"owned nodes stopped", "owned composition cleaned"}}, nil
}

// webLaunchesNodes reports whether an operation composes and launches a
// network, which the engine refuses while the same binary runs on the host
// outside the workspace; its plan claims that executable. Single-node
// controls are not checked by the engine and claim no executable.
func webLaunchesNodes(operation string) bool {
	return operation == "chain.deploy" || operation == "test.run"
}
