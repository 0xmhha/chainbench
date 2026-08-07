package driver

import (
	"context"
	"fmt"
	"io/fs"
)

// RemoteFileSink materializes files on a remote host over SSH. It satisfies the
// provision.FileSink contract (Exists + Write) structurally, so a launcher's
// FileSink seam can ship genesis and config to another host with no code change:
// writes go through a FileProvisioner, and existence is a `test -f` probe (so
// upload-if-absent reuses files already present on the host).
type RemoteFileSink struct {
	// Run executes a shell command on the remote host.
	Run Runner
	// Files writes a file on the remote host.
	Files FileProvisioner
}

// NewRemoteFileSink returns a RemoteFileSink backed by run for both the
// existence probe and (via a RemoteDriver) the file writes.
func NewRemoteFileSink(run Runner) RemoteFileSink {
	return RemoteFileSink{Run: run, Files: NewRemoteDriver(run)}
}

// Exists reports whether remotePath is present on the host. A `test -f` exit of
// 0 means present, non-zero means absent; only a transport failure is an error.
func (s RemoteFileSink) Exists(ctx context.Context, remotePath string) (bool, error) {
	res, err := s.Run(ctx, "test -f "+shq(remotePath))
	if err != nil {
		return false, fmt.Errorf("driver: remote exists %s: %w", remotePath, err)
	}
	return res.ExitCode == 0, nil
}

// Write ships content to remotePath with the given mode.
func (s RemoteFileSink) Write(ctx context.Context, remotePath string, content []byte, mode fs.FileMode) error {
	return s.Files.ProvisionFile(ctx, remotePath, content, mode)
}
