package testengine_test

import (
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestPersonalCasesGenerateWorkspaceLocalKeystores(t *testing.T) {
	for _, name := range []string{"01b-personal-normal.json", "02b-personal-sender.json", "03b-personal-feepayer.json", "04b-personal-insufficient.json"} {
		t.Run(name, func(t *testing.T) {
			raw, err := dsl.ReadFiles([]string{filepath.Join(repoRoot(t), "tests/tc/go-stablenet/regression/fee-delegation", name)})
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(t.TempDir(), "w")
			up, plan, err := testengine.PlanSuiteLayout(t.Context(), testengine.RunSuiteIn{SpecContent: raw, DataDir: workspace})
			if err != nil {
				t.Fatal(err)
			}
			if plan.Keys.Source != "generate" || up.KeysSource != "generate" || plan.Keys.Dir != filepath.Join(workspace, "keys") || up.KeysDir != plan.Keys.Dir {
				t.Fatalf("personal import would use a shared keystore: %+v", plan.Keys)
			}
		})
	}
}
