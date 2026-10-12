package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

func (e *WebChainEngine) execute(ctx context.Context, a DeploymentActor, p webChainPayload, report func(WebJobPhase) error) (WebJobResult, error) {
	if p.Input.Operation == "test.run" {
		return e.executeTestRun(ctx, a, p, report)
	}
	if p.Input.Operation == webTestAttachOperation {
		return e.executeTestAttach(ctx, a, p, report)
	}
	result := WebJobResult{NodeDisposition: "retained", PartialEffects: []string{}}
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
			if err = verifyWebRecordLedger(state, p.ControlDir); err != nil {
				return err
			}
			index := 0
			for _, ns := range state.Nodes {
				if string(ns.NodeLabel()) == p.Input.NodeIDs[0] {
					index = ns.Index
				}
			}
			currentBinding, bindingsDigest, err := e.currentWebNodeBinary(ctx, state, p, lookup)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(currentBinding, p.CurrentNodeBinary) || p.NodeBindingsDigest != "" && bindingsDigest != p.NodeBindingsDigest {
				return ErrDeploymentConflict
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
			if p.Input.Operation == "node.restart" {
				if err = verifyWebNodeInputs(ctx, state, p, lookup); err != nil {
					return err
				}
			}
			if p.Input.Operation == "node.swap" {
				if _, err = webConfigChanges(state, p); err != nil {
					return err
				}
				if err = verifyWebNodeInputs(ctx, state, p, lookup); err != nil {
					return err
				}
				if len(p.Arguments.ConfigOverrides) > 0 || p.ConfigAsset != nil {
					keys, err := e.bindWebConfigKeys(ctx, state)
					if err != nil || keys != p.Keys {
						return ErrDeploymentConflict
					}
				}
				candidate := p
				if err = e.prepareWebBinaryReplacement(ctx, state, &candidate); err != nil {
					return err
				}
				if !reflect.DeepEqual(candidate.Replacement, p.Replacement) {
					return ErrDeploymentConflict
				}

			}
			if err = verifyWebNodeProcesses(ctx, state, p.Input.NodeIDs, lookup, p.Targets...); err != nil {
				return err
			}
			if p.Input.Operation == "node.start" {
				return e.runControl(ctx, a, p, "node", "start", "--index", strconv.Itoa(index))
			}
			if p.Input.Operation == "node.swap" {
				changes, err := webConfigChanges(state, p)
				if err != nil {
					return err
				}
				binary := ""
				if p.Replacement != nil {
					if err = e.stageWebBinaryReplacement(ctx, state, p, lookup); err != nil {
						result.UnresolvedResources = []string{p.ControlDir, p.Replacement.Path}
						result.PartialEffects = append(result.PartialEffects, "Replacement binary staging attempted; the selected node has not been stopped by this phase")
						return err
					}
					result.PartialEffects = append(result.PartialEffects, "Reviewed replacement executable staged and binding retained")
					result.UnresolvedResources = []string{p.ControlDir, p.Replacement.Path}
					raw, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
					if err != nil || manifestHash(raw) != p.RecordDigest {
						return ErrDeploymentConflict
					}
					if _, err = bindWebControlBinary(ctx, state, p, lookup); err != nil {
						return err
					}
					if err = verifyWebNodeInputs(ctx, state, p, lookup); err != nil {
						return err
					}
					if len(p.Arguments.ConfigOverrides) > 0 || p.ConfigAsset != nil {
						keys, err := e.bindWebConfigKeys(ctx, state)
						if err != nil || keys != p.Keys {
							return ErrDeploymentConflict
						}
					}
					if err = verifyWebNodeProcesses(ctx, state, p.Input.NodeIDs, lookup, p.Targets...); err != nil {
						return err
					}
					binary = p.Replacement.Path
				}
				err = e.runControl(ctx, a, p, nodeSwapArgs(index, binary, changes)...)
				if err != nil {
					result.UnresolvedResources = []string{p.ControlDir, p.Target.DataPath}
					result.PartialEffects = append(result.PartialEffects, "Node replacement attempted; the selected node may be stopped or its executable, config and launch incomplete")
				} else {
					result.UnresolvedResources = nil
				}
				return err
			}
			if p.Input.Operation == "node.restart" {
				err := e.runControl(ctx, a, p, "chain", "restart", "--node", strconv.Itoa(index))
				if err != nil {
					result.UnresolvedResources = []string{p.ControlDir, p.Target.DataPath}
					result.PartialEffects = append(result.PartialEffects, "Node restart attempted; the selected node may be stopped or its launch incomplete")
				}
				return err
			}
			if p.Input.Operation == "node.reset" {
				err := e.runControl(ctx, a, p, "node", "reset", "--index", strconv.Itoa(index))
				if err != nil {
					result.UnresolvedResources = []string{p.ControlDir, p.Target.DataPath}
					result.PartialEffects = append(result.PartialEffects, "Node reset attempted; the selected node may be stopped or partially initialized")
				}
				return err
			}
			return e.runControl(ctx, a, p, "node", "stop", "--index", strconv.Itoa(index))
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
	setPath, configPath := filepath.Join(p.ControlDir, "server-set.yaml"), filepath.Join(p.ControlDir, "workspace-config.yaml")
	setDeclaration, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return result, err
	}
	servers, err := webPlanServers(setDeclaration, p.Arguments, p.Input.Operation)
	if err != nil {
		return result, err
	}
	steps := []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init"}
	if p.Input.Operation == "chain.deploy" {
		steps = append(steps, "start")
	}
	failed, err := e.composeChain(ctx, a, p, asset, keysDir, servers, setPath, configPath, steps, lookup, report)
	if err != nil {
		// Known effects are named per node; anything present or uncheckable
		// stays unresolved, and the disposition says whether targets changed.
		result.NodeDisposition = webFailureDisposition(failed)
		result.UnresolvedResources = []string{p.ControlDir}
		if webTargetPhases[failed] {
			// A target that just failed may not answer; each check is bounded
			// and an unanswered node stays unresolved.
			checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
			effects, unresolved := e.nodeEffects(checkCtx, p, lookup)
			cancel()
			result.PartialEffects = append(result.PartialEffects, effects...)
			result.UnresolvedResources = append(result.UnresolvedResources, unresolved...)
		}
		return result, err
	}
	for _, step := range steps {
		result.PartialEffects = append(result.PartialEffects, step+" completed")
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
