// Package webcli runs the chainbench CLI on behalf of the Web dashboard.
//
// The Web does not carry an execution engine of its own: what an operator can
// do from a terminal, the dashboard does by running the same command, and it
// reads back what that command printed or recorded. This package is the one
// place that starts those processes, so cancellation, error reporting and the
// choice of binary are decided once.
package webcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// unwindGrace is how long a cancelled command gets to unwind after its
// interrupt, the way an operator's first Ctrl-C lets it record what it started.
const unwindGrace = 30 * time.Second

// Runner runs one chainbench binary.
type Runner struct {
	// Binary is the chainbench executable.
	Binary string
	// Env is added to the dashboard's own environment for every command.
	Env []string
}

// Run executes chainbench with args and returns what it printed on stdout. A
// failed command is reported with the message it printed on stderr. Cancelling
// ctx interrupts the command and waits for it to unwind.
func (r Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.Binary, args...)
	cmd.Env = append(os.Environ(), r.Env...)
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGINT) }
	cmd.WaitDelay = unwindGrace
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("chainbench %s: %s", commandName(args), reason(stderr.String(), err))
	}
	return stdout.Bytes(), nil
}

// ChainStatus returns the composition state `chainbench chain status` reports
// for the workspace at dir.
func (r Runner) ChainStatus(ctx context.Context, dir string) (chainsetup.State, error) {
	out, err := r.Run(ctx, "chain", "status", "--json", "--workspace-dir", dir)
	if err != nil {
		return chainsetup.State{}, err
	}
	var st chainsetup.State
	if err = json.Unmarshal(out, &st); err != nil {
		return chainsetup.State{}, fmt.Errorf("chainbench chain status: unreadable output: %w", err)
	}
	return st, nil
}

// FindBinary finds the chainbench binary: the explicit path when one is given,
// otherwise the one installed beside the dashboard executable, otherwise the
// one on PATH. A dashboard without it cannot run anything, so the caller
// refuses to start rather than fail at the first job.
func FindBinary(explicit, dashboard string) (string, error) {
	if explicit != "" {
		if !executable(explicit) {
			return "", fmt.Errorf("webcli: %s is not an executable chainbench binary", explicit)
		}
		return explicit, nil
	}
	if dashboard != "" {
		if beside := filepath.Join(filepath.Dir(dashboard), "chainbench"); executable(beside) {
			return beside, nil
		}
	}
	if found, err := exec.LookPath("chainbench"); err == nil {
		return found, nil
	}
	return "", errors.New("webcli: no chainbench binary beside the dashboard or on PATH; build it or pass its path")
}

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

// commandName is the subcommand path an error names, without its flags.
func commandName(args []string) string {
	var words []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		words = append(words, a)
	}
	return strings.Join(words, " ")
}

// reason is the CLI's own last error line, or the process error when it
// printed none.
func reason(stderr string, err error) string {
	for _, line := range slices.Backward(strings.Split(strings.TrimSpace(stderr), "\n")) {
		if line = strings.TrimSpace(line); line != "" {
			return strings.TrimPrefix(line, "error: ")
		}
	}
	return err.Error()
}
