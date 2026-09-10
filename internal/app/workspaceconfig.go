package app

import (
	"github.com/0xmhha/chainbench/internal/resource"
)

// WithWorkspaceConfig folds a --workspace-config file's data root into a compose
// target, so the step-form surfaces (CLI and MCP) resolve the target the one way
// the run path does. An empty config path leaves the target unchanged. The rule
// for what folding means — and what a target that names a different root does —
// belongs to the config itself, in AdoptDataRoot.
func WithWorkspaceConfig(target resource.Spec, wcPath string) (resource.Spec, error) {
	if wcPath == "" {
		return target, nil
	}
	wc, err := resource.LoadWorkspaceConfig(wcPath)
	if err != nil {
		return target, err
	}
	return wc.AdoptDataRoot(target, "")
}

// TargetForWorkspaceConfig returns a fresh compose target rooted at the
// workspace-config's data root, for a surface that has no target of its own to
// fold into (the MCP step-form up). An empty path yields the zero target, which
// the composition defaults to a local root.
func TargetForWorkspaceConfig(wcPath string) (resource.Spec, error) {
	return WithWorkspaceConfig(resource.Spec{}, wcPath)
}
