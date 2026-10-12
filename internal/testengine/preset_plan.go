package testengine

import (
	"context"
	"encoding/json"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// PlanChainPreset uses the suite's composition lowering without running a test.
// It may materialize content-addressed overlays beneath in.DataDir; callers
// choose a private planning directory rather than a retained network directory.
func PlanChainPreset(ctx context.Context, raw []byte, in RunSuiteIn) (chainsetup.ChainUpIn, ComposePlan, error) {
	if err := ctx.Err(); err != nil {
		return chainsetup.ChainUpIn{}, ComposePlan{}, err
	}
	if err := ValidateChainPreset(raw); err != nil {
		return chainsetup.ChainUpIn{}, ComposePlan{}, err
	}
	// The existing validator uses this same grammar projection. The statement
	// is never executed; only the environment reaches compositionOf.
	content, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "preset-plan", "chainPreset": json.RawMessage(raw), "steps": []map[string]any{{"expect": "blockNumber", "is": 0}}})
	if err != nil {
		return chainsetup.ChainUpIn{}, ComposePlan{}, err
	}
	spec, err := dsl.Parse(content)
	if err != nil {
		return chainsetup.ChainUpIn{}, ComposePlan{}, err
	}
	c, err := compositionOf(ctx, spec, in)
	if err != nil {
		return chainsetup.ChainUpIn{}, ComposePlan{}, err
	}
	return *c.up, planOf(c, spec.Chain.Name), nil
}
