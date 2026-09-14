package inspector_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/inspector"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// TestOnPath_LocalAsksThisProcessesPath: with no runner the target is this
// machine, so the answer is this process's PATH — the same one exec will use.
func TestOnPath_LocalAsksThisProcessesPath(t *testing.T) {
	path, ok, err := inspector.OnPath(context.Background(), nil, "sh")
	if err != nil {
		t.Fatalf("OnPath(sh): %v", err)
	}
	if !ok || path == "" {
		t.Fatalf("sh must be on PATH: %q, %v", path, ok)
	}
	_, ok, err = inspector.OnPath(context.Background(), nil, "chainbench-no-such-command")
	if err != nil {
		t.Fatalf("a missing command is an answer, not a failure: %v", err)
	}
	if ok {
		t.Error("a command that is not there must answer no")
	}
}

// TestOnPath_RemoteAsksTheTargetsShell: the answer has to be the target's PATH,
// not ours. A binary present here and absent there would otherwise pass the
// check and fail the launch.
func TestOnPath_RemoteAsksTheTargetsShell(t *testing.T) {
	var asked string
	run := func(_ context.Context, cmd string) (remote.ExecResult, error) {
		asked = cmd
		return remote.ExecResult{Stdout: "/data/chainbench/bin/gstable\n"}, nil
	}
	path, ok, err := inspector.OnPath(context.Background(), process.Runner(run), "gstable")
	if err != nil || !ok {
		t.Fatalf("OnPath = %q, %v, %v", path, ok, err)
	}
	if path != "/data/chainbench/bin/gstable" {
		t.Errorf("path = %q, want the target's answer", path)
	}
	if !strings.Contains(asked, "command -v") || !strings.Contains(asked, "gstable") {
		t.Errorf("asked %q, want a PATH lookup on the target", asked)
	}
}

// TestOnPath_RemoteNotFoundIsNoAndNotAnError: a non-zero exit means the command
// is not there, which is the question being asked. Only a broken connection is
// a failure.
func TestOnPath_RemoteNotFoundIsNoAndNotAnError(t *testing.T) {
	notFound := func(context.Context, string) (remote.ExecResult, error) {
		return remote.ExecResult{ExitCode: 1}, nil
	}
	if _, ok, err := inspector.OnPath(context.Background(), process.Runner(notFound), "nope"); err != nil || ok {
		t.Errorf("got %v, %v; want no and no error", ok, err)
	}

	broken := func(context.Context, string) (remote.ExecResult, error) {
		return remote.ExecResult{}, errors.New("connection refused")
	}
	if _, _, err := inspector.OnPath(context.Background(), process.Runner(broken), "nope"); err == nil {
		t.Error("an unreachable target must be an error, not a no")
	}
}

// TestOnPath_RefusesAPath: a path is [Paths]'s question. Answering it here
// would make two ways to ask one thing, and they would drift.
func TestOnPath_RefusesAPath(t *testing.T) {
	if _, _, err := inspector.OnPath(context.Background(), nil, "/usr/bin/sh"); err == nil {
		t.Error("a path must be refused")
	}
}
