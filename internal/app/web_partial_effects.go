package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/resource"
)

// webTargetPhases change the target machines; the phases before them only
// write the control directory on this server.
var webTargetPhases = map[string]bool{"deploy": true, "init": true, "start": true}

// webFailureDisposition is what a failed composition left behind. A failure
// before any target phase leaves the nodes untouched; one during or after a
// target phase may have left some, so the outcome is unknown until checked.
func webFailureDisposition(phase string) string {
	if webTargetPhases[phase] {
		return "unknown"
	}
	return "retained"
}

// webRecordedNodeEffects reports each recorded node's known state: whether its
// data directory exists on its machine and whether a launch was recorded. A
// node that exists or could not be checked stays unresolved.
func webRecordedNodeEffects(state State, exists func(path string) (bool, error)) (effects, unresolved []string) {
	for _, ns := range state.Nodes {
		label := string(ns.NodeLabel())
		present, err := exists(ns.DataDir)
		switch {
		case err != nil:
			effects = append(effects, label+": state could not be checked")
			unresolved = append(unresolved, ns.DataDir)
		case !present:
			effects = append(effects, label+": no data directory")
		case ns.PID > 0:
			effects = append(effects, fmt.Sprintf("%s: data directory present, launched with recorded PID %d", label, ns.PID))
			unresolved = append(unresolved, ns.DataDir)
		default:
			effects = append(effects, label+": data directory present, not launched")
			unresolved = append(unresolved, ns.DataDir)
		}
	}
	return effects, unresolved
}

// nodeEffects checks every node the record names on its own machine after a
// failed target phase. Without a record, nothing per node is known.
func (e *WebChainEngine) nodeEffects(ctx context.Context, p webChainPayload, lookup resource.Lookup) ([]string, []string) {
	raw, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
	if err != nil {
		return []string{"no node record was written, so no node is known"}, []string{p.Target.DataPath}
	}
	var state State
	if err = json.Unmarshal(raw, &state); err != nil {
		return []string{"the node record could not be read"}, []string{p.Target.DataPath}
	}
	opener := resource.Opener{Lookup: lookup}
	return webRecordedNodeEffects(state, func(path string) (bool, error) {
		for _, ns := range state.Nodes {
			if ns.DataDir != path {
				continue
			}
			access, err := opener.Open(webNodeTarget(state, ns))
			if err != nil {
				return false, err
			}
			return access.Files.Exists(ctx, path)
		}
		return false, fmt.Errorf("unrecorded path")
	})
}
