package process

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// SSHFileProvisioner sends bytes on SSH stdin, keeping private identity content
// and large binaries out of remote shell arguments. Files become visible only
// after the complete upload succeeds; cancellation may leave a temporary file
// if the remote shell cannot finish its trap, so callers still track effects.
type SSHFileProvisioner struct {
	Credentials remote.Credentials
	HostKey     remote.HostKeyCallback
}

func (p SSHFileProvisioner) ProvisionFile(ctx context.Context, remotePath string, content []byte, mode fs.FileMode) error {
	command := "set -eu; umask 077; mkdir -p " + remote.ShellQuote(path.Dir(remotePath)) +
		"; tmp=$(mktemp " + remote.ShellQuote(remotePath+".upload.XXXXXX") + "); trap 'rm -f \"$tmp\"' EXIT HUP INT TERM; cat > \"$tmp\"; chmod " + fmt.Sprintf("%o", mode.Perm()) +
		" \"$tmp\"; mv -f \"$tmp\" " + remote.ShellQuote(remotePath)
	result, err := remote.ExecWithInput(ctx, p.Credentials, p.HostKey, "/bin/sh -c "+remote.ShellQuote(command), string(content))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return uploadFailure(remotePath, result)
	}
	return nil
}

// maxUploadReason bounds how much of the server's stderr an upload failure
// carries; the last lines are the ones that name the cause.
const maxUploadReason = 1024

// uploadFailure says why the server refused an upload. The command writes no
// file content to stderr, so its text is safe to show.
func uploadFailure(remotePath string, result remote.ExecResult) error {
	reason := strings.TrimSpace(result.Stderr)
	if reason == "" {
		return fmt.Errorf("driver: SSH file upload %s failed (exit %d)", remotePath, result.ExitCode)
	}
	if len(reason) > maxUploadReason {
		reason = reason[len(reason)-maxUploadReason:]
	}
	return fmt.Errorf("driver: SSH file upload %s failed (exit %d): %s", remotePath, result.ExitCode, reason)
}
