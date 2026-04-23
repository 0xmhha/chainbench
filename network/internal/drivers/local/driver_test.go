package local

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// fakeExecFn returns an exec factory that spawns `sh -c "<script>"` to
// reliably control stdout/stderr/exit code in tests without needing
// OS-specific binaries.
func fakeExecFn(script string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Ignore real name/args; run the script via sh -c so tests control
		// stdout/stderr/exit precisely. Capture the real args on the wrapped
		// cmd via Env for assertion if needed.
		full := append([]string{name}, args...)
		c := exec.CommandContext(ctx, "sh", "-c", script)
		c.Env = append(c.Env, "FAKE_ARGS="+strings.Join(full, "|"))
		return c
	}
}

func TestDriver_Run_Success(t *testing.T) {
	d := NewDriverWithExec("/nowhere", fakeExecFn(`echo "hello"; exit 0`))
	res, err := d.Run(context.Background(), "anything")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit: got %d, want 0", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("stdout: got %q", res.Stdout)
	}
	if res.Duration <= 0 {
		t.Errorf("duration: got %v, want > 0", res.Duration)
	}
}

func TestDriver_Run_NonZeroExit_NoError(t *testing.T) {
	d := NewDriverWithExec("/nowhere", fakeExecFn(`echo "boom" >&2; exit 3`))
	res, err := d.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("run returned error for non-zero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("exit: got %d, want 3", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "boom") {
		t.Errorf("stderr: got %q", res.Stderr)
	}
}

func TestDriver_Run_ContextCanceled_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before exec
	d := NewDriverWithExec("/nowhere", fakeExecFn(`sleep 5; exit 0`))
	_, err := d.Run(ctx, "x")
	if err == nil {
		t.Fatal("expected error for canceled ctx")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("err: got %v, want canceled-related", err)
	}
}

func TestDriver_Run_StreamsLinesToSlog(t *testing.T) {
	// Capture slog output by redirecting the default logger to a buffer.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	d := NewDriverWithExec("/nowhere", fakeExecFn(`echo "line1"; echo "line2" >&2; exit 0`))
	if _, err := d.Run(context.Background(), "x"); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "line1") {
		t.Errorf("slog missing stdout line: %q", out)
	}
	if !strings.Contains(out, "line2") {
		t.Errorf("slog missing stderr line: %q", out)
	}
}

// Ensure NewDriver (prod constructor) compiles and wires exec.CommandContext.
func TestNewDriver_ProdConstructor(t *testing.T) {
	d := NewDriver("/nowhere")
	if d == nil {
		t.Fatal("nil driver")
	}
	// Calling Run with a missing chainbench.sh should return an error with
	// no usable exec, but NOT panic. We use a short-lived ctx so it doesn't
	// hang if somehow things work.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := d.Run(ctx, "help")
	if err == nil {
		t.Fatal("expected error for missing chainbench.sh at /nowhere")
	}
}
