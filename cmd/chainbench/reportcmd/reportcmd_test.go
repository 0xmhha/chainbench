package reportcmd_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/reportcmd"
)

// report and log read what a run left behind. Both refuse without being told
// where to look, which is the behaviour these pin — until U1 moved them out of
// package main there was no way to check it.

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(reportcmd.NewReport(), reportcmd.NewLog())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestReport_RefusesWithoutAWorkspace: with no directory there is no run to
// report on, and an empty report would read as a run that produced nothing.
func TestReport_RefusesWithoutAWorkspace(t *testing.T) {
	out, err := run(t, "report")
	if err == nil {
		t.Fatalf("report with no workspace was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--workspace-dir is required") {
		t.Errorf("the error does not name the missing flag: %v", err)
	}
}

// TestLog_RefusesWithoutADirectory: same reason, and the same distinction
// between "nothing matched" and "nowhere was searched".
func TestLog_RefusesWithoutADirectory(t *testing.T) {
	out, err := run(t, "log")
	if err == nil {
		t.Fatalf("log with no directory was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
}

// TestReport_OnAnEmptyDirectorySaysSo: a directory with no run in it is a fact,
// not a crash.
func TestReport_OnAnEmptyDirectorySaysSo(t *testing.T) {
	out, err := run(t, "report", "--workspace-dir", t.TempDir())
	if err == nil && strings.TrimSpace(out) == "" {
		t.Fatal("an empty workspace produced neither an error nor a word of output")
	}
}
