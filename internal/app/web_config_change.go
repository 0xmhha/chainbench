package app

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// webConfigChanges accepts only the engine's generated-config choices and
// refuses an explicit launch layer that would override the reviewed value.
func webConfigChanges(state State, p webChainPayload) ([]string, error) {
	if err := webSelectedNode(state, p.Input.NodeIDs); err != nil {
		return nil, err
	}
	var selected node.Record
	for _, n := range state.Nodes {
		if string(n.NodeLabel()) == p.Input.NodeIDs[0] {
			selected = n
		}
	}
	if selected.Config != "" || selected.Binary != "" {
		return nil, ErrDeploymentConflict
	}
	schema := nodeconfig.ConfigOptionSchemas()
	patch := p.Arguments.ConfigOverrides
	if len(patch) == 0 || len(patch) > len(schema) {
		return nil, errors.New("select at least one supported configuration change")
	}
	launch := slices.Clone(state.LaunchCommand)
	for _, scope := range node.ScopeFor(node.Role(selected.Role), selected.Index) {
		launch = append(launch, state.LaunchSet[scope]...)
	}
	changes := []string{}
	for key, value := range patch {
		raw, ok := schema[key]
		if !ok {
			return nil, errors.New("unsupported configuration field")
		}
		field := raw.(map[string]any)
		if !slices.Contains(field["enum"].([]string), value) {
			return nil, errors.New("unsupported configuration choice")
		}
		for _, entry := range launch {
			option, _, _ := strings.Cut(entry, "=")
			if slices.Contains(field["x-launch-options"].([]string), option) {
				return nil, errors.New("an explicit launch option overrides this configuration field")
			}
		}
		changes = append(changes, key+"="+value)
	}
	slices.Sort(changes)
	return changes, nil
}

// bindWebConfigKeys verifies the existing encrypted snapshot and immutable
// material that config rendering reads. Source key edits cannot replace it.
func (e *WebChainEngine) bindWebConfigKeys(ctx context.Context, state State) (webKeySnapshot, error) {
	snapshot := webKeySnapshot{SHA256: filepath.Base(state.KeysDir)}
	if len(snapshot.SHA256) != 64 || state.KeysDir != webAcceptedKeyPath(e.root, snapshot.SHA256) {
		return webKeySnapshot{}, ErrDeploymentConflict
	}
	files, err := session.OpenKeySnapshotFiles(e.root)
	if err != nil {
		return webKeySnapshot{}, err
	}
	encrypted, err := files.Read(snapshot.SHA256)
	if err != nil {
		return webKeySnapshot{}, ErrDeploymentConflict
	}
	if _, err = e.decryptKeys(snapshot, encrypted); err != nil {
		return webKeySnapshot{}, ErrDeploymentConflict
	}
	raw, err := session.CaptureKeys(ctx, state.KeysDir)
	if err != nil || manifestHash(raw) != snapshot.SHA256 {
		return webKeySnapshot{}, ErrDeploymentConflict
	}
	return snapshot, nil
}
