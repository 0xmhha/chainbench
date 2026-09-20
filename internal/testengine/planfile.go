// Keeping the plan after the run that made it.
//
// The plan answers "why is this value this value", and until now it answered it
// once, on stderr, while the run was starting. A week later the workspace holds
// the network and the record of what it was asked to compose, but not who asked
// for each value — so the question M4 exists to answer had to be re-derived by
// merging two files in someone's head, which is the thing it set out to remove.
//
// So the plan is written beside the record. It is the plan of the run that last
// composed here, not a history: the record's step marks carry the times, and a
// second copy of that ordering would be a second thing to keep right.

package testengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PlanFile is the plan's name inside a workspace, next to chain-record.json.
const PlanFile = "compose-plan.json"

// WritePlan saves the plan into the workspace directory.
//
// A plan that cannot be written does not stop a run: it is a record for later,
// and refusing to test a healthy network because a display file could not be
// saved trades a real answer for a bookkeeping one. The caller reports it.
func WritePlan(dir string, p ComposePlan) error {
	if dir == "" {
		return fmt.Errorf("testengine: write plan: no workspace directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("testengine: write plan: %w", err)
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("testengine: write plan: %w", err)
	}
	path := filepath.Join(dir, PlanFile)
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("testengine: write plan: %w", err)
	}
	return nil
}

// ReadPlan loads the plan a workspace last composed from.
//
// It is how an operator asks, after the fact, who chose a value — and how a
// later run can see that the workspace in front of it was composed from a
// different declaration than the one it holds.
func ReadPlan(dir string) (ComposePlan, error) {
	raw, err := os.ReadFile(filepath.Join(dir, PlanFile))
	if err != nil {
		return ComposePlan{}, fmt.Errorf("testengine: read plan: %w", err)
	}
	var p ComposePlan
	if err := json.Unmarshal(raw, &p); err != nil {
		return ComposePlan{}, fmt.Errorf("testengine: read plan: %w", err)
	}
	return p, nil
}
