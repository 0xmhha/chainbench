package chainsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Checking a composition against the environment's approved baseline.
//
// The reuse check asks "is what is running what this run would build?"; both
// sides of that move with the run. The baseline asks the different question a
// regression test needs: "is what is running what a person approved?" — so a
// prepared genesis edited on the server is caught instead of silently adopted.
//
// A run only ever READS the baseline. Nothing here writes one: a run that
// refreshed the baseline would re-approve whatever it found, which is exactly
// the drift the record exists to catch. Approving is a separate, explicit act.

// ObserveBaseline reports the fingerprint of what this workspace composed: the
// genesis and per-node config hashes it recorded when it wrote them, and the
// validator addresses its key set derives. Only public identities are read —
// never key material.
func (w *Workspace) ObserveBaseline() (resource.Observed, error) {
	obs := resource.Observed{Configs: map[string]string{}}
	obs.Genesis = w.state.LaunchInputs[w.state.GenesisPath]
	for _, ns := range w.state.Nodes {
		if h := w.state.LaunchInputs[ns.ConfigPath]; h != "" {
			obs.Configs[string(ns.NodeLabel())] = h
		}
	}
	if w.state.KeysDir != "" {
		preset, err := store.LoadPreset(w.state.KeysDir)
		if err != nil {
			return obs, fmt.Errorf("chainsetup: baseline: load keys: %w", err)
		}
		obs.Validators = preset.NetworkFor(w.state.Validators).Validators
	}
	return obs, nil
}

// BaselineCheck is the outcome of comparing a composition to its environment's
// approved baseline.
type BaselineCheck struct {
	// Path is the baseline file consulted.
	Path string `json:"path"`
	// Approved reports whether a baseline exists to check against.
	Approved bool `json:"approved"`
	// Match is true when the composition is exactly what was approved.
	Match bool `json:"match"`
	// Diffs name what drifted, when Match is false.
	Diffs []string `json:"diffs,omitempty"`
}

// CheckBaseline compares this workspace's composition against the baseline
// approved for its workspace-config. A workspace with no workspace-config, or
// an environment with no approved baseline, reports Approved=false and is not a
// failure — the caller decides whether to demand one.
func (w *Workspace) CheckBaseline() (BaselineCheck, error) {
	if w.state.WorkspaceConfig == "" {
		return BaselineCheck{}, fmt.Errorf("chainsetup: baseline: this workspace was composed without a --workspace-config, so it has no environment to hold a baseline")
	}
	path := resource.BaselinePathFor(w.state.WorkspaceConfig)
	out := BaselineCheck{Path: path}
	b, err := resource.LoadBaseline(path)
	if err != nil {
		return out, err
	}
	if b == nil {
		return out, nil
	}
	out.Approved = true
	obs, err := w.ObserveBaseline()
	if err != nil {
		return out, err
	}
	out.Diffs = b.Check(obs)
	out.Match = len(out.Diffs) == 0
	return out, nil
}

// NetBaselineIn names the workspace to check or approve.
type NetBaselineIn struct {
	DataDir string `json:"dataDir,omitempty"`
	// Note is the approver's reason, recorded with an approval.
	Note string `json:"note,omitempty"`
	// ApprovedAt stamps the approval (RFC3339); the surface supplies it so the
	// record does not depend on a clock read here.
	ApprovedAt string `json:"approvedAt,omitempty"`
}

// NetBaselineOut carries the check result.
type NetBaselineOut struct {
	Check BaselineCheck `json:"check"`
}

// NetBaselineCheck compares a composed workspace against its environment's
// approved baseline. It writes nothing.
func NetBaselineCheck(_ context.Context, d Deps, in NetBaselineIn) (NetBaselineOut, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetBaselineOut{}, err
	}
	ws.SetEnv(d.Env)
	check, err := ws.CheckBaseline()
	return NetBaselineOut{Check: check}, err
}

// NetBaselineApprove records this composition as the environment's approved
// baseline. It is the one path that writes a baseline, and it exists as its own
// verb because approving is a decision a person makes after looking at what
// changed — never a side effect of a run.
func NetBaselineApprove(_ context.Context, d Deps, in NetBaselineIn) (NetBaselineOut, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetBaselineOut{}, err
	}
	ws.SetEnv(d.Env)
	if ws.state.WorkspaceConfig == "" {
		return NetBaselineOut{}, fmt.Errorf("chainsetup: baseline: this workspace was composed without a --workspace-config, so there is no environment to approve a baseline for")
	}
	obs, err := ws.ObserveBaseline()
	if err != nil {
		return NetBaselineOut{}, err
	}
	if obs.Genesis == "" && len(obs.Configs) == 0 && len(obs.Validators) == 0 {
		return NetBaselineOut{}, fmt.Errorf("chainsetup: baseline: nothing to approve — compose the network first")
	}
	path := resource.BaselinePathFor(ws.state.WorkspaceConfig)
	b := resource.Baseline{
		Version: 1, ApprovedAt: strings.TrimSpace(in.ApprovedAt), Note: strings.TrimSpace(in.Note),
		Genesis: obs.Genesis, Configs: obs.Configs, Validators: obs.Validators,
	}
	if err := resource.SaveBaseline(path, b); err != nil {
		return NetBaselineOut{}, err
	}
	return NetBaselineOut{Check: BaselineCheck{Path: path, Approved: true, Match: true}}, nil
}
