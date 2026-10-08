package resource

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// VerifyRegularFile refuses a deployed file alias even when its target has
// matching bytes. Parent identity must be checked separately by the caller.
func (a *Access) VerifyRegularFile(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil {
		return errors.New("resource: target file access unavailable")
	}
	if a.Runner == nil {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("resource: target is not a regular file")
		}
		return nil
	}
	command := "test -f " + remote.ShellQuote(path) + " && test ! -L " + remote.ShellQuote(path)
	result, err := a.Runner(ctx, "/bin/sh -c "+remote.ShellQuote(command))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return errors.New("resource: target is not a regular file")
	}
	return nil
}

// VerifyExecutable checks the target account's effective execute permission.
// It does not repair file permissions or follow a symbolic file alias.
func (a *Access) VerifyExecutable(ctx context.Context, path string) error {
	if err := a.VerifyRegularFile(ctx, path); err != nil {
		return err
	}
	command := "test -x " + remote.ShellQuote(path)
	if a.Runner == nil {
		if err := exec.CommandContext(ctx, "/bin/sh", "-c", command).Run(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("resource: target file is not executable")
		}
		return nil
	}
	result, err := a.Runner(ctx, "/bin/sh -c "+remote.ShellQuote(command))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return errors.New("resource: target file is not executable")
	}
	return nil
}
