package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestMCPPlansPersonalDelegationCasesWithRawTamperInput(t *testing.T) {
	for _, name := range []string{"01b-personal-normal.json", "02b-personal-sender.json", "03b-personal-feepayer.json", "04b-personal-insufficient.json"} {
		t.Run(name, func(t *testing.T) {
			raw, err := dsl.ReadFiles([]string{filepath.Join("../../tests/tc/go-stablenet/regression/fee-delegation", name)})
			if err != nil {
				t.Fatal(err)
			}
			workspace := t.TempDir()
			args := map[string]any{"plan": true, "suite": true, "dataDir": workspace, "spec": string(raw[0]), "binaryFor": map[string]any{"default": "/accepted/gstable"}}
			out, err := runTool().Handler(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			var plans []app.SuitePlan
			if err = json.Unmarshal([]byte(out), &plans); err != nil || len(plans) != 1 || plans[0].Layout.Binary != "/accepted/gstable" || plans[0].Layout.KeysSource != "generate" || plans[0].Layout.KeysDir != filepath.Join(workspace, "keys") {
				t.Fatal("public case did not reach the shared planner", err, out)
			}
		})
	}
}
