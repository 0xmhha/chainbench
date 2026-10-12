package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
)

// planningCLI resolves the real engine projection in unit fixtures only.
// Production adapters require a configured CLI and never fall back in process.
type planningCLI struct{}

func (planningCLI) ChainStatus(context.Context, string) (State, error) { return State{}, nil }
func (planningCLI) Run(ctx context.Context, args ...string) ([]byte, error) {
	if !slices.Contains(args, "--plan") {
		return nil, fmt.Errorf("unexpected planning fixture command: %v", args)
	}
	in := RunSuiteIn{DataDir: flagValue(args, "--workspace-dir"), Chain: flagValue(args, "--chain"), Binary: flagValue(args, "--binary"), KeysDir: flagValue(args, "--keys"), WorkspaceConfigPath: flagValue(args, "--workspace-config")}
	in.Server.SetPath, in.Server.Name = flagValue(args, "--server-set"), flagValue(args, "--server")
	if args[0] == "chain" {
		raw, err := os.ReadFile(flagValue(args, "--chain-preset"))
		if err != nil {
			return nil, err
		}
		out, err := PlanChainPreset(ctx, raw, ChainUpIn{DataDir: in.DataDir, Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir, WorkspaceConfigPath: in.WorkspaceConfigPath, Server: in.Server, Stage: UpStart})
		if err != nil {
			return nil, err
		}
		return json.Marshal(out)
	}
	for _, a := range args[1:] {
		if len(a) > 1 && a[:2] == "--" {
			break
		}
		in.SpecPaths = append(in.SpecPaths, a)
	}
	in.ReadOnlyKeys = slices.Contains(args, "--read-only-keys")
	in.BinaryOverrides = map[string]string{}
	for i, a := range args {
		if a == "--binary-for" && i+1 < len(args) {
			pair := args[i+1]
			for j, c := range pair {
				if c == '=' {
					in.BinaryOverrides[pair[:j]] = pair[j+1:]
					break
				}
			}
		}
	}
	layout, plan, err := PlanSuiteLayout(ctx, in)
	if err != nil {
		return nil, err
	}
	return json.Marshal([]SuitePlan{{Spec: "suite", Layout: layout, Plan: plan}})
}

func TestWebPlansUseCLIAndPropagateItsFailure(t *testing.T) {
	stop := errors.New("CLI planner rejected the reviewed input")
	for _, preset := range []bool{false, true} {
		t.Run(map[bool]string{false: "suite", true: "preset"}[preset], func(t *testing.T) {
			var e *WebChainEngine
			var p webChainPayload
			if preset {
				e, p, _ = webPresetFixture(t, `{"bp":4}`)
			} else {
				e, p = webRunPlanFixture(t, `{"bp":4}`)
			}
			cli := &runCLI{err: stop}
			e.UseCLI(cli)
			var err error
			if preset {
				_, err = e.preparePresetComposition(context.Background(), &p)
			} else {
				_, err = e.prepareTestRun(context.Background(), &p)
			}
			if !errors.Is(err, stop) {
				t.Fatalf("planner bypassed CLI: %v", err)
			}
			if !slices.Contains(cli.args, "--plan") || !slices.Contains(cli.args, "--json") {
				t.Fatalf("not a read-only JSON plan: %v", cli.args)
			}
		})
	}
}
