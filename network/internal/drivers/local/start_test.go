package local

import (
	"context"
	"strings"
	"testing"
)

func TestStartNode_CallsCorrectArgs(t *testing.T) {
	var got []string
	d := NewDriverWithExec("/opt/chainbench", recordingExecFn(&got))
	res, err := d.StartNode(context.Background(), "3")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit: got %d", res.ExitCode)
	}
	want := []string{"/opt/chainbench/chainbench.sh", "node", "start", "3"}
	if len(got) != len(want) {
		t.Fatalf("argc: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
	if !strings.Contains(res.Stdout, "stopped") {
		// recordingExecFn in stop_test.go echoes "stopped" — fine for this test
		// since we only care about args.
		t.Logf("stdout (informational): %q", res.Stdout)
	}
}

func TestStartNode_EmptyNodeNum_StillExecs(t *testing.T) {
	var got []string
	d := NewDriverWithExec("/x", recordingExecFn(&got))
	_, err := d.StartNode(context.Background(), "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(got) != 4 || got[3] != "" {
		t.Errorf("expected empty 4th arg, got %v", got)
	}
}
