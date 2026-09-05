package suitecmd_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/exitcode"
	"github.com/0xmhha/chainbench/cmd/chainbench/suitecmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// run, validate and migrate-spec are the suite verbs. The exit code they choose
// is part of their contract with CI, and until U1 moved them out of package
// main the mounted commands could not be exercised from a test.

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(suitecmd.NewRun(), suitecmd.NewValidate(), suitecmd.NewMigrateSpec())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestValidate_AnInvalidSpecExitsOne: CI reads the exit code, so an invalid
// spec has to be told apart from a clean run by more than the words printed.
func TestValidate_AnInvalidSpecExitsOne(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(spec, []byte(`{"name":"x","steps":[{"nosuchaction":true}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "validate", spec)
	if got := exitcode.Of(err); got != 1 {
		t.Fatalf("an invalid spec exited %d, want 1\n%s", got, out)
	}
}

// TestValidate_AMissingFileExitsOne: a path that is not there is the operator's
// mistake, and it must not pass as "nothing to validate".
func TestValidate_AMissingFileExitsOne(t *testing.T) {
	out, err := run(t, "validate", filepath.Join(t.TempDir(), "no-such-spec.json"))
	if got := exitcode.Of(err); got != 1 {
		t.Fatalf("a missing spec exited %d, want 1\n%s", got, out)
	}
}

// TestMigrateSpec_RefusesWithoutAFile: the command rewrites a spec, so it needs
// one named rather than reading whatever is at hand.
func TestMigrateSpec_RefusesWithoutAFile(t *testing.T) {
	out, err := run(t, "migrate-spec")
	if err == nil {
		t.Fatalf("migrate-spec with no file was accepted:\n%s", out)
	}
}

// TestMigrateSpec_RefusesASpecThatIsAlreadyV2: converting a v2 spec again would
// either be a no-op presented as work or a corruption; either way the command
// should say the file is already in the new grammar.
func TestMigrateSpec_RefusesASpecThatIsAlreadyV2(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "v2.json")
	if err := os.WriteFile(spec, []byte(`{"version":2,"name":"x","cases":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "migrate-spec", spec)
	if err == nil && !strings.Contains(strings.ToLower(out), "v2") {
		t.Fatalf("a v2 spec was migrated again without a word about it:\n%s", out)
	}
}
