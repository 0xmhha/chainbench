package local

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Driver executes chainbench CLI subcommands as subprocesses.
type Driver struct {
	chainbenchDir string
	exec          func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewDriver creates a Driver that invokes chainbench.sh under chainbenchDir
// using the stdlib exec.CommandContext.
func NewDriver(chainbenchDir string) *Driver {
	return NewDriverWithExec(chainbenchDir, exec.CommandContext)
}

// NewDriverWithExec is the testable constructor allowing the exec factory
// to be replaced with a fake.
func NewDriverWithExec(chainbenchDir string,
	execFn func(ctx context.Context, name string, args ...string) *exec.Cmd) *Driver {
	return &Driver{chainbenchDir: chainbenchDir, exec: execFn}
}

// RunResult captures a single subprocess invocation outcome.
type RunResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Run executes `<chainbenchDir>/chainbench.sh <args...>`. Each stdout line
// is logged via slog.Info; each stderr line via slog.Warn. The full text
// is also captured in the returned RunResult.Stdout / Stderr.
//
// A non-zero exit code does NOT produce a returned error — callers inspect
// RunResult.ExitCode. Start / IO / context errors DO return non-nil.
func (d *Driver) Run(ctx context.Context, args ...string) (*RunResult, error) {
	script := filepath.Join(d.chainbenchDir, "chainbench.sh")
	cmd := d.exec(ctx, script, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("local: stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("local: stderr pipe: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("local: start %s: %w", script, err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go streamAndCapture(&wg, stdoutPipe, &stdoutBuf, slog.LevelInfo, "subprocess stdout")
	go streamAndCapture(&wg, stderrPipe, &stderrBuf, slog.LevelWarn, "subprocess stderr")
	wg.Wait()

	waitErr := cmd.Wait()
	duration := time.Since(start)

	exitCode := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return nil, fmt.Errorf("local: wait: %w", waitErr)
		}
	}

	return &RunResult{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}, nil
}

// streamAndCapture reads r line-by-line, logs each line at level with msg,
// and appends to buf. Signals done on wg.
func streamAndCapture(wg *sync.WaitGroup, r io.Reader, buf *bytes.Buffer, level slog.Level, msg string) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	// Allow larger lines than default (64KB) — node logs can be verbose.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteByte('\n')
		slog.Log(context.Background(), level, msg, "line", line)
	}
}
