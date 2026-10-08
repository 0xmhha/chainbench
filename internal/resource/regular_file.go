package resource

import (
	"context"
	"errors"
	"os"

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
