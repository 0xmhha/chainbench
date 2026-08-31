package process

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
	// execFn builds short-lived commands, which the caller waits for. Their
	// context bounds them, and cancelling it should stop them.
	execFn ExecFn
	// launchFn builds the node process, which is the opposite case: a node
	// outlives the call that starts it. The caller owns it from then on,
	// through Stop and procman, and the workspace records its pid so a later
	// invocation can stop it.
	//
	// The two are separate because binding a node to the request context ends
	// the node when the request ends. That was invisible while the root
	// context was never cancelled; the moment an interrupt could cancel it,
	// `net up` returned "4 node(s) started" and left three running.
	launchFn ExecFn
}

// NewLocalDriver returns a LocalDriver using os/exec.
func NewLocalDriver() *LocalDriver {
	return &LocalDriver{execFn: defaultExec, launchFn: detachedExec}
}

// NewLocalDriverWithExec returns a LocalDriver using a custom exec factory
// (tests). The one factory serves both roles, so a test keeps full control of
// every process the driver starts.
func NewLocalDriverWithExec(fn ExecFn) *LocalDriver {
	return &LocalDriver{execFn: fn, launchFn: fn}
}

func defaultExec(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

// detachedExec builds a command whose lifetime is not the caller's. The context
// still governs how long the caller waits to start it — Start returns at once —
// but not how long the process lives.
func detachedExec(_ context.Context, name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
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
	cmd := d.launch(ctx, spec.Binary, spec.Args...)
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

// launch builds the node command, falling back to the short-lived factory when
// a caller constructed the driver with a zero value.
func (d *LocalDriver) launch(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if d.launchFn != nil {
		return d.launchFn(ctx, name, arg...)
	}
	return detachedExec(ctx, name, arg...)
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
