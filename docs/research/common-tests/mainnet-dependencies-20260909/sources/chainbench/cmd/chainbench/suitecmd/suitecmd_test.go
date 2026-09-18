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

// TestRun_AttachNeedsAWorkspace: --attach takes its endpoints from a
// workspace, so without one there is nothing to attach to.
func TestRun_AttachNeedsAWorkspace(t *testing.T) {
	out, err := run(t, "run", "--attach", "spec.json")
	if err == nil {
		t.Fatalf("--attach with no workspace was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--workspace-dir") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
}

// TestRun_AttachAndRPCAreDifferentAnswersToTheSameQuestion: both name the
// network to run against, so giving both leaves it ambiguous which one meant
// it. Refusing beats picking one.
func TestRun_AttachAndRPCAreDifferentAnswersToTheSameQuestion(t *testing.T) {
	out, err := run(t, "run", "--attach", "--workspace-dir", t.TempDir(),
		"--rpc", "http://127.0.0.1:1", "spec.json")
	if err == nil {
		t.Fatalf("--attach with --rpc was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "does not combine with --rpc") {
		t.Errorf("the error does not explain the conflict: %v", err)
	}
}

// TestRun_ComposingAndAttachingAreStillToldApart: --workspace-dir alone still
// composes, and the refusal that says so now points at --attach, which is what
// the operator wanted if their network is already up.
func TestRun_ComposingAndAttachingAreStillToldApart(t *testing.T) {
	out, err := run(t, "run", "--workspace-dir", t.TempDir(), "--rpc", "http://127.0.0.1:1", "spec.json")
	if err == nil {
		t.Fatalf("--workspace-dir with --rpc was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--attach") {
		t.Errorf("the refusal does not mention the mode that does what was asked: %v", err)
	}
}

// TestRun_AttachRefusesAWorkspaceWithNoNetwork: attaching to a directory that
// composed nothing has to say so, rather than run against an empty endpoint
// list and report that every spec passed.
func TestRun_AttachRefusesAWorkspaceWithNoNetwork(t *testing.T) {
	out, err := run(t, "run", "--workspace-dir", t.TempDir(), "--attach", "spec.json")
	if err == nil {
		t.Fatalf("attaching to an empty workspace was accepted:\n%s", out)
	}
}
