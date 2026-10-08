package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

func (e *WebChainEngine) execute(ctx context.Context, a DeploymentActor, p webChainPayload, report func(WebJobPhase) error) (WebJobResult, error) {
	result := WebJobResult{NodeDisposition: "retained", PartialEffects: []string{}}
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
	if p.Input.Operation == "node.start" || p.Input.Operation == "node.stop" {
		index, err := strconv.Atoi(strings.TrimPrefix(p.Input.NodeIDs[0], "node"))
		if err != nil {
			return result, err
		}
		err = e.phase(ctx, a, p.Input.Operation, report, func() error {
			if p.Input.Operation == "node.start" {
				_, err := NodeStart(ctx, d, NodeStartIn{DataDir: p.ControlDir, Index: index})
				return err
			}
			return NodeStop(ctx, d, NodeStopIn{DataDir: p.ControlDir, Index: index})
		})
		if err == nil {
			result.PartialEffects = append(result.PartialEffects, p.Input.Operation+" completed")
		}
		return result, err
	}
	asset, ok := e.assets[p.Arguments.AssetID]
	if !ok {
		return result, ErrDeploymentNotFound
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
			_, err := ChainAllocate(ctx, d, ChainAllocateIn{DataDir: p.ControlDir, BPCount: p.Arguments.Validators, Server: resource.ServerRef{SetPath: setPath, Name: p.Arguments.ServerRef}})
			return err
		}},
		{"keys", func() error {
			_, err := ChainKeys(ctx, d, ChainKeysIn{DataDir: p.ControlDir, Nodes: p.Arguments.Validators})
			return err
		}},
		{"genesis", func() error { _, err := ChainGenesis(ctx, d, ChainGenesisIn{DataDir: p.ControlDir}); return err }},
		{"config", func() error { _, err := ChainConfig(ctx, d, ChainConfigIn{DataDir: p.ControlDir}); return err }},
		{"build", func() error { _, err := ChainLaunchOpts(ctx, d, ChainLaunchOptsIn{DataDir: p.ControlDir}); return err }},
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
	if state.Target.DataRoot != wc.DataRoot || state.BPCount != p.Arguments.Validators {
		return errors.New("selected placement differs from the owned network")
	}
	return nil
}
