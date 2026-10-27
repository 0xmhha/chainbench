package app

import (
	"context"
	"encoding/json"
	"reflect"
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

// The contract lists, for each operation, exactly the arguments the engine
// accepts, each with a title and what it becomes on the command line.
func TestJobsContractNamesEveryArgumentTheEngineReads(t *testing.T) {
	fields := map[string]bool{}
	for _, f := range reflect.VisibleFields(reflect.TypeFor[webChainArguments]()) {
		fields[strings.Split(f.Tag.Get("json"), ",")[0]] = true
	}
	listed := map[string]bool{}
	for _, op := range JobsContract().Operations {
		if op.Command == "" || len(op.Arguments) == 0 {
			t.Errorf("%s names no command or arguments", op.ID)
		}
		for _, a := range op.Arguments {
			listed[a.Name] = true
			if !fields[a.Name] || a.Title == "" || a.Type == "" || a.CLI == "" {
				t.Errorf("%s argument %+v", op.ID, a)
			}
		}
	}
	for name := range fields {
		if !listed[name] {
			t.Errorf("the engine reads %s but no operation lists it", name)
		}
	}
}

// A plan naming an argument its operation does not read is refused, so an
// ignored value never looks as if it took effect.
func TestPlanRefusesAnArgumentItsOperationDoesNotRead(t *testing.T) {
	e := &WebChainEngine{}
	for op, args := range map[string]string{
		"test.attach":     `{"binaryAssets":{"x":"y"}}`,
		"chain.deploy":    `{"noSkips":true}`,
		"node.stop":       `{"chainPresetRef":{"id":"p","revision":1}}`,
		"network.monitor": `{"assetId":"a"}`,
	} {
		_, err := e.Prepare(context.Background(), DeploymentActor{ID: "a", Role: "operator"}, WebPlanInput{Operation: op, Arguments: json.RawMessage(args)})
		if err == nil || !strings.Contains(err.Error(), "does not read") {
			t.Errorf("%s %s: %v", op, args, err)
		}
	}
	for _, args := range []string{`{"waitBlocks":-1}`, `{"waitBlocks":1000001}`, `{"nodeMonitorTimeoutSeconds":3601}`} {
		_, err := e.Prepare(context.Background(), DeploymentActor{ID: "a", Role: "operator"}, WebPlanInput{Operation: "test.run", Arguments: json.RawMessage(args)})
		if err == nil || !strings.Contains(err.Error(), "between") {
			t.Errorf("test.run %s: %v", args, err)
		}
	}
}
