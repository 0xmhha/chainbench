package app

import (
	"context"
	"strings"
	"testing"
)

// The job screen builds its operation choices from this contract, so the list
// must be exactly the operations the chain engine accepts.
func TestJobsContractListsTheOperationsTheEngineAccepts(t *testing.T) {
	c := JobsContract()
	ids := map[string]bool{}
	for _, op := range c.Operations {
		if op.Label == "" || (op.Scope != "chain" && op.Scope != "test") {
			t.Errorf("operation %+v", op)
		}
		ids[op.ID] = true
		if !webKnownOperation(op.ID) {
			t.Errorf("%s listed but not accepted", op.ID)
		}
	}
	for _, want := range []string{"chain.setup", "chain.deploy", "node.start", "node.stop", "node.restart", "node.swap", "node.reset", "network.monitor", "test.run", "test.attach"} {
		if !ids[want] {
			t.Errorf("%s missing", want)
		}
	}
	if len(ids) != 10 || webKnownOperation("chain.destroy") {
		t.Fatalf("operations = %v", ids)
	}
	e := &WebChainEngine{}
	if _, err := e.Prepare(context.Background(), DeploymentActor{ID: "a", Role: "operator"}, WebPlanInput{Operation: "chain.destroy"}); err == nil || !strings.Contains(err.Error(), "operation") {
		t.Fatalf("unknown operation: %v", err)
	}
}

// The chain contract titles each node count and the sync field for the preset screen.
func TestChainPresetContractTitlesCountsAndSync(t *testing.T) {
	c, err := ChainPresetContract("stablenet")
	if err != nil {
		t.Fatal(err)
	}
	env := c["$defs"].(map[string]any)["envSpec"].(map[string]any)["properties"].(map[string]any)
	counts := env["topology"].(map[string]any)["oneOf"].([]any)[0].(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"bp", "en", "pn", "syncMode"} {
		if title, _ := counts[field].(map[string]any)["title"].(string); title == "" {
			t.Errorf("%s has no title", field)
		}
	}
}
