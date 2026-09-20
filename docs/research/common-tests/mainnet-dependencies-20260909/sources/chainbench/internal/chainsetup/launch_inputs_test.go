package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

// A composed network launches from files on its target: the genesis and one
// config per node. Deploy used to check only that they were there, and reported
// them "reused, not rewritten" — which is equally true of a genesis someone
// edited and of a config left behind by a previous composition. The nodes then
// launch from whatever is present.
//
// These compose far enough to write the launch inputs, then change one on disk
// and re-run deploy.

// composed runs the steps that produce the launch inputs and returns the
// workspace directory.
func composedForInputs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{
		Chain: "stablenet", KeysDir: filepath.Join("..", "..", "keys", "preset"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Validators: 2, Endpoints: 0}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if _, err := ws.Config(ctx); err != nil {
		t.Fatalf("config: %v", err)
	}
	if _, err := ws.Provision(ctx); err != nil {
		t.Fatalf("deploy on a fresh composition should pass: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	return dir
}

// reprovision re-runs deploy on an existing workspace.
func reprovision(t *testing.T, dir string) (string, error) {
	t.Helper()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	return ws.Provision(context.Background())
}

// TestDeploy_RefusesAGenesisSomethingElseWrote: a genesis at the expected path
// whose content is not the one this workspace built must not be launched from.
func TestDeploy_RefusesAGenesisSomethingElseWrote(t *testing.T) {
	dir := composedForInputs(t)
	genesis := filepath.Join(dir, "genesis.json")

	before, err := os.ReadFile(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(genesis, append(before, ' '), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := reprovision(t, dir)
	if err == nil {
		t.Fatalf("deploy accepted an edited genesis and called it reused:\n%s", out)
	}
	if !strings.Contains(err.Error(), "genesis.json") {
		t.Errorf("the refusal does not name the file: %v", err)
	}
	if !strings.Contains(err.Error(), "not the file this workspace built") {
		t.Errorf("the refusal does not say what is wrong: %v", err)
	}
}

// TestDeploy_RefusesAConfigSomethingElseWrote: the same for a node's config,
// which is where a stale composition's settings would otherwise survive.
func TestDeploy_RefusesAConfigSomethingElseWrote(t *testing.T) {
	dir := composedForInputs(t)
	cfg := filepath.Join(dir, "config_node1.toml")
	if _, err := os.Stat(cfg); err != nil {
		t.Skipf("this composition names its config differently: %v", err)
	}
	if err := os.WriteFile(cfg, []byte("# something else wrote this\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := reprovision(t, dir)
	if err == nil {
		t.Fatalf("deploy accepted a replaced config and called it reused:\n%s", out)
	}
	if !strings.Contains(err.Error(), "config_node1.toml") {
		t.Errorf("the refusal does not name the file: %v", err)
	}
}

// TestDeploy_AcceptsWhatItBuilt: re-running deploy on an untouched composition
// stays a no-op. A check that refused everything would be no better than one
// that accepted everything.
func TestDeploy_AcceptsWhatItBuilt(t *testing.T) {
	dir := composedForInputs(t)
	out, err := reprovision(t, dir)
	if err != nil {
		t.Fatalf("deploy refused the files it wrote: %v", err)
	}
	if !strings.Contains(out, "present on the target") {
		t.Errorf("deploy did not report the inputs as present: %s", out)
	}
}
