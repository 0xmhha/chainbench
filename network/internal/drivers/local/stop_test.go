package local

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// recordingExecFn returns a factory that stores the args it received
// for later assertion.
func recordingExecFn(capturedArgs *[]string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Record full argv: name (chainbench.sh path) then args.
		full := append([]string{name}, args...)
		*capturedArgs = append([]string(nil), full...)
		// Still need a runnable command — echo the args to stdout, exit 0.
		return exec.CommandContext(ctx, "sh", "-c", `echo "stopped"`)
	}
}

func TestStopNode_CallsCorrectArgs(t *testing.T) {
	var got []string
	d := NewDriverWithExec("/opt/chainbench", recordingExecFn(&got))
	res, err := d.StopNode(context.Background(), "3")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit: got %d", res.ExitCode)
	}
	want := []string{"/opt/chainbench/chainbench.sh", "node", "stop", "3"}
	if len(got) != len(want) {
		t.Fatalf("argc: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
	if !strings.Contains(res.Stdout, "stopped") {
		t.Errorf("stdout: got %q", res.Stdout)
	}
}

func TestStopNode_EmptyNodeNum_StillExecs(t *testing.T) {
	// StopNode does not validate input — that's the handler's job. Empty
	// string is passed through to the subprocess.
	var got []string
	d := NewDriverWithExec("/x", recordingExecFn(&got))
	_, err := d.StopNode(context.Background(), "")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(got) != 4 || got[3] != "" {
		t.Errorf("expected empty 4th arg, got %v", got)
	}
}
