package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

func (e *WebChainEngine) execute(ctx context.Context, a DeploymentActor, p webChainPayload, report func(WebJobPhase) error) (WebJobResult, error) {
	if p.Input.Operation == "test.run" {
		return e.executeTestRun(ctx, a, p, report)
	}
	result := WebJobResult{NodeDisposition: "retained", PartialEffects: []string{}}
	composition := ChainUpIn{BPCount: p.Arguments.Validators}
	if p.Preset != nil {
		current, err := e.projectPresetComposition(ctx, p)
		if err != nil {
			return result, err
		}
		before, _ := json.Marshal(p.Preset)
		after, _ := json.Marshal(current)
		if string(before) != string(after) {
			return result, ErrDeploymentConflict
		}
		composition = current.Request
	}
	keysDir := ""
	if p.Input.Operation == "chain.setup" || p.Input.Operation == "chain.deploy" {
		var err error
		keysDir, err = e.materializeKeys(ctx, p.Keys)
		if err != nil {
			return result, err
		}
	}
	d := Deps{Command: "web " + p.Input.Operation + " by " + a.ID}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return result, err
	}
	d.ServerLookup = lookup
	if webNodeControlOperation(p.Input.Operation) {
		err = e.phase(ctx, a, p.Input.Operation, report, func() error {
			raw, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
			if err != nil {
				return err
			}
			if manifestHash(raw) != p.RecordDigest {
				return ErrDeploymentConflict
			}
			var state State
			if err = json.Unmarshal(raw, &state); err != nil {
				return err
			}
			if err = validateWebChainRecord(state, p); err != nil {
				return err
			}
			if err = webSelectedNode(state, p.Input.NodeIDs); err != nil {
				return err
			}
			index := 0
			for _, ns := range state.Nodes {
				if string(ns.NodeLabel()) == p.Input.NodeIDs[0] {
					index = ns.Index
				}
			}
			bound, err := bindWebControlBinary(ctx, state, p, lookup)
			if err != nil {
				return err
			}
			if bound != p.ExecutionBinary {
				return ErrDeploymentConflict
			}
			if p.Input.Operation == "node.reset" {
				if err = verifyWebResetInputs(ctx, state, p, lookup); err != nil {
					return err
				}
			}
			if err = verifyWebNodeProcesses(ctx, state, p.Input.NodeIDs, lookup); err != nil {
				return err
			}
			if p.Input.Operation == "node.start" {
				_, err := NodeStart(ctx, d, NodeStartIn{DataDir: p.ControlDir, Index: index})
				return err
			}
			if p.Input.Operation == "node.reset" {
				err := NodeReset(ctx, d, NodeResetIn{DataDir: p.ControlDir, Index: index})
				if err != nil {
					result.UnresolvedResources = []string{p.ControlDir, p.Target.DataPath}
					result.PartialEffects = append(result.PartialEffects, "Node reset attempted; the selected node may be stopped or partially initialized")
				}
				return err
			}
			return NodeStop(ctx, d, NodeStopIn{DataDir: p.ControlDir, Index: index})
		})
		if err == nil {
			result.PartialEffects = append(result.PartialEffects, p.Input.Operation+" completed")
		}
		return result, err
	}
	asset, err := e.binaryAsset(p.Arguments.AssetID)
	if err != nil {
		return result, err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return result, err
	}
	setPath, configPath := filepath.Join(p.ControlDir, "server-set.yaml"), filepath.Join(p.ControlDir, "workspace-config.yaml")
	steps := []struct {
		name string
		run  func() error
	}{
		{"new", func() error {
			if err := recheckWebNetworkRecord(ctx, p, lookup, true); err != nil {
				return err
			}
			set, err := ExportDeploymentDocument(p.Set.DeploymentDocumentInput, "yaml")
			if err != nil {
				return err
			}
			config, err := ExportDeploymentDocument(p.Config.DeploymentDocumentInput, "yaml")
			if err != nil {
				return err
			}
			files := filestore.Local{}
			if err = files.Write(ctx, setPath, set, 0600); err != nil {
				return err
			}
			if err = files.Write(ctx, configPath, config, 0600); err != nil {
				return err
			}
			setDeclaration, err := deploymentSet(p.Set.DeploymentDocumentInput)
			if err != nil {
				return err
			}
			server, err := setDeclaration.ByName(p.Arguments.ServerRef)
			if err != nil {
				return err
			}
			in := ChainNewIn{DataDir: p.ControlDir, Chain: p.Manifest.ID, Binary: p.ExecutionBinary, KeysDir: keysDir, Target: resource.TargetOf(server, wc.DataRoot), ServerSet: setPath, WorkspaceConfigPath: configPath}
			if p.Manifest.Source == "external" {
				in.ManifestPath, in.TemplatePath, err = e.manifests.files.Pin(p.ControlDir, p.Manifest.Manifest, []byte(p.Manifest.Template))
				if err != nil {
					return err
				}
			}
			if _, err = ChainNew(ctx, d, in); err != nil {
				return err
			}
			metadata, err := json.Marshal(p.Target)
			if err != nil {
				return err
			}
			return files.Write(ctx, filepath.Join(p.ControlDir, "web-target.json"), metadata, 0600)
		}},
		{"place", func() error {
			_, err := ChainAllocate(ctx, d, ChainAllocateIn{DataDir: p.ControlDir, BPCount: composition.BPCount, ENCount: composition.ENCount, PNCount: composition.PNCount, EndpointSyncMode: composition.EndpointSyncMode, Topology: composition.Topology, Binaries: composition.Binaries, BinaryChains: composition.BinaryChains, Peering: composition.Peering, Server: resource.ServerRef{SetPath: setPath, Name: p.Arguments.ServerRef}})
			return err
		}},
		{"keys", func() error {
			_, err := ChainKeys(ctx, d, ChainKeysIn{DataDir: p.ControlDir})
			return err
		}},
		{"genesis", func() error {
			if p.Preset != nil && p.Preset.Genesis != nil {
				g := p.Preset.Genesis
				path, err := e.manifests.assets.MaterializeFile(ctx, g.Asset.ID, g.Asset.Checksum, g.Fingerprint)
				if err != nil || path != g.Path {
					return ErrDeploymentConflict
				}
			}
			_, err := ChainGenesis(ctx, d, ChainGenesisIn{DataDir: p.ControlDir, ChainID: composition.ChainID, Set: composition.GenesisSet, OverlayPath: composition.OverlayPath, GenesisExisting: composition.GenesisExisting})
			return err
		}},
		{"config", func() error {
			_, err := ChainConfig(ctx, d, ChainConfigIn{DataDir: p.ControlDir, ScopedSet: composition.ConfigSet})
			return err
		}},
		{"build", func() error {
			_, err := ChainLaunchOpts(ctx, d, ChainLaunchOptsIn{DataDir: p.ControlDir, Set: composition.LaunchSet, ScopedSet: composition.LaunchScoped})
			return err
		}},
		{"deploy", func() error {
			if p.Target.Transport == "ssh" {
				set, err := deploymentSet(p.Set.DeploymentDocumentInput)
				if err != nil {
					return err
				}
				server, err := set.ByName(p.Arguments.ServerRef)
				if err != nil {
					return err
				}
				access, err := (resource.Opener{Lookup: lookup}).Open(resource.TargetOf(server, wc.DataRoot))
				if err != nil {
					return err
				}
				b, err := os.ReadFile(asset.Path)
				if err != nil {
					return err
				}
				if manifestHash(b) != p.Binary.SHA256 {
					return ErrDeploymentConflict
				}
				if err = access.Files.Write(ctx, p.ExecutionBinary, b, 0755); err != nil {
					return err
				}
				checksum, err := access.Files.Checksum(ctx, p.ExecutionBinary)
				if err != nil {
					return err
				}
				if checksum != "sha256:"+p.Binary.SHA256 {
					return errors.New("remote binary checksum differs from accepted asset")
				}
			}
			_, err := ChainProvision(ctx, d, ChainProvisionIn{DataDir: p.ControlDir})
			return err
		}},
		{"init", func() error { _, err := ChainInit(ctx, d, ChainInitIn{DataDir: p.ControlDir}); return err }},
	}
	if p.Input.Operation == "chain.deploy" {
		steps = append(steps, struct {
			name string
			run  func() error
		}{"start", func() error { _, err := ChainStart(ctx, d, ChainStartIn{DataDir: p.ControlDir}); return err }})
	}
	for _, step := range steps {
		if err = e.phase(ctx, a, step.name, report, step.run); err != nil {
			result.UnresolvedResources = []string{p.ControlDir, wc.DataRoot}
			return result, err
		}
		result.PartialEffects = append(result.PartialEffects, step.name+" completed")
	}
	return result, nil
}

func validateWebChainRecord(state State, p webChainPayload) error {
	var manifest struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(p.Manifest.Manifest, &manifest); err != nil {
		return err
	}
	if state.Chain != manifest.ID || state.Binary != p.ExecutionBinary {
		return errors.New("selected manifest or binary differs from the owned network")
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return err
	}
	return validateWebControlLayout(state, p, wc)
}
