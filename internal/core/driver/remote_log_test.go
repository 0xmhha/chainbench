package driver_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

func TestRemoteLogReader_TailsFromAByteOffset(t *testing.T) {
	var got string
	r := driver.NewRemoteLogReader(func(_ context.Context, cmd string) (remote.ExecResult, error) {
		got = cmd
		return remote.ExecResult{Stdout: "line\n"}, nil
	})
	out, err := r.ReadFrom(context.Background(), "/data/node1.log", 42)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if string(out) != "line\n" {
		t.Fatalf("out = %q", out)
	}
	// tail -c is 1-based, so byte offset 42 is +43.
	if !strings.Contains(got, "tail -c +43") {
		t.Fatalf("command = %q, want a 1-based byte offset", got)
	}
	if !strings.Contains(got, "/data/node1.log") {
		t.Fatalf("command = %q, missing the path", got)
	}
}

func TestRemoteLogReader_OffsetZeroReadsFromTheStart(t *testing.T) {
	var got string
	r := driver.NewRemoteLogReader(func(_ context.Context, cmd string) (remote.ExecResult, error) {
		got = cmd
		return remote.ExecResult{}, nil
	})
	if _, err := r.ReadFrom(context.Background(), "/l", 0); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !strings.Contains(got, "tail -c +1") {
		t.Fatalf("command = %q", got)
	}
}

func TestRemoteLogReader_MissingFileIsNotAnError(t *testing.T) {
	r := driver.NewRemoteLogReader(func(context.Context, string) (remote.ExecResult, error) {
		return remote.ExecResult{ExitCode: 1}, nil
	})
	out, err := r.ReadFrom(context.Background(), "/nope", 0)
	if err != nil {
		t.Fatalf("a log that does not exist yet must not be an error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("out = %q, want empty", out)
	}
}

func TestRemoteLogReader_TransportFailureIsAnError(t *testing.T) {
	r := driver.NewRemoteLogReader(func(context.Context, string) (remote.ExecResult, error) {
		return remote.ExecResult{}, errors.New("ssh: connection lost")
	})
	if _, err := r.ReadFrom(context.Background(), "/l", 0); err == nil {
		t.Fatal("expected an error when the transport fails")
	}
}

func TestRemoteLogReader_WithoutARunnerIsAnError(t *testing.T) {
	var r driver.RemoteLogReader
	if _, err := r.ReadFrom(context.Background(), "/l", 0); err == nil {
		t.Fatal("expected an error with no runner wired")
	}
}
