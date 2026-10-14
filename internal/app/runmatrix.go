package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// RunMatrixIn runs one ordered case set on every selected declaration. Each
// target is a suite with its own session and composition workspace.
type RunMatrixIn struct {
	RunSuiteIn
	Presets []string
	BaseDir string
	Plan    bool
}

type PresetRunResult struct {
	Index         int          `json:"index"`
	PresetRef     string       `json:"preset"`
	PresetID      string       `json:"presetId"`
	Chain         string       `json:"chain"`
	Workspace     string       `json:"workspace,omitempty"`
	Attempted     bool         `json:"attempted"`
	Session       string       `json:"session,omitempty"`
	Error         string       `json:"error,omitempty"`
	FailedAt      string       `json:"failedAt,omitempty"`
	Stopped       bool         `json:"stopped"`
	CleanupFailed bool         `json:"cleanupFailed,omitempty"`
	SetupSteps    []string     `json:"setupSteps,omitempty"`
	Plan          *ComposePlan `json:"plan,omitempty"`
	Layout        *ChainUpIn   `json:"layout,omitempty"`
	RunSummary
}

type RunMatrixOut struct {
	Runs        []PresetRunResult `json:"runs"`
	Summary     session.Counts    `json:"summary"`
	SetupErrors int               `json:"setupErrors"`
}

// ExitCode has the same verdict precedence for CLI and MCP: infrastructure or
// blocked = 2, failing tests = 1, otherwise = 0. Strict skips count as 2.
func (out RunMatrixOut) ExitCode(noSkips bool) int {
	if out.SetupErrors > 0 || out.Summary.Blocked > 0 || noSkips && out.Summary.Skip > 0 {
		return 2
	}
	if out.Summary.Fail > 0 {
		return 1
	}
	return 0
}

type preparedMatrixTarget struct {
	result   PresetRunResult
	request  RunSuiteIn
	declared *AttachDecl
}

// RunMatrix preflights every target before the first execution. Failures during
// execution are retained per target and do not prevent later targets running.
// Cancellation prevents later targets being attempted. No parallel node sets
// are created by default; each suite finishes and stops before the next begins.
func RunMatrix(ctx context.Context, d Deps, in RunMatrixIn) (RunMatrixOut, error) {
	return runMatrix(ctx, d, in, RunSuite)
}

func runMatrix(ctx context.Context, d Deps, in RunMatrixIn, runSuite func(context.Context, Deps, RunSuiteIn) (RunSuiteOut, error)) (RunMatrixOut, error) {
	targets, err := prepareMatrix(ctx, in)
	if err != nil {
		return RunMatrixOut{}, err
	}
	out := RunMatrixOut{Runs: make([]PresetRunResult, 0, len(targets))}
	cleanupFailed := false
	for _, target := range targets {
		row := target.result
		if cleanupFailed {
			row.Error = "not run: the previous owned network could not be cleaned up"
		} else if err := ctx.Err(); err != nil {
			row.Error = "not run: " + err.Error()
		} else if in.Plan {
			out.Runs = append(out.Runs, row)
			continue
		} else {
			row.Attempted = true
			var runErr error
			if target.declared != nil {
				start, err := StartFor(Named{Spell: CLISpelling, Specs: func() ([][]byte, []string, error) { return target.request.SpecContent, nil, nil }})
				if err != nil {
					runErr = err
				} else {
					row.Session, runErr = AttachRun(ctx, d, AttachRunIn{At: start.At, Declared: start.Declared, Chain: in.Chain, ArtifactRoot: target.request.ArtifactRoot, Specs: target.request.SpecContent, KeysDir: in.KeysDir, ReadOnlyKeys: in.ReadOnlyKeys, NoNodeControl: in.NoNodeControl})
					if row.Session != "" {
						var summaryErr error
						row.RunSummary, summaryErr = SessionSummary(row.Session)
						if runErr == nil {
							runErr = summaryErr
						}
					}
				}
			} else {
				res, err := runSuite(ctx, d, target.request)
				row.Session, row.RunSummary = res.SessionRoot, res.Summary
				row.Stopped, row.FailedAt, row.SetupSteps = res.Stopped, res.FailedAt, res.SetupSteps
				row.CleanupFailed = res.CleanupFailed
				cleanupFailed = res.CleanupFailed
				if cleanupFailed && err == nil {
					err = fmt.Errorf("owned network cleanup failed")
				}
				runErr = err
			}
			if runErr != nil {
				row.Error = runErr.Error()
			}
		}
		if row.Error != "" {
			out.SetupErrors++
		}
		out.Summary.Pass += row.Summary.Pass
		out.Summary.Fail += row.Summary.Fail
		out.Summary.Blocked += row.Summary.Blocked
		out.Summary.Skip += row.Summary.Skip
		out.Runs = append(out.Runs, row)
	}
	return out, nil
}

func prepareMatrix(ctx context.Context, in RunMatrixIn) ([]preparedMatrixTarget, error) {
	if len(in.Presets) == 0 {
		return nil, fmt.Errorf("matrix needs at least one chain preset")
	}
	if in.Env != "" {
		return nil, fmt.Errorf("matrix presets cannot combine with a single chain preset override")
	}
	raws := in.SpecContent
	baseDir := in.BaseDir
	if len(raws) == 0 {
		if len(in.SpecPaths) == 0 {
			return nil, fmt.Errorf("matrix needs test definitions")
		}
		if baseDir == "" {
			baseDir = filepath.Dir(in.SpecPaths[0])
		}
		for _, path := range in.SpecPaths {
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			raws = append(raws, raw)
		}
	}
	targets := make([]preparedMatrixTarget, 0, len(in.Presets))
	for index, ref := range in.Presets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ref == "" {
			return nil, fmt.Errorf("matrix preset %d is empty", index+1)
		}
		preset, specs, err := dsl.CasesOnPreset(raws, baseDir, ref)
		if err != nil {
			return nil, fmt.Errorf("matrix preset %d (%s): %w", index+1, ref, err)
		}
		parsed := make([]dsl.Spec, 0, len(specs))
		for _, raw := range specs {
			spec, err := dsl.Parse(raw)
			if err != nil {
				return nil, err
			}
			parsed = append(parsed, spec)
		}
		if err := testengine.Precheck(parsed); err != nil {
			return nil, fmt.Errorf("matrix preset %d: %w", index+1, err)
		}
		if in.Chain != "" && in.Chain != parsed[0].Chain.Name {
			return nil, fmt.Errorf("matrix preset %d declares %s, not %s", index+1, parsed[0].Chain.Name, in.Chain)
		}
		declared, err := declaredAttachIn(specs, nil)
		if err != nil {
			return nil, err
		}
		one := in.RunSuiteIn
		one.SpecPaths, one.SpecContent, one.Env = nil, specs, ""
		if one.ArtifactRoot == "" {
			if one.DataDir != "" {
				one.ArtifactRoot = filepath.Join(one.DataDir, "sessions")
			} else {
				one.ArtifactRoot = filepath.Join(".chainbench", "matrix")
			}
		}
		row := PresetRunResult{Index: index + 1, PresetRef: ref, PresetID: preset.ID, Chain: parsed[0].Chain.Name}
		if declared == nil {
			if in.NoNodeControl {
				return nil, fmt.Errorf("matrix preset %d composes a network; noNodeControl is an attach option", index+1)
			}
			if one.DataDir == "" {
				return nil, fmt.Errorf("matrix preset %d needs a workspace directory to compose", index+1)
			}
			one.DataDir = filepath.Join(one.DataDir, "matrix", fmt.Sprintf("preset-%03d", index+1))
			layout, plan, err := PlanSuiteLayout(ctx, one)
			if err != nil {
				return nil, fmt.Errorf("matrix preset %d: %w", index+1, err)
			}
			row.Workspace = one.DataDir
			if in.Plan {
				row.Plan, row.Layout = &plan, &layout
			}
		} else if in.Plan {
			return nil, fmt.Errorf("matrix plan is only available for composition (preset %d attaches)", index+1)
		}
		targets = append(targets, preparedMatrixTarget{result: row, request: one, declared: declared})
	}
	return targets, nil
}
