package app

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/resource"
)

// WithWorkspaceConfig folds a --workspace-config file's data root into a compose
// target, so the step-form surfaces (CLI and MCP) resolve the target the one way
// the run path does. The workspace-config is the single owner of the data root:
// a target that already names a different one is a conflict, not a silent
// override, and an empty config path leaves the target unchanged. The target's
// locality (local vs remote/SSH) still comes from the server set — only the data
// root comes from this file.
func WithWorkspaceConfig(target resource.Spec, wcPath string) (resource.Spec, error) {
	if wcPath == "" {
		return target, nil
	}
	wc, err := resource.LoadWorkspaceConfig(wcPath)
	if err != nil {
		return target, err
	}
	if target.DataRoot != "" && target.DataRoot != wc.DataRoot {
		return target, fmt.Errorf(
			"data root conflict: the target says %q but --workspace-config says %q — put the data root in workspace-config alone",
			target.DataRoot, wc.DataRoot)
	}
	target.DataRoot = wc.DataRoot
	return target, nil
}

// TargetForWorkspaceConfig returns a fresh compose target rooted at the
// workspace-config's data root, for a surface that has no target of its own to
// fold into (the MCP step-form up). An empty path yields the zero target, which
// the composition defaults to a local root.
func TargetForWorkspaceConfig(wcPath string) (resource.Spec, error) {
	return WithWorkspaceConfig(resource.Spec{}, wcPath)
}
