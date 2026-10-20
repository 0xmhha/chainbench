package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

func composedMatrixInput(t *testing.T) RunMatrixIn {
	t.Helper()
	dir := t.TempDir()
	presets := []string{}
	for _, id := range []string{"first", "second"} {
		raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": id, "chain": "stablenet", "topology": map[string]any{"bp": 4}, "binaries": map[string]any{"default": "gstable"}})
		path := filepath.Join(dir, id+".json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		presets = append(presets, path)
	}
	keys, err := filepath.Abs("../../presets/keys")
	if err != nil {
		t.Fatal(err)
	}
	return RunMatrixIn{RunSuiteIn: RunSuiteIn{DataDir: filepath.Join(dir, "workspace"), KeysDir: keys, SpecContent: [][]byte{[]byte(`{"schemaVersion":"2","kind":"case","id":"shared","chainPreset":"base","steps":[{"expect":"blockNumber","is":0}]}`)}}, Presets: presets}
}

func TestMatrixDoesNotStartAnotherNetworkAfterCleanupFailure(t *testing.T) {
	in := composedMatrixInput(t)
	calls := 0
	out, err := runMatrix(t.Context(), Deps{}, in, func(context.Context, Deps, RunSuiteIn) (RunSuiteOut, error) {
		calls++
		return RunSuiteOut{CleanupFailed: true}, errors.New("owned node could not be stopped")
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(out.Runs) != 2 || out.Runs[1].Attempted || out.Runs[1].Error == "" || out.ExitCode(false) != 2 {
		t.Fatalf("cleanup failure launched another network: calls=%d out=%+v", calls, out)
	}
}

func TestMatrixCancellationPreservesTheCompletedTargetAndDoesNotAttemptTheNext(t *testing.T) {
	in := composedMatrixInput(t)
	calls := 0
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out, err := runMatrix(ctx, Deps{}, in, func(context.Context, Deps, RunSuiteIn) (RunSuiteOut, error) {
		calls++
		cancel()
		return RunSuiteOut{Stopped: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(out.Runs) != 2 || !out.Runs[0].Attempted || out.Runs[1].Attempted || out.Runs[1].Error == "" || out.ExitCode(false) != 2 {
		t.Fatalf("cancellation lost target boundaries: calls=%d out=%+v", calls, out)
	}
}

func TestMatrixPlanningDoesNotExecuteTargetsOrCreateTheirWorkspaces(t *testing.T) {
	in := composedMatrixInput(t)
	in.Plan = true
	calls := 0
	out, err := runMatrix(t.Context(), Deps{}, in, func(context.Context, Deps, RunSuiteIn) (RunSuiteOut, error) { calls++; return RunSuiteOut{}, nil })
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || len(out.Runs) != 2 || out.ExitCode(true) != 0 {
		t.Fatalf("plan executed targets: %+v calls=%d", out, calls)
	}
	for _, row := range out.Runs {
		if row.Attempted || row.Session != "" || row.Plan == nil || row.Layout == nil || row.PresetID == "" {
			t.Fatalf("plan lost target or invented execution: %+v", row)
		}
		if _, err := os.Stat(row.Workspace); !os.IsNotExist(err) {
			t.Fatal("planning created a workspace", row.Workspace, err)
		}
	}
}
