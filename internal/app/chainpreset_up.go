package app

import (
	"context"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// ChainUpFromPreset is the composition a chain-preset document declares, with
// the caller's own surface settings on top: where the workspace is, how far to
// go, which target, server set and workspace config, and, when given, which
// binary and key set. It is the same projection a test run applies to the
// preset its cases name, so composing a preset by hand and composing it for a
// test give the same network.
//
// Files the projection needs (a genesis overlay) are written under the
// workspace, because the genesis step reads them later.
func ChainUpFromPreset(ctx context.Context, raw []byte, base ChainUpIn) (ChainUpIn, error) {
	out, err := PlanChainPreset(ctx, raw, base)
	return out.Layout, err
}

// PresetPlan is the resolved request and its human-readable compose plan.
type PresetPlan struct {
	Layout ChainUpIn   `json:"layout"`
	Plan   ComposePlan `json:"plan"`
}

// PlanChainPreset is also the projection ChainUpFromPreset executes. Only
// content-addressed planning inputs are materialised; nodes are not touched.
func PlanChainPreset(ctx context.Context, raw []byte, base ChainUpIn) (PresetPlan, error) {
	in := RunSuiteIn{
		DataDir: filepath.Join(base.DataDir, "chain-preset-plan"), Chain: base.Chain, Binary: base.Binary,
		KeysDir: base.KeysDir, Server: base.Server, WorkspaceConfigPath: base.WorkspaceConfigPath, Docker: base.Docker,
	}
	if base.Binary != "" {
		in.BinaryOverrides = map[string]string{dsl.BinaryDefault: base.Binary}
	}
	req, plan, err := testengine.PlanChainPreset(ctx, raw, in)
	if err != nil {
		return PresetPlan{}, err
	}
	req.DataDir, req.Stage, req.Target = base.DataDir, base.Stage, base.Target
	req.Server, req.Docker, req.WorkspaceConfigPath = base.Server, base.Docker, base.WorkspaceConfigPath
	if base.ManifestPath != "" {
		req.ManifestPath, req.TemplatePath = base.ManifestPath, base.TemplatePath
	}
	if base.KeysDir != "" {
		req.KeysDir = base.KeysDir
	}
	if base.Binary != "" {
		req.Binary = base.Binary
	}
	if base.KeysSource != "" {
		req.KeysSource = base.KeysSource
	}
	plan.Workspace = base.DataDir
	return PresetPlan{Layout: req, Plan: plan}, nil
}
