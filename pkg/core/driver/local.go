package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ExecFn constructs the command a LocalDriver launches. It is injectable so
// tests can substitute a controllable process without a real node binary
// (mirrors network/internal/drivers/local).
type ExecFn func(ctx context.Context, name string, arg ...string) *exec.Cmd

// LocalDriver launches nodes as subprocesses on the local host.
type LocalDriver struct {
	execFn ExecFn
}

// NewLocalDriver returns a LocalDriver using os/exec.
func NewLocalDriver() *LocalDriver {
	return &LocalDriver{execFn: defaultExec}
}

// NewLocalDriverWithExec returns a LocalDriver using a custom exec factory (tests).
func NewLocalDriverWithExec(fn ExecFn) *LocalDriver {
	return &LocalDriver{execFn: fn}
}

func defaultExec(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

// Provision creates the data dir and writes the node config file if provided.
func (d *LocalDriver) Provision(_ context.Context, spec NodeSpec) error {
	if err := os.MkdirAll(spec.DataDir, 0o755); err != nil {
		return fmt.Errorf("driver: mkdir datadir: %w", err)
	}
	if spec.ConfigPath != "" && len(spec.ConfigContent) > 0 {
		if err := os.MkdirAll(filepath.Dir(spec.ConfigPath), 0o755); err != nil {
			return fmt.Errorf("driver: mkdir config dir: %w", err)
		}
		if err := os.WriteFile(spec.ConfigPath, spec.ConfigContent, 0o644); err != nil {
			return fmt.Errorf("driver: write config: %w", err)
		}
	}
	return nil
}

// Launch starts the node process, redirecting stdout/stderr to the log file,
// and returns its Handle. The process is not waited on (nodes run
// indefinitely); a reaper goroutine closes the log file when it exits.
func (d *LocalDriver) Launch(ctx context.Context, spec NodeSpec) (Handle, error) {
	if err := os.MkdirAll(filepath.Dir(spec.LogPath), 0o755); err != nil {
		return Handle{}, fmt.Errorf("driver: mkdir log dir: %w", err)
	}
	logf, err := os.OpenFile(spec.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Handle{}, fmt.Errorf("driver: open log: %w", err)
	}
	cmd := d.execFn(ctx, spec.Binary, spec.Args...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		_ = logf.Close()
		return Handle{}, fmt.Errorf("driver: start node%d: %w", spec.Index, err)
	}
	pid := cmd.Process.Pid
	go func() {
		_ = cmd.Wait()
		_ = logf.Close()
	}()
	return Handle{Index: spec.Index, PID: pid}, nil
}

// Stop sends SIGTERM (via Process) to the handle's PID. A process that has
// already exited yields no error.
func (d *LocalDriver) Stop(_ context.Context, h Handle) error {
	proc, err := os.FindProcess(h.PID)
	if err != nil {
		return fmt.Errorf("driver: find process %d: %w", h.PID, err)
	}
	if err := proc.Kill(); err != nil && !isFinished(err) {
		return fmt.Errorf("driver: stop node%d (pid %d): %w", h.Index, h.PID, err)
	}
	return nil
}

func isFinished(err error) bool {
	return err != nil && err.Error() == "os: process already finished"
}
